package database

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestGetThresholds(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	dbMock := New(db)

	t.Run("Successful retrieval", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"config", "value"}).
			AddRow("drizzleThreshold", "50").
			AddRow("rainBeforeThreshold", "70")
		mock.ExpectQuery("SELECT config, value FROM weather_config").WillReturnRows(rows)

		thresholds, err := dbMock.GetThresholds()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if len(thresholds) != 2 {
			t.Errorf("expected 2 thresholds, got %d", len(thresholds))
		}

		if thresholds["drizzleThreshold"] != 50 {
			t.Errorf("expected drizzleThreshold to be 50, got %d", thresholds["drizzleThreshold"])
		}

		if thresholds["rainBeforeThreshold"] != 70 {
			t.Errorf("expected rainBeforeThreshold to be 70, got %d", thresholds["rainBeforeThreshold"])
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})
}

func TestShouldNotify(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	dbMock := New(db)

	t.Run("No recent notifications", func(t *testing.T) {
		mock.ExpectQuery("SELECT state, created_at FROM weather_notifications").WillReturnRows(sqlmock.NewRows([]string{"state", "created_at"}))

		notify, err := dbMock.ShouldNotify(map[string]int{"rainBeforeThreshold": 70})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if !notify {
			t.Error("expected to be notified, but it was not")
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})

	t.Run("Recent notification with low rain chance", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"state", "created_at"}).AddRow(60, time.Now().Unix())
		mock.ExpectQuery("SELECT state, created_at FROM weather_notifications").WillReturnRows(rows)

		notify, err := dbMock.ShouldNotify(map[string]int{"rainBeforeThreshold": 70})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if !notify {
			t.Error("expected to be notified, but it was not")
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})

	t.Run("Recent notification with high rain chance", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"state", "created_at"}).AddRow(80, time.Now().Unix())
		mock.ExpectQuery("SELECT state, created_at FROM weather_notifications").WillReturnRows(rows)

		notify, err := dbMock.ShouldNotify(map[string]int{"rainBeforeThreshold": 70})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if notify {
			t.Error("expected not to be notified, but it was")
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})
}

func TestRecordNotification(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	dbMock := New(db)

	t.Run("Successful recording", func(t *testing.T) {
		mock.ExpectExec("INSERT INTO weather_notifications").
			WithArgs(80, sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := dbMock.RecordNotification(80)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})
}
