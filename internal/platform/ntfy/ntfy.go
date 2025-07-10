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
		"ALERT! Rain in %s at %s!\n%.2fmm expected.\nChance: %d%%\nGrab your umbrella or face the splash!",
		"SKY LEAK! %s, %s — %.2fmm incoming!\nWetness odds: %d%%",
		"RAIN TIME!\n%s, %s\n%.2fmm on the way.\nChance: %d%%\nRejoice or retreat!",
		"UMBRELLA ALERT!\n%s at %s\n%.2fmm forecasted.\nRain chance: %d%%",
		"NOT A DRILL!\nRain in %s at %s\n%.2fmm expected.\nChance: %d%%",
		"☁️ WET MODE ACTIVATED ☁️\n%s, %s\nRain: %.2fmm\nChance: %d%%",
		"MOISTURE INCOMING!\n%s, %s\n%.2fmm with %d%% chance\nGet poncho-ready!",
		"DRENCH MODE: ON 💦\n%s, %s\n%.2fmm rain\n%d%% chance",
		"DRYNESS ERROR!\n%s, %s\n%.2fmm of sogginess\nOdds: %d%%",
		"⚠️ RAIN WARNING ⚠️\n%s, %s\n%.2fmm\nChance: %d%%\nStay dry or embrace the drip.",
	}
	rand.Seed(uint64(time.Now().UnixNano()))
	template := botRainMessages[rand.Intn(len(botRainMessages))]
	return fmt.Sprintf(template, location, timeStr, precipMM, chanceOfRain)
}
