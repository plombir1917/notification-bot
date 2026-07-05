// Command bot — Telegram-бот, напоминающий о подписании табеля.
package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"
	"time"

	"notification-bot/internal/calendar"
	"notification-bot/internal/config"
	"notification-bot/internal/reminder"
	"notification-bot/internal/store"
	"notification-bot/internal/telegram"
)

func main() {
	log.SetFlags(log.LstdFlags | log.LUTC)

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("конфигурация: %v", err)
	}

	loc, err := time.LoadLocation(cfg.TZ)
	if err != nil {
		log.Fatalf("часовой пояс %q: %v", cfg.TZ, err)
	}

	cal, err := calendar.Load()
	if err != nil {
		log.Fatalf("календарь: %v", err)
	}

	st, err := store.Load(cfg.StateDir)
	if err != nil {
		log.Fatalf("хранилище: %v", err)
	}

	tb, err := telegram.New(cfg.Token, st, cal, loc)
	if err != nil {
		log.Fatalf("telegram: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	svc := reminder.New(tb, st, cal, loc, cfg.RemindHour)
	go svc.Run(ctx)

	log.Printf("бот запущен (TZ=%s, час напоминаний=%d:00, состояние=%s)", cfg.TZ, cfg.RemindHour, cfg.StateDir)
	tb.Start(ctx) // блокирует до отмены ctx
	log.Println("бот остановлен")
}
