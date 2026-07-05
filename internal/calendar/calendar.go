// Package calendar предоставляет производственный календарь РФ:
// определяет рабочие/нерабочие дни с учётом праздников и переносов.
//
// Данные хранятся во встроенных JSON-файлах (data/<год>.json) в формате
// xmlcalendar.ru. Чтобы добавить новый год — положите рядом файл вида
// data/2027.json, скачанный с https://xmlcalendar.ru/data/ru/2027/calendar.json
package calendar

import (
	"embed"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

//go:embed data/*.json
var dataFS embed.FS

// rawYear описывает формат xmlcalendar.ru.
//
// Поле days каждого месяца — список особых дней через запятую, где число —
// это нерабочий день (выходной или праздник), суффикс "+" — перенесённый
// праздничный/выходной день (тоже нерабочий), а суффикс "*" — сокращённый
// предпраздничный РАБОЧИЙ день (в набор нерабочих не попадает).
type rawYear struct {
	Year   int `json:"year"`
	Months []struct {
		Month int    `json:"month"`
		Days  string `json:"days"`
	} `json:"months"`
}

// Calendar хранит множество нерабочих дней по загруженным годам.
type Calendar struct {
	nonWorking map[string]struct{} // ключ — дата в формате "2006-01-02"
	years      map[int]struct{}    // годы, для которых есть данные
}

// Load читает все встроенные JSON-файлы календаря.
func Load() (*Calendar, error) {
	entries, err := dataFS.ReadDir("data")
	if err != nil {
		return nil, fmt.Errorf("чтение встроенных данных календаря: %w", err)
	}

	c := &Calendar{
		nonWorking: make(map[string]struct{}),
		years:      make(map[int]struct{}),
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		b, err := dataFS.ReadFile("data/" + e.Name())
		if err != nil {
			return nil, fmt.Errorf("чтение %s: %w", e.Name(), err)
		}

		var ry rawYear
		if err := json.Unmarshal(b, &ry); err != nil {
			return nil, fmt.Errorf("разбор %s: %w", e.Name(), err)
		}
		if err := c.addYear(ry); err != nil {
			return nil, fmt.Errorf("обработка %s: %w", e.Name(), err)
		}
	}

	if len(c.years) == 0 {
		return nil, fmt.Errorf("не найдено ни одного файла календаря")
	}
	return c, nil
}

func (c *Calendar) addYear(ry rawYear) error {
	c.years[ry.Year] = struct{}{}
	for _, m := range ry.Months {
		for _, tok := range strings.Split(m.Days, ",") {
			tok = strings.TrimSpace(tok)
			if tok == "" {
				continue
			}
			// "*" — сокращённый рабочий день, не нерабочий.
			if strings.HasSuffix(tok, "*") {
				continue
			}
			tok = strings.TrimRight(tok, "+")
			day, err := strconv.Atoi(tok)
			if err != nil {
				return fmt.Errorf("некорректный день %q в месяце %d: %w", tok, m.Month, err)
			}
			key := fmt.Sprintf("%04d-%02d-%02d", ry.Year, m.Month, day)
			c.nonWorking[key] = struct{}{}
		}
	}
	return nil
}

// HasYear сообщает, есть ли точные данные календаря для года.
func (c *Calendar) HasYear(year int) bool {
	_, ok := c.years[year]
	return ok
}

// IsWorkday возвращает true, если день рабочий.
//
// Для годов с данными используется точный производственный календарь.
// Для прочих годов — простое правило: рабочие дни Пн–Пт.
func (c *Calendar) IsWorkday(d time.Time) bool {
	if c.HasYear(d.Year()) {
		_, nonWork := c.nonWorking[d.Format("2006-01-02")]
		return !nonWork
	}
	wd := d.Weekday()
	return wd != time.Saturday && wd != time.Sunday
}

// PrevWorkday возвращает ближайший рабочий день, начиная с d и двигаясь назад.
// Если d уже рабочий — возвращает d (нормализованный к началу дня).
func (c *Calendar) PrevWorkday(d time.Time) time.Time {
	d = time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, d.Location())
	for !c.IsWorkday(d) {
		d = d.AddDate(0, 0, -1)
	}
	return d
}
