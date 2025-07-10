package weather

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
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

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type API struct {
	HttpClient HTTPClient
	URL        string
	ApiKey     string
}

func NewAPI(client HTTPClient, url, apiKey string) *API {
	return &API{HttpClient: client, URL: url, ApiKey: apiKey}
}

const (
	checkAheadHours = 1
	userAgent       = "rain-alert/1.0"
)

func (a *API) GetNextHourForecast(location, timezone string) (*WeatherResponse, *Hour, error) {
	weather, err := a.fetchWeather(location)
	if err != nil {
		return nil, nil, err
	}

	hour, err := a.getNextHourForecast(weather, timezone)
	if err != nil {
		return nil, nil, err
	}

	return weather, hour, nil
}

func (a *API) fetchWeather(location string) (*WeatherResponse, error) {
	params := url.Values{}
	params.Set("key", a.ApiKey)
	params.Set("q", location)
	params.Set("days", "1")
	params.Set("aqi", "no")
	params.Set("alerts", "no")

	fullURL := fmt.Sprintf("%s?%s", a.URL, params.Encode())
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := a.HttpClient.Do(req)
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

func (a *API) getNextHourForecast(weather *WeatherResponse, timezone string) (*Hour, error) {
	if len(weather.Forecast.ForecastDay) == 0 {
		return nil, fmt.Errorf("no forecast days found")
	}

	hours := weather.Forecast.ForecastDay[0].Hour
	if len(hours) < 24 {
		return nil, fmt.Errorf("hourly forecast incomplete")
	}

	tz, err := time.LoadLocation(timezone)
	if err != nil {
		return nil, fmt.Errorf("invalid timezone: %w", err)
	}
	nowIn := time.Now().In(tz)

	nextHour := (nowIn.Hour() + checkAheadHours) % 24 // if 23 check 0, hours starts at 00
	return &weather.Forecast.ForecastDay[0].Hour[nextHour], nil
}
