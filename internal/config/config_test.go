package config

import (
	"context"
	"os"
	"testing"
)

func TestNewConfig(t *testing.T) {
	t.Run("Successful config loading", func(t *testing.T) {
		// Set environment variables for the test
		os.Setenv("WEATHER_API_KEY", "test_api_key")
		os.Setenv("PUSH_NOTIFICATION_TOPIC", "test_topic")
		os.Setenv("DB_URL", "test_db_url")
		os.Setenv("DB_TOKEN", "test_db_token")
		os.Setenv("LOCATION", "test_location")
		os.Setenv("TIMEZONE", "test_timezone")

		// Unset environment variables after the test
		defer func() {
			os.Unsetenv("WEATHER_API_KEY")
			os.Unsetenv("PUSH_NOTIFICATION_TOPIC")
			os.Unsetenv("DB_URL")
			os.Unsetenv("DB_TOKEN")
			os.Unsetenv("LOCATION")
			os.Unsetenv("TIMEZONE")
		}()

		cfg, err := NewConfig(context.Background())
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if cfg.WeatherApiKey != "test_api_key" {
			t.Errorf("expected WeatherApiKey to be 'test_api_key', got '%s'", cfg.WeatherApiKey)
		}
		if cfg.PushNotificationTopic != "test_topic" {
			t.Errorf("expected PushNotificationTopic to be 'test_topic', got '%s'", cfg.PushNotificationTopic)
		}
		if cfg.DatabaseUrl != "test_db_url" {
			t.Errorf("expected DatabaseUrl to be 'test_db_url', got '%s'", cfg.DatabaseUrl)
		}
		if cfg.DatabaseToken != "test_db_token" {
			t.Errorf("expected DatabaseToken to be 'test_db_token', got '%s'", cfg.DatabaseToken)
		}
		if cfg.Location != "test_location" {
			t.Errorf("expected Location to be 'test_location', got '%s'", cfg.Location)
		}
		if cfg.Timezone != "test_timezone" {
			t.Errorf("expected Timezone to be 'test_timezone', got '%s'", cfg.Timezone)
		}
	})

	t.Run("Missing environment variable", func(t *testing.T) {
		// Unset all required environment variables
		os.Unsetenv("WEATHER_API_KEY")
		os.Unsetenv("PUSH_NOTIFICATION_TOPIC")
		os.Unsetenv("DB_URL")
		os.Unsetenv("DB_TOKEN")
		os.Unsetenv("LOCATION")
		os.Unsetenv("TIMEZONE")

		_, err := NewConfig(context.Background())
		if err == nil {
			t.Error("expected an error, but got nil")
		}
	})
}
