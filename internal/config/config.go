package config

import (
	"context"

	"github.com/sethvargo/go-envconfig"
)

type Config struct {
	WeatherApiKey         string `env:"WEATHER_API_KEY,required"`
	PushNotificationTopic string `env:"PUSH_NOTIFICATION_TOPIC,required"`
	DatabaseUrl           string `env:"DB_URL,required"`
	DatabaseToken         string `env:"DB_TOKEN,required"`
	Location              string `env:"LOCATION,required"`
	Timezone              string `env:"TIMEZONE,required"`
}

func NewConfig(ctx context.Context) (*Config, error) {
	var c Config
	if err := envconfig.Process(ctx, &c); err != nil {
		return nil, err
	}
	return &c, nil
}