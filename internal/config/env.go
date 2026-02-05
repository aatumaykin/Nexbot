package config

import (
	"os"
	"strings"
)

// LoadEnv загружает переменные окружения из .env файла.
// Читает файл по указанному пути, парсит строки в формате KEY=VALUE,
// игнорирует пустые строки и комментарии (строки начинающиеся с #),
// устанавливает переменные через os.Setenv().
// Возвращает ошибку если файл не существует или не может быть прочитан.
func LoadEnv(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	lines := strings.Split(string(data), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Пропустить пустые строки и комментарии
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Разделить ключ и значение
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		if key != "" {
			os.Setenv(key, value)
		}
	}

	return nil
}

// LoadEnvOptional загружает переменные окружения из .env файла, если он существует.
// Проверяет существование файла через os.Stat().
// Если файл существует - вызывает LoadEnv(path).
// Если файл не существует - возвращает nil (без ошибки).
func LoadEnvOptional(path string) error {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	return LoadEnv(path)
}
