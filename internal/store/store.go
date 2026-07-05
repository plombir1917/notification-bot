// Package store хранит подписчиков и журнал отправленных напоминаний
// в JSON-файлах. Все операции потокобезопасны и пишутся на диск атомарно.
package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"sync"
)

const (
	subscribersFile = "subscribers.json"
	sentFile        = "sent.json"
)

// Store — хранилище подписчиков и журнала отправок.
type Store struct {
	dir  string
	mu   sync.Mutex
	subs map[int64]struct{}
	sent map[string]struct{}
}

type subscribersDoc struct {
	ChatIDs []int64 `json:"chat_ids"`
}

type sentDoc struct {
	Dates []string `json:"dates"`
}

// Load открывает (или создаёт) хранилище в каталоге dir.
func Load(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("создание каталога состояния %s: %w", dir, err)
	}

	s := &Store{
		dir:  dir,
		subs: make(map[int64]struct{}),
		sent: make(map[string]struct{}),
	}

	var subs subscribersDoc
	if err := readJSON(filepath.Join(dir, subscribersFile), &subs); err != nil {
		return nil, err
	}
	for _, id := range subs.ChatIDs {
		s.subs[id] = struct{}{}
	}

	var sent sentDoc
	if err := readJSON(filepath.Join(dir, sentFile), &sent); err != nil {
		return nil, err
	}
	for _, d := range sent.Dates {
		s.sent[d] = struct{}{}
	}

	return s, nil
}

// AddSubscriber добавляет chat_id. Возвращает true, если подписчик новый.
func (s *Store) AddSubscriber(chatID int64) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.subs[chatID]; ok {
		return false, nil
	}
	s.subs[chatID] = struct{}{}
	return true, s.persistSubscribers()
}

// RemoveSubscriber удаляет chat_id. Возвращает true, если подписчик существовал.
func (s *Store) RemoveSubscriber(chatID int64) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.subs[chatID]; !ok {
		return false, nil
	}
	delete(s.subs, chatID)
	return true, s.persistSubscribers()
}

// Subscribers возвращает отсортированный список подписчиков.
func (s *Store) Subscribers() []int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	ids := make([]int64, 0, len(s.subs))
	for id := range s.subs {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	return ids
}

// IsSent сообщает, было ли уже отправлено напоминание за дату dateKey ("2006-01-02").
func (s *Store) IsSent(dateKey string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.sent[dateKey]
	return ok
}

// MarkSent помечает дату как обработанную, чтобы не отправлять повторно.
func (s *Store) MarkSent(dateKey string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.sent[dateKey]; ok {
		return nil
	}
	s.sent[dateKey] = struct{}{}
	return s.persistSent()
}

// persistSubscribers сохраняет подписчиков (вызывать под mu).
func (s *Store) persistSubscribers() error {
	ids := make([]int64, 0, len(s.subs))
	for id := range s.subs {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	return writeJSON(filepath.Join(s.dir, subscribersFile), subscribersDoc{ChatIDs: ids})
}

// persistSent сохраняет журнал отправок (вызывать под mu).
func (s *Store) persistSent() error {
	dates := make([]string, 0, len(s.sent))
	for d := range s.sent {
		dates = append(dates, d)
	}
	sort.Strings(dates)
	return writeJSON(filepath.Join(s.dir, sentFile), sentDoc{Dates: dates})
}

// readJSON читает JSON из файла; отсутствие файла — не ошибка.
func readJSON(path string, v any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("чтение %s: %w", path, err)
	}
	if len(b) == 0 {
		return nil
	}
	if err := json.Unmarshal(b, v); err != nil {
		return fmt.Errorf("разбор %s: %w", path, err)
	}
	return nil
}

// writeJSON пишет JSON атомарно (во временный файл + rename).
func writeJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("сериализация %s: %w", path, err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return fmt.Errorf("запись %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("переименование %s: %w", tmp, err)
	}
	return nil
}
