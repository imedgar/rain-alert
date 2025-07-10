
package weather

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"testing"
	"time"
)

type MockClient struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

func (m *MockClient) Do(req *http.Request) (*http.Response, error) {
	return m.DoFunc(req)
}

func NewMockClient(statusCode int, body string) *MockClient {
	return &MockClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: statusCode,
				Body:       io.NopCloser(bytes.NewReader([]byte(body))),
			}, nil
		},
	}
}

func TestGetNextHourForecast(t *testing.T) {
	t.Run("Successful forecast retrieval", func(t *testing.T) {
		weatherResponse := &WeatherResponse{
			Forecast: struct {
				ForecastDay []struct {
					Date string `json:"date"`
					Hour []Hour `json:"hour"`
				} `json:"forecastday"`
			}{
				ForecastDay: []struct {
					Date string `json:"date"`
					Hour []Hour `json:"hour"`
				}{
					{
						Date: "2025-07-10",
						Hour: make([]Hour, 24),
					},
				},
			},
		}
		nextHour := (time.Now().In(time.UTC).Hour() + 1) % 24
		weatherResponse.Forecast.ForecastDay[0].Hour[nextHour] = Hour{ChanceOfRain: 80}

		weatherBody, _ := json.Marshal(weatherResponse)
		mockClient := NewMockClient(http.StatusOK, string(weatherBody))

		api := NewAPI(mockClient, "http://test.com", "test-key")

		_, hour, err := api.GetNextHourForecast("Test Location", "UTC")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if hour.ChanceOfRain != 80 {
			t.Errorf("expected chance of rain to be 80, got %d", hour.ChanceOfRain)
		}
	})

	t.Run("Weather API error", func(t *testing.T) {
		mockClient := &MockClient{
			DoFunc: func(req *http.Request) (*http.Response, error) {
				return nil, errors.New("API error")
			},
		}
		api := NewAPI(mockClient, "http://test.com", "test-key")

		_, _, err := api.GetNextHourForecast("Test Location", "UTC")
		if err == nil {
			t.Error("expected an error, but got nil")
		}
	})
}

func TestMain(m *testing.M) {
	// Set a fixed time for tests
	time.Local = time.UTC
	exitVal := m.Run()
	os.Exit(exitVal)
}
