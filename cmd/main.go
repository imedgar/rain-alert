package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"

	"github.com/imedgar/rain-alert/internal/weather"
	"github.com/sethvargo/go-envconfig"
)

type WeatherConfig struct {
	Config *Config
}

type Config struct {
	WeatherApiKey         string `env:"WEATHER_API_KEY"`
	PushNotificationTopic string `env:"PUSH_NOTIFICATION_TOPIC"`
	DatabaseUrl           string `env:"DB_URL"`
	DatabaseToken         string `env:"DB_TOKEN"`
	Location              string `env:"LOCATION"`
	Timezone              string `env:"TIMEZONE"`
}

func main() {
	if err := run(); err != nil {
		log.Fatal("oh no ", err)
	}
}

func run() error {
	ctx := context.Background()

	var c Config
	if err := envconfig.Process(ctx, &c); err != nil {
		return err
	}

	db, err := sql.Open("libsql", fmt.Sprintf("%s?authToken=%s", c.DatabaseUrl, c.DatabaseToken))
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		return fmt.Errorf("pinging database: %w", err)
	}

	api := weather.NewWeatherAPI(weather.Config{
		WeatherApiKey:         c.WeatherApiKey,
		PushNotificationTopic: c.PushNotificationTopic,
		Location:              c.Location,
		Timezone:              c.Timezone,
	}, db, http.DefaultClient)

	err = api.GetNextHourForecast()
	if err != nil {
		return err
	}

	return nil
}
