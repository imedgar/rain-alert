package weather

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	_ "github.com/tursodatabase/libsql-client-go/libsql"
	"golang.org/x/exp/rand"
)

type WeatherResponse struct {
	Location struct {
		Name      string `json:"name"`
		TzID      string `json:"tz_id"`
		Localtime string `json:"localtime"`
	} `json:"location"`

	Forecast struct {
		ForecastDay []struct {
			Date string `json:"date"`
			Hour []Hour `json:"hour"`
		} `json:"forecastday"`
	} `json:"forecast"`
}

type Hour struct {
	Time         string  `json:"time"`
	PrecipMM     float64 `json:"precip_mm"`
	WillItRain   int     `json:"will_it_rain"`
	ChanceOfRain int     `json:"chance_of_rain"`
}

type Config struct {
	WeatherApiKey         string
	PushNotificationTopic string
	DatabaseUrl           string
	DatabaseToken         string
	Location              string
	Timezone              string
}

type WeatherAPI struct {
	Config Config
}

func NewWeatherAPI(config Config) *WeatherAPI {
	return &WeatherAPI{config}
}

const (
	apiURL              = "http://api.weatherapi.com/v1/forecast.json"
	rainBeforeThreshold = 70
	drizzleThreshold    = 50
	checkAheadHours     = 1
	userAgent           = "rain-alert/1.0"
	ntfyBaseURL         = "https://ntfy.sh"
	dateLayout          = "2006-01-02 15:04"
)

func (api *WeatherAPI) GetNextHourForecast() error {
	weather, err := api.fetchWeather()
	if err != nil {
		return err
	}

	hour, err := api.getNextHourForecast(weather)
	if err != nil {
		return err
	}

	if hour.ChanceOfRain < drizzleThreshold {
		log.Printf("Chance of rain (%d%%) too low, not notifying.\n", hour.ChanceOfRain)
		return nil
	}

	db, err := sql.Open("libsql", fmt.Sprintf("%s?authToken=%s", api.Config.DatabaseUrl, api.Config.DatabaseToken))
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		return fmt.Errorf("pinging database: %w", err)
	}

	notify, err := api.shouldNotify(db)
	if err != nil {
		return err
	}
	if !notify {
		log.Println("Recent rain detected, skipping notification.")
		return nil
	}

	if err := api.sendNotification(hour, weather.Location.Name); err != nil {
		return err
	}

	if err := api.recordNotification(db, hour.ChanceOfRain); err != nil {
		return err
	}

	return nil
}

func (api *WeatherAPI) fetchWeather() (*WeatherResponse, error) {
	params := url.Values{}
	params.Set("key", api.Config.WeatherApiKey)
	params.Set("q", api.Config.Location)
	params.Set("days", "1")
	params.Set("aqi", "no")
	params.Set("alerts", "no")

	fullURL := fmt.Sprintf("%s?%s", apiURL, params.Encode())
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected response: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	var weather WeatherResponse
	if err := json.Unmarshal(body, &weather); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return &weather, nil
}

func (api *WeatherAPI) getNextHourForecast(weather *WeatherResponse) (Hour, error) {
	if len(weather.Forecast.ForecastDay) == 0 {
		return Hour{}, fmt.Errorf("no forecast days found")
	}

	hours := weather.Forecast.ForecastDay[0].Hour
	if len(hours) < 24 {
		return Hour{}, fmt.Errorf("hourly forecast incomplete")
	}

	tz, err := time.LoadLocation(api.Config.Timezone)
	if err != nil {
		return Hour{}, fmt.Errorf("invalid timezone: %w", err)
	}
	nowIn := time.Now().In(tz)

	nextHour := (nowIn.Hour() + checkAheadHours) % 24 // if 23 check 0, hours starts at 00
	return weather.Forecast.ForecastDay[0].Hour[nextHour], nil
}

