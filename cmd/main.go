package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"

	"github.com/imedgar/rain-alert/internal/alert"
	"github.com/imedgar/rain-alert/internal/config"
	"github.com/imedgar/rain-alert/internal/platform/database"
	"github.com/imedgar/rain-alert/internal/platform/ntfy"
	"github.com/imedgar/rain-alert/internal/weather"

	_ "github.com/tursodatabase/libsql-client-go/libsql"
)

func main() {
	if err := run(); err != nil {
		log.Fatal("oh no ", err)
	}
}

func run() error {
	ctx := context.Background()

	c, err := config.NewConfig(ctx)
	if err != nil {
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

	dbPlatform := database.New(db)
	weatherAPI := weather.NewAPI(http.DefaultClient, "http://api.weatherapi.com/v1/forecast.json", c.WeatherApiKey)
	ntfyClient := ntfy.New(http.DefaultClient, "https://ntfy.sh", c.PushNotificationTopic)

	alerter := alert.NewAlerter(weatherAPI, dbPlatform, ntfyClient)

	if err := alerter.CheckAndAlert(c.Location, c.Timezone); err != nil {
		return err
	}

	return nil
}
