package ntfy

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"golang.org/x/exp/rand"
)

type Client struct {
	HttpClient HTTPClient
	URL        string
	Topic      string
}

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func New(client HTTPClient, url, topic string) *Client {
	return &Client{HttpClient: client, URL: url, Topic: topic}
}

func (c *Client) Send(title, message, tags string) error {
	url := fmt.Sprintf("%s/%s", c.URL, c.Topic)
	req, err := http.NewRequest("POST", url, strings.NewReader(message))
	if err != nil {
		return fmt.Errorf("creating notification request: %w", err)
	}
	req.Header.Set("Title", title)
	req.Header.Set("Tags", tags)

	resp, err := c.HttpClient.Do(req)
	if err != nil {
		return fmt.Errorf("sending notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("notification failed: %s", resp.Status)
	}

	log.Printf("Notification sent: %s", message)
	return nil
}

func (c *Client) GenerateRainMessage(location, timeStr string, precipMM float64, chanceOfRain int) string {
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
	return fmt.Sprintf(template, location, timeStr, precipMM, chanceOfRain)
}
