// Package config читает настройки бота из переменных окружения.
package config

import (
	"errors"
	"os"
	"strconv"
	"time"
)

// Config — настройки приложения.
type Config struct {
	Token      string    // токен Telegram-бота (@BotFather)
	TZ         string    // имя часового пояса, напр. "Europe/Moscow"
	RemindHour int       // час отправки напоминаний (0–23), по МСК
	StateDir   string    // каталог для JSON-файлов состояния
	TestDate   time.Time // необязательная тестовая дата: имитировать этот день при старте
}

// Load считывает конфигурацию из окружения и подставляет значения по умолчанию.
// Перед чтением подхватывает файл .env из текущего каталога (если он есть),
// не перезаписывая уже заданные переменные окружения.
func Load() (*Config, error) {
	loadDotEnv(".env")

	cfg := &Config{
		Token:      os.Getenv("BOT_TOKEN"),
		TZ:         getEnv("TZ", "Europe/Moscow"),
		RemindHour: 11,
		StateDir:   getEnv("STATE_DIR", "data"),
	}

	if cfg.Token == "" {
		return nil, errors.New("не задана переменная окружения BOT_TOKEN")
	}

	if v := os.Getenv("REMIND_HOUR"); v != "" {
		h, err := strconv.Atoi(v)
		if err != nil || h < 0 || h > 23 {
			return nil, errors.New("REMIND_HOUR должен быть числом от 0 до 23")
		}
		cfg.RemindHour = h
	}

	if v := os.Getenv("TEST_DATE"); v != "" {
		d, err := time.Parse("2006-01-02", v)
		if err != nil {
			return nil, errors.New("TEST_DATE должен быть в формате ГГГГ-ММ-ДД, напр. 2026-12-16")
		}
		cfg.TestDate = d
	}

	return cfg, nil
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
