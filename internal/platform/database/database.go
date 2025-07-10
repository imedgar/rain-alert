package database

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"time"
)

type DB struct {
	*sql.DB
}

func New(db *sql.DB) *DB {
	return &DB{db}
}

func (db *DB) GetThresholds() (map[string]int, error) {
	rows, err := db.Query("SELECT config, value FROM weather_config")
	if err != nil {
		return nil, fmt.Errorf("querying config: %w", err)
	}
	defer rows.Close()

	configs := make(map[string]int)
	for rows.Next() {
		var config string
		var value string
		if err := rows.Scan(&config, &value); err != nil {
			return nil, fmt.Errorf("scanning row: %w", err)
		}
		th, err := strconv.Atoi(value)
		if err != nil {
			return nil, fmt.Errorf("conversion error: %w", err)
		}
		configs[config] = th
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row error: %w", err)
	}

	return configs, nil
}

func (db *DB) ShouldNotify(thresholds map[string]int) (bool, error) {
	var state int
	var createdAt int64

	row := db.QueryRow("SELECT state, created_at FROM weather_notifications ORDER BY id DESC LIMIT 1")
	err := row.Scan(&state, &createdAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return true, nil
		}
		return false, fmt.Errorf("querying last notification: %w", err)
	}

	age := time.Since(time.Unix(createdAt, 0))
	if age > time.Hour {
		log.Println("Last notification is older than 1 hour, ignoring previous state.")
		return true, nil
	}

	if state > thresholds["rainBeforeThreshold"] {
		return false, nil
	}

	return true, nil
}

func (db *DB) RecordNotification(state int) error {
	_, err := db.Exec("INSERT INTO weather_notifications(state, created_at) VALUES (?, ?)", state, time.Now().Unix())
	if err != nil {
		return fmt.Errorf("inserting notification: %w", err)
	}
	return nil
}