package db

import (
	"context"
	"database/sql"
	"fmt"
	"testing-platform/pkg/logger"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// кофигурация подключения к бд
type Config struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbname"`
	SSLMode  string `yaml:"sslmode"`
}

func Connect(cfg Config) (*pgxpool.Pool, error) {
	logger.Info("Подключение к базе данных: %s@%s:%d/%s", cfg.User, cfg.Host, cfg.Port, cfg.DBName)
	conStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode) //строка подключения
	// создание конфиг. для пула подключений
	poolConfig, err := pgxpool.ParseConfig(conStr)
	if err != nil {
		logger.Error("Ошибка парсинга конфигурации БД: %v", err)
		return nil, fmt.Errorf("ошибка парсинга конфигурации БД: %w", err)
	}

	// настройка пула подключений
	poolConfig.MaxConns = 10                   // макс. кол-во подключений
	poolConfig.MinConns = 2                    // мин. кол-во подключений
	poolConfig.MaxConnLifetime = time.Hour     // макс. время жизни подключения
	poolConfig.HealthCheckPeriod = time.Minute // период проверки здоровья подключений

	// создание пула подключений
	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		logger.Error("Ошибка создания пула подключений: %v", err)
		return nil, fmt.Errorf("ошибка создания пула подключений: %w", err)
	}

	// проверка подключения
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = pool.Ping(ctx)
	if err != nil {
		logger.Error("Ошибка ping к базе данных: %v", err)
		return nil, fmt.Errorf("ошибка подключения к базе данных: %w", err)
	}

	logger.Info("Подключение к базе данных установлено успешно")
	return pool, nil
}

// функцию для создания стандартного подключения
func ConnectSQL(cfg Config) (*sql.DB, error) {
	logger.Info("Подключение к базе данных (стандартное): %s@%s:%d/%s", cfg.User, cfg.Host, cfg.Port, cfg.DBName)

	conStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode)

	db, err := sql.Open("pgx", conStr)
	if err != nil {
		return nil, fmt.Errorf("ошибка подключения: %w", err)
	}

	// настройка пула подключений
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(time.Hour)

	// проверка подключения
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ошибка ping: %w", err)
	}

	logger.Info("Подключение к базе данных установлено успешно")
	return db, nil
}

// закрытие пула подключений к бд
func Close(pool *pgxpool.Pool) {
	if pool != nil {
		pool.Close()
		logger.Info("Подключение к базе данных закрыто")
	}
}
