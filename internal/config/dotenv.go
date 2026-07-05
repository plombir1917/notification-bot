package config

import (
	"bufio"
	"os"
	"strings"
)

// loadDotEnv читает файл вида KEY=VALUE и выставляет переменные окружения,
// которые ещё не заданы. Отсутствие файла — не ошибка (в Docker переменные
// приходят через env_file). Уже установленные переменные окружения имеют
// приоритет и не перезаписываются.
func loadDotEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return // нет файла — молча пропускаем
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		// Снимаем обрамляющие кавычки, если есть.
		val = strings.Trim(val, `"'`)
		if key == "" {
			continue
		}
		if _, exists := os.LookupEnv(key); !exists {
			_ = os.Setenv(key, val)
		}
	}
}
