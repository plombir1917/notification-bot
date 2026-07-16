package schedule

import (
	"testing"
	"time"

	"notification-bot/internal/calendar"
)

func mustCalendar(t *testing.T) *calendar.Calendar {
	t.Helper()
	cal, err := calendar.Load()
	if err != nil {
		t.Fatalf("загрузка календаря: %v", err)
	}
	return cal
}

func day(loc *time.Location, y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 11, 0, 0, 0, loc)
}

func TestDecide2026(t *testing.T) {
	cal := mustCalendar(t)
	msk, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		t.Fatalf("tz: %v", err)
	}

	tests := []struct {
		name string
		date time.Time
		want Action
	}{
		// Январь 2026: 15-е — четверг (рабочий) → напоминание в этот день.
		{"jan-15-workday", day(msk, 2026, time.January, 15), SendNormal},
		// Январь: последний рабочий день — 30-е (пятница), т.к. 31-е — суббота.
		{"jan-30-last-workday", day(msk, 2026, time.January, 30), SendNormal},
		{"jan-31-saturday-none", day(msk, 2026, time.January, 31), None},
		// Январь 14-е — не целевой день.
		{"jan-14-none", day(msk, 2026, time.January, 14), None},

		// Февраль 2026: 15-е — воскресенье → перенос назад на пятницу 13-е.
		{"feb-15-sunday-none", day(msk, 2026, time.February, 15), None},
		{"feb-13-friday-shifted", day(msk, 2026, time.February, 13), SendNormal},
		// Февраль: второе напоминание = последний день (28-е, суббота) → перенос на пятницу 27-е.
		{"feb-28-saturday-none", day(msk, 2026, time.February, 28), None},
		{"feb-27-friday-shifted", day(msk, 2026, time.February, 27), SendNormal},

		// Март 2026: 15-е — воскресенье → перенос на пятницу 13-е.
		{"mar-15-sunday-none", day(msk, 2026, time.March, 15), None},
		{"mar-13-friday-shifted", day(msk, 2026, time.March, 13), SendNormal},
		// Март: последний рабочий день — 31-е (вторник). 30-е — не целевой день.
		{"mar-30-none", day(msk, 2026, time.March, 30), None},
		{"mar-31-last-workday", day(msk, 2026, time.March, 31), SendNormal},

		// Август 2026: 15-е — суббота → перенос на пятницу 14-е.
		{"aug-15-saturday-none", day(msk, 2026, time.August, 15), None},
		{"aug-14-friday-shifted", day(msk, 2026, time.August, 14), SendNormal},
		// Август: последний рабочий день — 31-е (понедельник); 30-е (вс) и 28-е — не целевые.
		{"aug-28-none", day(msk, 2026, time.August, 28), None},
		{"aug-30-sunday-none", day(msk, 2026, time.August, 30), None},
		{"aug-31-last-workday", day(msk, 2026, time.August, 31), SendNormal},

		// Декабрь 2026: 15-е — вторник (рабочий) → обычное напоминание.
		{"dec-15-normal", day(msk, 2026, time.December, 15), SendNormal},
		// Декабрь: 16-е — спец-сообщение (строго).
		{"dec-16-notice", day(msk, 2026, time.December, 16), SendDecemberNotice},
		// Декабрь: второе напоминание (30-е) НЕ отправляется.
		{"dec-30-skipped", day(msk, 2026, time.December, 30), None},
		// Декабрь: 28-е (перенесённое 30-е в обычной логике) тоже не шлётся.
		{"dec-28-skipped", day(msk, 2026, time.December, 28), None},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Decide(tt.date, cal); got != tt.want {
				t.Errorf("Decide(%s) = %v, want %v", tt.date.Format("2006-01-02"), got, tt.want)
			}
		})
	}
}

// TestExactlyOneReminderPerHalfMonth проверяет, что в каждом месяце ровно два
// обычных напоминания (кроме декабря, где второе пропущено), и что декабрьский
// спец-текст приходит ровно один раз.
func TestExactlyOneReminderPerHalfMonth(t *testing.T) {
	cal := mustCalendar(t)
	msk, _ := time.LoadLocation("Europe/Moscow")

	for m := time.January; m <= time.December; m++ {
		normals, notices := 0, 0
		last := time.Date(2026, m+1, 0, 0, 0, 0, 0, msk).Day()
		for d := 1; d <= last; d++ {
			switch Decide(day(msk, 2026, m, d), cal) {
			case SendNormal:
				normals++
			case SendDecemberNotice:
				notices++
			}
		}

		wantNormals, wantNotices := 2, 0
		if m == time.December {
			wantNormals, wantNotices = 1, 1
		}
		if normals != wantNormals {
			t.Errorf("месяц %d: обычных напоминаний %d, ожидалось %d", m, normals, wantNormals)
		}
		if notices != wantNotices {
			t.Errorf("месяц %d: спец-сообщений %d, ожидалось %d", m, notices, wantNotices)
		}
	}
}
