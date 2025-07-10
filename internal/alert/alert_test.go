package alert

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/imedgar/rain-alert/internal/platform/database"
	"github.com/imedgar/rain-alert/internal/platform/ntfy"
	"github.com/imedgar/rain-alert/internal/weather"
)

type MockClient struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

func (m *MockClient) Do(req *http.Request) (*http.Response, error) {
	return m.DoFunc(req)
}

func TestCheckAndAlert(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	dbMock := database.New(db)

	t.Run("Successful alert", func(t *testing.T) {
		weatherResponse := &weather.WeatherResponse{
			Location: struct {
				Name      string `json:"name"`
				TzID      string `json:"tz_id"`
				Localtime string `json:"localtime"`
			}{Name: "Test Location"},
			Forecast: struct {
				ForecastDay []struct {
					Date string `json:"date"`
					Hour []weather.Hour `json:"hour"`
				} `json:"forecastday"`
			}{
				ForecastDay: []struct {
					Date string `json:"date"`
					Hour []weather.Hour `json:"hour"`
				}{
					{
						Date: "2025-07-10",
						Hour: make([]weather.Hour, 24),
					},
				},
			},
		}
		nextHour := (time.Now().In(time.UTC).Hour() + 1) % 24
		weatherResponse.Forecast.ForecastDay[0].Hour[nextHour] = weather.Hour{ChanceOfRain: 80}

		weatherBody, _ := json.Marshal(weatherResponse)
		mockHTTPClient := &MockClient{
			DoFunc: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader(weatherBody)),
				}, nil
			},
		}

		weatherAPI := weather.NewAPI(mockHTTPClient, "http://weather.com", "test-key")
		ntfyClient := ntfy.New(mockHTTPClient, "http://ntfy.sh", "test-topic")
		alerter := NewAlerter(weatherAPI, dbMock, ntfyClient)

		rows := sqlmock.NewRows([]string{"config", "value"}).
			AddRow("drizzleThreshold", "50").
			AddRow("rainBeforeThreshold", "70")
		mock.ExpectQuery("SELECT config, value FROM weather_config").WillReturnRows(rows)
		mock.ExpectQuery("SELECT state, created_at FROM weather_notifications").WillReturnError(sql.ErrNoRows)
		mock.ExpectExec("INSERT INTO weather_notifications").
			WithArgs(80, sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := alerter.CheckAndAlert("Test Location", "UTC")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})
}
