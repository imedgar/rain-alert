
package weather

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
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
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	mockConfig := Config{
		WeatherApiKey:         "test-key",
		PushNotificationTopic: "test-topic",
		Location:              "Test Location",
		Timezone:              "UTC",
	}

	t.Run("Successful notification", func(t *testing.T) {
		weatherResponse := &WeatherResponse{
			Location: struct {
				Name      string `json:"name"`
				TzID      string `json:"tz_id"`
				Localtime string `json:"localtime"`
			}{Name: "Test Location"},
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
		// Set the chance of rain for the next hour to be above the threshold
		nextHour := (time.Now().In(time.UTC).Hour() + 1) % 24
		weatherResponse.Forecast.ForecastDay[0].Hour[nextHour] = Hour{ChanceOfRain: 80}

		weatherBody, _ := json.Marshal(weatherResponse)
		mockClient := NewMockClient(http.StatusOK, string(weatherBody))

		api := NewWeatherAPI(mockConfig, db, mockClient)

		// Mock database interactions
		rows := sqlmock.NewRows([]string{"config", "value"}).
			AddRow("drizzleThreshold", "50").
			AddRow("rainBeforeThreshold", "70")
		mock.ExpectQuery("SELECT config, value FROM weather_config").WillReturnRows(rows)
		mock.ExpectQuery("SELECT state, created_at FROM weather_notifications").WillReturnError(sql.ErrNoRows)
		mock.ExpectExec("INSERT INTO weather_notifications").
			WithArgs(80, sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		// Mock the ntfy call
		mockClient.DoFunc = func(req *http.Request) (*http.Response, error) {
			if req.URL.Host == "ntfy.sh" {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader([]byte(""))),
				}, nil
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(weatherBody)),
			}, nil
		}

		err := api.GetNextHourForecast()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})

	t.Run("Chance of rain too low", func(t *testing.T) {
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
		// Set the chance of rain for the next hour to be below the threshold
		nextHour := (time.Now().In(time.UTC).Hour() + 1) % 24
		weatherResponse.Forecast.ForecastDay[0].Hour[nextHour] = Hour{ChanceOfRain: 30}

		weatherBody, _ := json.Marshal(weatherResponse)
		mockClient := NewMockClient(http.StatusOK, string(weatherBody))

		api := NewWeatherAPI(mockConfig, db, mockClient)

		rows := sqlmock.NewRows([]string{"config", "value"}).AddRow("drizzleThreshold", "50")
		mock.ExpectQuery("SELECT config, value FROM weather_config").WillReturnRows(rows)

		err := api.GetNextHourForecast()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})

	t.Run("Recent notification sent", func(t *testing.T) {
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

		api := NewWeatherAPI(mockConfig, db, mockClient)

		configRows := sqlmock.NewRows([]string{"config", "value"}).
			AddRow("drizzleThreshold", "50").
			AddRow("rainBeforeThreshold", "70")
		mock.ExpectQuery("SELECT config, value FROM weather_config").WillReturnRows(configRows)

		notificationRows := sqlmock.NewRows([]string{"state", "created_at"}).
			AddRow(80, time.Now().Unix())
		mock.ExpectQuery("SELECT state, created_at FROM weather_notifications").WillReturnRows(notificationRows)

		err := api.GetNextHourForecast()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})

	t.Run("Weather API error", func(t *testing.T) {
		mockClient := &MockClient{
			DoFunc: func(req *http.Request) (*http.Response, error) {
				return nil, errors.New("API error")
			},
		}
		api := NewWeatherAPI(mockConfig, db, mockClient)

		err := api.GetNextHourForecast()
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