func (api *WeatherAPI) shouldNotify(db *sql.DB) (bool, error) {
	var state int
	var createdAt int64

	row := db.QueryRow("SELECT state, created_at FROM weather_notifications ORDER BY id DESC LIMIT 1")
	err := row.Scan(&state, &createdAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return true, nil
		}
		return false, fmt.Errorf("querying last notification: %w", err)
	}

	age := time.Since(time.Unix(createdAt, 0))
	if age > time.Hour {
		log.Println("Last notification is older than 1 hour, ignoring previous state.")
		return true, nil
	}

	if state > rainBeforeThreshold {
		return false, nil
	}

	return true, nil
}

func (api *WeatherAPI) sendNotification(hour Hour, location string) error {
	parsedTime, err := time.Parse(dateLayout, hour.Time)
	if err == nil {
		log.Printf("Forecast for: %s", parsedTime.Format(time.RFC822))
	}

	botRainMessages := []string{
		"ALERT! LIQUID SKY ATTACK IMMINENT!\nRain detected in %s at %s!\nPrecipitation: %.2fmm\nChance of wetness: %d%%\nDeploy umbrellas or perish gloriously!",
		"OH NO! THE SKY IS LEAKING AGAIN!\nIn %s, at %s, the sky shall unleash %.2fmm of watery doom!\n%d%% chance of H2-OMG!",
		"RAAAAAIN! GLORIOUS RAIN!\nTime: %s\nLocation: %s\n%.2fmm of beautiful, inconvenient precipitation!\nChance: %d%%\nRejoice or run — your call!",
		"BETTER POLISH YOUR UMBRELLA, CHAMP!\nHeads up in %s at %s — %.2fmm of sky sweat incoming!\nRain chance: %d%%\nWaterproof your dignity!",
		"THIS IS NOT A DRILL! IT’S JUST RAIN!\nForecast for %s at %s:\n%.2fmm of sky juice!\nProbability: %d%%\nCry harder, clouds!",
		"☁️✖️ DRY MODE: OFF ✖️☁️\nSystem override: %s at %s is entering soak cycle.\nRainfall: %.2fmm\nPrecipitation probability: %d%%\nRun for cover, squishy human!",
		"MOISTURE INBOUND, SOLDIER!\nBattlefield: %s\nTime: %s\nA damp %.2fmm approaches with a %d%% chance!\nTactical ponchos recommended.",
		"UPLOADING... DRENCH PROTOCOL 💦\n%s at %s is about to get slippery!\n%.2fmm of rain\n%d%% chance\nInitiating boots-to-slosh conversion...",
		"ERROR 404: DRYNESS NOT FOUND.\nLocation: %s\nTime: %s\nMoisture level: %.2fmm\nProbability of getting soggy: %d%%\nRecommend: activating anti-damp systems.",
		"WEATHER SYSTEM WARNING: INCOMING WETNESS.\nTarget: %s\nETA: %s\nFluid quantity: %.2fmm\nSplash zone probability: %d%%\nConclusion: regret is waterproof.",
	}
	rand.Seed(uint64(time.Now().UnixNano()))
	template := botRainMessages[rand.Intn(len(botRainMessages))]
	msg := fmt.Sprintf(template, location, hour.Time, hour.PrecipMM, hour.ChanceOfRain)

	url := fmt.Sprintf("%s/%s", ntfyBaseURL, api.Config.PushNotificationTopic)
	req, err := http.NewRequest("POST", url, strings.NewReader(msg))
	if err != nil {
		return fmt.Errorf("creating notification request: %w", err)
	}
	req.Header.Set("Title", "Rain alert")
	req.Header.Set("Tags", "umbrella,robot")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("sending notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("notification failed: %s", resp.Status)
	}

	log.Printf("Notification sent: %s", msg)
	return nil
}

func (api *WeatherAPI) recordNotification(db *sql.DB, state int) error {
	_, err := db.Exec("INSERT INTO weather_notifications(state, created_at) VALUES (?, ?)", state, time.Now().Unix())
	if err != nil {
		return fmt.Errorf("inserting notification: %w", err)
	}
	return nil
}
