package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/djherbis/times"
	"gopkg.in/yaml.v2"
)

// Config описывает параметры запуска программы.
type Config struct {
	Days    int      `yaml:"days"`
	Folders []string `yaml:"folders"`
}

// readYAMLConfig читает конфигурацию из YAML файла.
func readYAMLConfig(path string) (Config, error) {
	data, err := os.ReadFile(path) // использование os.ReadFile вместо ioutil.ReadFile
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// parseEnvConfig пытается прочесть параметры из переменных окружения.
func parseEnvConfig() (Config, error) {
	var cfg Config
	daysStr := os.Getenv("DAYS")
	if daysStr != "" {
		days, err := strconv.Atoi(daysStr)
		if err != nil {
			return cfg, errors.New("переменная окружения DAYS должна быть числом")
		}
		cfg.Days = days
	}
	foldersStr := os.Getenv("FOLDERS")
	if foldersStr != "" {
		// предполагается, что папки перечислены через запятую
		cfg.Folders = strings.Split(foldersStr, ",")
		for i := range cfg.Folders {
			cfg.Folders[i] = strings.TrimSpace(cfg.Folders[i])
		}
	}
	return cfg, nil
}

// mergeConfigs объединяет конфигурацию из аргументов и окружения.
// Приоритет у аргументов, если они заданы.
func mergeConfigs(argCfg, envCfg Config) Config {
	if argCfg.Days == 0 {
		argCfg.Days = envCfg.Days
	}
	if len(argCfg.Folders) == 0 {
		argCfg.Folders = envCfg.Folders
	}
	return argCfg
}

// processFolder очищает одну папку по заданной логике.
// Возвращает количество найденных файлов и количество удалённых.
func processFolder(folder string, days int) (int, int, error) {
	entries, err := os.ReadDir(folder) // использование os.ReadDir вместо ioutil.ReadDir
	if err != nil {
		return 0, 0, err
	}

	totalFiles := 0
	deletedFiles := 0

	// Находим самый свежий файл (по модификации или созданию)
	var newestTime time.Time
	var fileEntries []os.DirEntry

	// Отбираем обычные файлы
	for _, entry := range entries {
		if entry.Type().IsRegular() {
			totalFiles++
			fileEntries = append(fileEntries, entry)
			fullPath := filepath.Join(folder, entry.Name())
			t, err := times.Stat(fullPath)
			if err != nil {
				log.Printf("Ошибка получения времени для %s: %v\n", fullPath, err)
				continue
			}
			// Определяем максимальную дату между модификацией и созданием
			fileNewest := t.ModTime()
			birth := t.BirthTime()
			if birth.After(fileNewest) {
				fileNewest = birth
			}
			if fileNewest.After(newestTime) {
				newestTime = fileNewest
			}
		}
	}

	// Если файлов не найдено, пропускаем папку.
	if newestTime.IsZero() {
		log.Printf("Папка %s не содержит файлов для анализа\n", folder)
		return totalFiles, deletedFiles, nil
	}

	// Вычисляем день отсечки: от самой свежей даты отступаем назад на days дней.
	cutoff := newestTime.AddDate(0, 0, -days)
	log.Printf("Папка: %s, самая свежая дата: %v, день отсечки: %v\n", folder, newestTime, cutoff)

	// Удаляем файлы, если и время модификации, и время создания старше cutoff.
	for _, entry := range fileEntries {
		fullPath := filepath.Join(folder, entry.Name())
		t, err := times.Stat(fullPath)
		if err != nil {
			log.Printf("Ошибка получения времени для %s: %v\n", fullPath, err)
			continue
		}
		modTime := t.ModTime()
		birthTime := t.BirthTime()

		if modTime.Before(cutoff) && birthTime.Before(cutoff) {
			err := os.Remove(fullPath)
			if err != nil {
				log.Printf("Ошибка удаления файла %s: %v\n", fullPath, err)
			} else {
				log.Printf("Удалён файл: %s\n", fullPath)
				deletedFiles++
			}
		}
	}
	return totalFiles, deletedFiles, nil
}

// writeLog записывает результаты работы в лог-файл.
func writeLog(timestamp time.Time, totalFiles, deletedFiles int) error {
	logFile := "cleanup.log"
	line := fmt.Sprintf("%s - файлов обнаружено: %d, удалено: %d\n", timestamp.Format(time.RFC3339), totalFiles, deletedFiles)
	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(line)
	return err
}

// isNumber проверяет, можно ли преобразовать строку в число.
func isNumber(s string) bool {
	_, err := strconv.Atoi(s)
	return err == nil
}

func main() {
	// Флаг для вывода справки
	help := flag.Bool("help", false, "Показать справку")
	flag.Parse()
	if *help {
		fmt.Println("Usage: cleanup [days|config.yml] [folder1 folder2 ...]")
		return
	}

	var cfg Config

	args := flag.Args()
	// Если аргументы командной строки заданы
	if len(args) > 0 {
		if isNumber(args[0]) {
			// Первый аргумент – количество дней
			days, err := strconv.Atoi(args[0])
			if err != nil {
				log.Fatalf("Неверное значение для количества дней: %v", err)
			}
			cfg.Days = days
			if len(args) > 1 {
				cfg.Folders = args[1:]
			}
		} else {
			// Первый аргумент – путь к YAML файлу конфигурации
			loadedCfg, err := readYAMLConfig(args[0])
			if err != nil {
				log.Fatalf("Ошибка чтения YAML файла: %v", err)
			}
			cfg = loadedCfg
		}
	}

	// Если не все параметры заданы через аргументы, пытаемся прочесть из переменных окружения.
	envCfg, _ := parseEnvConfig()
	cfg = mergeConfigs(cfg, envCfg)

	if cfg.Days <= 0 || len(cfg.Folders) == 0 {
		log.Fatal("Не заданы необходимые параметры. Требуется указать количество дней (целое число) и список папок для очистки.")
	}

	overallTotal := 0
	overallDeleted := 0

	for _, folder := range cfg.Folders {
		folder = strings.TrimSpace(folder)
		if folder == "" {
			continue
		}
		// Проверяем, существует ли папка
		info, err := os.Stat(folder)
		if err != nil || !info.IsDir() {
			log.Printf("Папка '%s' не найдена или не является директорией, пропускаем\n", folder)
			continue
		}
		total, deleted, err := processFolder(folder, cfg.Days)
		if err != nil {
			log.Printf("Ошибка обработки папки '%s': %v\n", folder, err)
			continue
		}
		overallTotal += total
		overallDeleted += deleted
	}

	now := time.Now()
	if err := writeLog(now, overallTotal, overallDeleted); err != nil {
		log.Printf("Ошибка записи лога: %v\n", err)
	} else {
		log.Printf("Результаты работы записаны в cleanup.log\n")
	}
}
