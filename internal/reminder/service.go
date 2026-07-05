// Package reminder связывает планировщик, логику решения и рассылку:
// ежедневно в заданный час проверяет, нужно ли слать напоминание.
package reminder

import (
	"context"
	"log"
	"time"

	"notification-bot/internal/calendar"
	"notification-bot/internal/schedule"
	"notification-bot/internal/store"
	"notification-bot/internal/telegram"
)

// Service — планировщик напоминаний.
type Service struct {
	bot   *telegram.Bot
	store *store.Store
	cal   *calendar.Calendar
	loc   *time.Location
	hour  int
}

// New создаёт сервис напоминаний.
func New(b *telegram.Bot, st *store.Store, cal *calendar.Calendar, loc *time.Location, hour int) *Service {
	return &Service{bot: b, store: st, cal: cal, loc: loc, hour: hour}
}

// Run запускает цикл планировщика. Блокирует до отмены ctx.
//
// При старте выполняется «догоняющая» проверка: если сегодня уже наступил час
// напоминания и напоминание за сегодня ещё не отправлялось — оно уйдёт сразу
// (это защищает от пропуска при рестарте процесса после нужного времени).
func (s *Service) Run(ctx context.Context) {
	now := time.Now().In(s.loc)
	if now.Hour() >= s.hour {
		s.tick(ctx, now)
	}

	for {
		next := s.nextFire(time.Now().In(s.loc))
		wait := time.Until(next)
		log.Printf("следующая проверка напоминаний: %s (через %s)", next.Format(time.RFC3339), wait.Truncate(time.Second))

		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			s.tick(ctx, time.Now().In(s.loc))
		}
	}
}

// tick проверяет дату и при необходимости рассылает напоминание (один раз в день).
func (s *Service) tick(ctx context.Context, now time.Time) {
	action := schedule.Decide(now, s.cal)
	if action == schedule.None {
		return
	}

	key := now.Format("2006-01-02")
	if s.store.IsSent(key) {
		return
	}

	var text string
	switch action {
	case schedule.SendDecemberNotice:
		text = telegram.MsgDecember
	default:
		text = telegram.MsgNormal
	}

	s.bot.Broadcast(ctx, text)
	if err := s.store.MarkSent(key); err != nil {
		log.Printf("не удалось сохранить журнал отправок за %s: %v", key, err)
	}
}

// nextFire возвращает ближайший момент времени hour:00 после from.
func (s *Service) nextFire(from time.Time) time.Time {
	fire := time.Date(from.Year(), from.Month(), from.Day(), s.hour, 0, 0, 0, s.loc)
	if !fire.After(from) {
		fire = fire.AddDate(0, 0, 1)
	}
	return fire
}
