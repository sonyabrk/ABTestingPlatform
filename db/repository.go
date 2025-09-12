package db

import (
	"context"
	"errors"
	"fmt"
	"testing-platform/db/models"
	"testing-platform/pkg/logger"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// хранение подключения к БД
type Repository struct {
	pool *pgxpool.Pool
}

// конструктор с проверкой подключения
func NewReposit(pool *pgxpool.Pool) (*Repository, error) {
	if pool == nil {
		return nil, errors.New("пул подключений не может быть nil")
	}
	// проверка работы подключения
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	err := pool.Ping(ctx)
	if err != nil {
		return nil, fmt.Errorf("пул подключений не активен: %w", err)
	}
	logger.Info("Репозиторий успешно инициализирован")
	return &Repository{pool: pool}, nil
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

// создание нового эксперимента
func (r *Repository) CreateExperiment(ctx context.Context, exp *models.Experiment) error {
	if err := exp.Validate(); err != nil {
		return err
	}
	// начало транзакции только для операций с бд
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	logger.Info("Выполнение DML: создание эксперимента '%s'", exp.Name)

	sql := `INSERT INTO experiments (name, algorithm_a, algorithm_b, user_percent, is_active) 
	         VALUES ($1, $2, $3, $4, $5) RETURNING id, start_date`

	err = tx.QueryRow(ctx, sql, exp.Name, exp.AlgorithmA, exp.AlgorithmB, exp.UserPercent, exp.IsActive).Scan(&exp.ID, &exp.StartDate)

	if err != nil {
		logger.Error("Ошибка при создании эксперимента: %v", err)
		return fmt.Errorf("не удалось создать эксперимент: %w", err)
	}
	// если все успешно, то деламе коммит транзакции
	if err := tx.Commit(ctx); err != nil {
		return err
	}

	logger.Info("Эксперимент '%s' успешно создан с ID %d", exp.Name, exp.ID)
	return nil
}

// добавление пользователя в эксперимент
func (r *Repository) AddUserToExperiment(ctx context.Context, user *models.User) error {
	if err := user.Validate(); err != nil {
		return err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	logger.Info("Добавление пользователя %s в эксперимент %d (группа %s)", user.UserId, user.ExperimentId, user.GroupName)
	sql := `INSERT INTO users (experiment_id, user_id, group_name) VALUES ($1, $2, $3)`

	_, err = tx.Exec(ctx, sql, user.ExperimentId, user.UserId, user.GroupName)
	if err != nil {
		logger.Error("Ошибка при добавлении пользователя в эксперимент: %v", err)
		return fmt.Errorf("не удалось добавить пользователя в эксперимент: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	logger.Info("Пользователь %s успешно добавлен в эксперимент %d", user.UserId, user.ExperimentId)
	return nil
}

// добавление результата рекомендации
func (r *Repository) AddResult(ctx context.Context, res *models.Result) error {
	if err := res.Validate(); err != nil {
		return err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	logger.Info("Добавление результата для пользователя %d, рекомендация %s", res.UserId, res.RecommendationId)

	var sql string

	if res.Clicked {
		sql = `INSERT INTO results (user_id, recommendation_id, clicked, clicked_at, rating) 
               VALUES ($1, $2, $3, CURRENT_TIMESTAMP, $4)`
		_, err = tx.Exec(ctx, sql, res.UserId, res.RecommendationId, res.Clicked, res.Rating)
	} else {
		sql = `INSERT INTO results (user_id, recommendation_id, clicked, rating) 
               VALUES ($1, $2, $3, $4)`
		_, err = tx.Exec(ctx, sql, res.UserId, res.RecommendationId, res.Clicked, res.Rating)
	}

	if err != nil {
		logger.Error("Ошибка при добавлении результата: %v", err)
		return fmt.Errorf("не удалось добавить результат: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}
	logger.Info("Результат для пользователя %d успешно добавлен", res.UserId)
	return nil
}

// возвращение списока всех экспериментов
func (r *Repository) GetExperiments(ctx context.Context) ([]models.Experiment, error) {
	logger.Info("Запрос списка экспериментов")

	sql := `SELECT id, name, algorithm_a, algorithm_b, user_percent, start_date, is_active 
	         FROM experiments ORDER BY start_date DESC`

	rows, err := r.pool.Query(ctx, sql)
	if err != nil {
		logger.Error("Ошибка при запросе списка экспериментов: %v", err)
		return nil, fmt.Errorf("не удалось получить список экспериментов: %w", err)
	}
	defer rows.Close()

	var experiments []models.Experiment
	for rows.Next() {
		var exp models.Experiment
		err := rows.Scan(&exp.ID, &exp.Name, &exp.AlgorithmA, &exp.AlgorithmB,
			&exp.UserPercent, &exp.StartDate, &exp.IsActive)
		if err != nil {
			logger.Error("Ошибка при сканировании строки эксперимента: %v", err)
			continue
		}
		experiments = append(experiments, exp)
	}

	logger.Info("Получено %d экспериментов", len(experiments))
	return experiments, nil
}

// возвращение результатов для конкретного эксперимента
func (r *Repository) GetExperimentResults(ctx context.Context, experimentID int) ([]models.Result, error) {
	logger.Info("Запрос результатов эксперимента %d", experimentID)

	sql := `SELECT r.id, r.user_id, r.recommendation_id, r.clicked, r.clicked_at, r.rating
	         FROM results r
	         JOIN users u ON r.user_id = u.id
	         WHERE u.experiment_id = $1`

	rows, err := r.pool.Query(ctx, sql, experimentID)
	if err != nil {
		logger.Error("Ошибка при запросе результатов эксперимента: %v", err)
		return nil, fmt.Errorf("не удалось получить результаты эксперимента: %w", err)
	}
	defer rows.Close()

	var results []models.Result
	for rows.Next() {
		var res models.Result
		err := rows.Scan(&res.ID, &res.UserId, &res.RecommendationId,
			&res.Clicked, &res.ClickedAt, &res.Rating)
		if err != nil {
			logger.Error("Ошибка при сканировании строки результата: %v", err)
			continue
		}
		results = append(results, res)
	}

	logger.Info("Получено %d результатов для эксперимента %d", len(results), experimentID)
	return results, nil
}

// обновление статуса эксперимента
func (r *Repository) UpdateExperimentStatus(ctx context.Context, experimentID int, isActive bool) error {
	status := "активирован"
	if !isActive {
		status = "деактивирован"
	}

	logger.Info("Обновление статуса эксперимента %d: %s", experimentID, status)

	sql := `UPDATE experiments SET is_active = $1 WHERE id = $2`

	_, err := r.pool.Exec(ctx, sql, isActive, experimentID)
	if err != nil {
		logger.Error("Ошибка при обновлении статуса эксперимента: %v", err)
		return fmt.Errorf("не удалось обновить статус эксперимента: %w", err)
	}

	logger.Info("Статус эксперимента %d успешно обновлен", experimentID)
	return nil
}

// возвращение статистики по эксперименту
// считает статистику по группам пользователей: количество рекомендаций, кликов, средний рейтинг и CTR (метрика кликабельности)
func (r *Repository) GetExperimentStats(ctx context.Context, experimentID int) (map[string]any, error) {
	logger.Info("Запрос статистики для эксперимента %d", experimentID)

	sql := `
		SELECT 
			u.group_name,
			COUNT(r.id) as total_recommendations,
			SUM(CASE WHEN r.clicked THEN 1 ELSE 0 END) as total_clicks,
			AVG(CASE WHEN r.rating > 0 THEN r.rating::float ELSE NULL END) as avg_rating
		FROM users u
		LEFT JOIN results r ON u.id = r.user_id
		WHERE u.experiment_id = $1
		GROUP BY u.group_name
	`

	rows, err := r.pool.Query(ctx, sql, experimentID)
	if err != nil {
		logger.Error("Ошибка при запросе статистики эксперимента: %v", err)
		return nil, fmt.Errorf("не удалось получить статистику эксперимента: %w", err)
	}
	defer rows.Close()

	stats := make(map[string]interface{})
	for rows.Next() {
		var group string
		var totalRec, totalClicks int
		var avgRating *float64

		err := rows.Scan(&group, &totalRec, &totalClicks, &avgRating)
		if err != nil {
			logger.Error("Ошибка при сканировании строки статистики: %v", err)
			continue
		}

		groupStats := make(map[string]interface{})
		groupStats["total_recommendations"] = totalRec
		groupStats["total_clicks"] = totalClicks

		if avgRating != nil {
			groupStats["avg_rating"] = *avgRating
		} else {
			groupStats["avg_rating"] = 0.0
		}

		if totalRec > 0 {
			groupStats["ctr"] = float64(totalClicks) / float64(totalRec)
		} else {
			groupStats["ctr"] = 0.0
		}

		stats[group] = groupStats
	}

	logger.Info("Статистика для эксперимента %d успешно получена", experimentID)
	return stats, nil
}
