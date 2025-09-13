package cmd

import (
	"testing-platform/db"
	"testing-platform/db/models"
	"testing-platform/pkg/logger"
)

type Application struct {
	rep *db.Repository
}

func main() {
	// инициализация логгера
	if err := logger.InitGlobal("logs/app.log", logger.LevelInfo); err != nil {
		logger.Fatal("Не удалось инициализировать логгер: %v", err)
	}
	defer logger.GetGlobal().Close()
	logger.Info("Запуск Testing Pltaform Application")
	// загрузка конфигурации
	config, err := models.LoadConfig("config/config.yaml")
	if err != nil {
		logger.Fatal("Ошибка загрузки конфигурации: %v", err)
	}
	// подключение к базе данных
	pool, err := db.Connect(config.Database)
	if err != nil {
		logger.Fatal("Оштбка подключения к базе данных: %v", err)
	}
	defer db.Close(pool)
	// инициализация репозитория
	rep, err := db.NewReposit(pool)
	if err != nil {
		logger.Fatal("Ошибка инициализации репозитория: %v", err)
	}

}
