package alert

import (
	"fmt"
	"log"

	"github.com/imedgar/rain-alert/internal/platform/database"
	"github.com/imedgar/rain-alert/internal/platform/ntfy"
	"github.com/imedgar/rain-alert/internal/weather"
)

type Alerter struct {
	Weather *weather.API
	DB      *database.DB
	Ntfy    *ntfy.Client
}

func NewAlerter(weather *weather.API, db *database.DB, ntfy *ntfy.Client) *Alerter {
	return &Alerter{Weather: weather, DB: db, Ntfy: ntfy}
}

func (a *Alerter) CheckAndAlert(location, timezone string) error {
	weatherData, hour, err := a.Weather.GetNextHourForecast(location, timezone)
	if err != nil {
		return fmt.Errorf("getting forecast: %w", err)
	}

	thresholds, err := a.DB.GetThresholds()
	if err != nil {
		return fmt.Errorf("getting thresholds: %w", err)
	}

	if hour.ChanceOfRain < thresholds["drizzleThreshold"] {
		log.Printf("Chance of rain (%d%%) too low, not notifying.\n", hour.ChanceOfRain)
		return nil
	}

	notify, err := a.DB.ShouldNotify(thresholds)
	if err != nil {
		return fmt.Errorf("checking notification history: %w", err)
	}

	if !notify {
		log.Println("Recent rain detected, skipping notification.")
		return nil
	}

	msg := a.Ntfy.GenerateRainMessage(weatherData.Location.Name, hour.Time, hour.PrecipMM, hour.ChanceOfRain)
	if err := a.Ntfy.Send("Rain Alert", msg, "umbrella,robot"); err != nil {
		return fmt.Errorf("sending notification: %w", err)
	}

	if err := a.DB.RecordNotification(hour.ChanceOfRain); err != nil {
		return fmt.Errorf("recording notification: %w", err)
	}

	return nil
}
