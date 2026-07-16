// Package schedule решает, какое напоминание (если вообще) нужно отправить
// в конкретный день. Вся логика переносов и декабрьских исключений здесь.
package schedule

import (
	"time"

	"notification-bot/internal/calendar"
)

// Action — что делать в данный день.
type Action int

const (
	// None — сегодня ничего слать не нужно.
	None Action = iota
	// SendNormal — обычное напоминание о подписании табеля.
	SendNormal
	// SendDecemberNotice — спец-сообщение (16 декабря) про финальный табель.
	SendDecemberNotice
)

func (a Action) String() string {
	switch a {
	case SendNormal:
		return "SendNormal"
	case SendDecemberNotice:
		return "SendDecemberNotice"
	default:
		return "None"
	}
}

// Decide определяет действие на дату today.
//
// Правила:
//   - Обычные месяцы: напоминания на 15-е и на последний рабочий день месяца
//     (30-е, 31-е или последний день февраля). Если целевая дата выпадает
//     на выходной/праздник — напоминание переносится на ближайший рабочий день ДО.
//   - Декабрь: 15-е — обычное напоминание (с переносом); строго 16-го числа —
//     спец-сообщение про финальный табель; второе напоминание (30-е) не шлётся,
//     с 16 по 31 декабря больше ничего.
func Decide(today time.Time, cal *calendar.Calendar) Action {
	year, month, day := today.Date()
	loc := today.Location()

	if month == time.December {
		if day == 16 {
			return SendDecemberNotice
		}
		if sameDate(today, firstTarget(year, month, loc, cal)) {
			return SendNormal
		}
		// Второе напоминание в декабре не отправляется.
		return None
	}

	if sameDate(today, firstTarget(year, month, loc, cal)) {
		return SendNormal
	}
	if sameDate(today, secondTarget(year, month, loc, cal)) {
		return SendNormal
	}
	return None
}

// firstTarget — перенесённая дата первого напоминания (целевое 15-е число).
func firstTarget(year int, month time.Month, loc *time.Location, cal *calendar.Calendar) time.Time {
	return cal.PrevWorkday(dateOf(year, month, 15, loc))
}

// secondTarget — дата второго напоминания: последний рабочий день месяца
// (30-е, 31-е или последний день февраля). Если последний календарный день —
// выходной/праздник, напоминание переносится назад на ближайший рабочий день.
func secondTarget(year int, month time.Month, loc *time.Location, cal *calendar.Calendar) time.Time {
	return cal.PrevWorkday(lastDayOfMonth(year, month, loc))
}

// NextReminder ищет ближайший день (начиная с from), в который будет отправлено
// напоминание, и возвращает его дату вместе с типом действия. Возвращает
// (zero, None), если в пределах горизонта поиска ничего не найдено.
func NextReminder(from time.Time, cal *calendar.Calendar) (time.Time, Action) {
	d := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, from.Location())
	for i := 0; i < 400; i++ {
		if a := Decide(d, cal); a != None {
			return d, a
		}
		d = d.AddDate(0, 0, 1)
	}
	return time.Time{}, None
}

func dateOf(year int, month time.Month, day int, loc *time.Location) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, loc)
}

func lastDayOfMonth(year int, month time.Month, loc *time.Location) time.Time {
	// День 0 следующего месяца == последний день текущего.
	return time.Date(year, month+1, 0, 0, 0, 0, 0, loc)
}

func sameDate(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}
