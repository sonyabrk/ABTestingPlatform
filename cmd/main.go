package main

import (
	"log"
	"strings"
	"testing-platform/db"
	"testing-platform/db/models"
	"testing-platform/pkg/logger"
	"testing-platform/ui"

	"fyne.io/fyne/v2/app"
)

func main() {
	// загрузка конфигурации
	config, err := models.LoadConfig("config/config.yaml")
	if err != nil {
		log.Fatalf("Ошибка загрузки конфигурации: %v", err)
	}

	// инициализация логгера с настройками из конфига
	if err := logger.InitGlobal(config.Logging.File, convertLogLevel(config.Logging.Level)); err != nil {
		log.Fatalf("Не удалось инициализировать логгер: %v", err)
	}
	defer logger.GetGlobal().Close()

	// установка дополнительных параметров логгера
	logger.GetGlobal().SetMaxSize(config.Logging.MaxSize)
	logger.GetGlobal().SetMaxBackups(config.Logging.MaxBackups)

	logger.Info("Запуск Testing Platform Application")

	// подключение к базе данных (стандартное для миграций)
	sqlDB, err := db.ConnectSQL(config.Database) // Добавьте эту строку
	if err != nil {
		logger.Fatal("Ошибка подключения к базе данных: %v", err)
	}
	defer sqlDB.Close()

	// подключение к базе данных (pgx для основного использования)
	pool, err := db.Connect(config.Database)
	if err != nil {
		logger.Fatal("Ошибка подключения к базе данных: %v", err)
	}
	defer db.Close(pool)

	// инициализация репозитория с обоими подключениями
	rep, err := db.NewReposit(pool, sqlDB)
	if err != nil {
		logger.Fatal("Ошибка инициализации репозитория: %v", err)
	}

	// создание UI
	fyneApp := app.New()
	mainWindow := ui.NewMainWindow(fyneApp, rep)
	mainWindow.CreateUI()
	mainWindow.Show()

	// запуск приложения
	fyneApp.Run()
}

// вспомогательная функция для преобразования строки в уровень логирования
func convertLogLevel(level string) int {
	switch strings.ToLower(level) {
	case "debug":
		return logger.LevelDebug
	case "info":
		return logger.LevelInfo
	case "warn":
		return logger.LevelWarn
	case "error":
		return logger.LevelError
	case "fatal":
		return logger.LevelFatal
	default:
		return logger.LevelInfo
	}
}
