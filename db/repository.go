package db

import (
	"context"
	"fmt"
	"testing-platform/pkg/logger"

	"github.com/jackc/pgx/v5/pgxpool"
)

// хранение подключения к БД
type Repository struct {
	pool *pgxpool.Pool
}

// конструктор
func NewReposit(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// метод для создания структуры БД
func (r *Repository) CreateSchema(ctx context.Context) error {
	logger.Info("Выполнение DDL: создание схемы БД")
	sql := `
		CREATE TYPE algorithm_type AS ENUM ('collaborative', 'content_based', 'hybrid', 'popularity_based');
        
        CREATE TABLE IF NOT EXISTS experiments (
            id SERIAL PRIMARY KEY,
            name VARCHAR(255) NOT NULL UNIQUE,
            algorithm_a algorithm_type NOT NULL,
            algorithm_b algorithm_type NOT NULL,
            user_percent INTEGER CHECK (user_percent > 0 AND user_percent <= 100),
            start_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            is_active BOOLEAN DEFAULT true
        );

		CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			experiment_id INTEGER REFERENCES experiments(id) ON DELETE CASCADE,
			user_id VARCHAR(255) NOT NULL,
			group_name VARCHAR(10) NOT NULL CHECK (group_name IN ('A', 'B')),
			UNIQUE(experiment_id, user_id)
		);

		CREATE TABLE IF NOT EXISTS results (
			id SERIAL PRIMARY KEY,
			user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
			recommendation_id VARCHAR(255) NOT NULL,
			clicked BOOLEAN DEFAULT false,
			clicked_at TIMESTAMP,
			rating INTEGER CHECK (rating >= 1 AND rating <= 5)
		)
	`

	// выполнение sql-запроса
	_, err := r.pool.Exec(ctx, sql)
	if err != nil {
		logger.Error("Ошибка при создании схемы БД: %v", err) // лог ошибки для диагностики
		return fmt.Errorf("не удалось создать структуру базы данных : %w", err)
	}
	logger.Info("Схема БД успешно создана") //лог успешного выполнения
	return nil
}
