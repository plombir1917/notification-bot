package calendar

import (
	"testing"
	"time"
)

func d(y int, m time.Month, day int) time.Time {
	return time.Date(y, m, day, 0, 0, 0, 0, time.UTC)
}

func TestIsWorkday2026(t *testing.T) {
	cal, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	cases := []struct {
		date time.Time
		work bool
		note string
	}{
		{d(2026, time.January, 1), false, "новогодний праздник"},
		{d(2026, time.January, 9), false, "перенесённый выходной (9+)"},
		{d(2026, time.January, 12), true, "рабочий понедельник"},
		{d(2026, time.April, 30), true, "сокращённый рабочий день (30*)"},
		{d(2026, time.May, 8), true, "сокращённый рабочий день (8*)"},
		{d(2026, time.May, 11), false, "перенос Дня Победы (11+)"},
		{d(2026, time.December, 15), true, "обычный рабочий вторник"},
		{d(2026, time.December, 31), false, "перенесённый выходной (31+)"},
		{d(2026, time.November, 4), false, "День народного единства"},
	}

	for _, c := range cases {
		if got := cal.IsWorkday(c.date); got != c.work {
			t.Errorf("IsWorkday(%s) = %v, want %v (%s)", c.date.Format("2006-01-02"), got, c.work, c.note)
		}
	}
}

func TestPrevWorkday(t *testing.T) {
	cal, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	// 15 февраля 2026 — воскресенье, 14-е суббота → ближайший рабочий 13-е (пятница).
	got := cal.PrevWorkday(d(2026, time.February, 15))
	want := d(2026, time.February, 13)
	if !got.Equal(want) {
		t.Errorf("PrevWorkday(2026-02-15) = %s, want %s", got.Format("2006-01-02"), want.Format("2006-01-02"))
	}

	// Рабочий день возвращается как есть.
	got = cal.PrevWorkday(d(2026, time.January, 15))
	if want := d(2026, time.January, 15); !got.Equal(want) {
		t.Errorf("PrevWorkday(2026-01-15) = %s, want %s", got.Format("2006-01-02"), want.Format("2006-01-02"))
	}
}

func TestFallbackUnknownYear(t *testing.T) {
	cal, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	// 2030 не загружен → используется правило Пн–Пт.
	if cal.HasYear(2030) {
		t.Skip("2030 присутствует в данных — тест на fallback неактуален")
	}
	// 2030-01-05 — суббота.
	if cal.IsWorkday(d(2030, time.January, 5)) {
		t.Errorf("ожидался выходной для субботы 2030-01-05 (fallback)")
	}
	// 2030-01-07 — понедельник.
	if !cal.IsWorkday(d(2030, time.January, 7)) {
		t.Errorf("ожидался рабочий день для понедельника 2030-01-07 (fallback)")
	}
}
