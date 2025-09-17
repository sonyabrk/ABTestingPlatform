package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing-platform/db/models"
	"testing-platform/pkg/logger"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
)

// хранение подключения к БД
type Repository struct {
	pool *pgxpool.Pool
	db   *sql.DB
}

// конструктор с проверкой подключения
func NewReposit(pool *pgxpool.Pool, db *sql.DB) (*Repository, error) {
	if pool == nil || db == nil {
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
	return &Repository{pool: pool, db: db}, nil
}

// метод для создания структуры БД
func (r *Repository) CreateSchema(ctx context.Context) error {
	logger.Info("Выполнение миграций БД")

	// использование стандартное подключение для миграций / получение драйвера для базы данных
	driver, err := postgres.WithInstance(r.db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("не удалось создать драйвер БД: %w", err)
	}
	// создание экземпляра миграций
	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		"postgres", driver)
	if err != nil {
		return fmt.Errorf("не удалось создать миграцию: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("не удалось применить миграции: %w", err)
	}

	logger.Info("Миграции БД успешно применены")
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

	sql := `INSERT INTO experiments (name, algorithm_a, algorithm_b, user_percent, is_active, tags) 
             VALUES ($1, $2, $3, $4, $5, $6) RETURNING id, start_date`

	err = tx.QueryRow(ctx, sql, exp.Name, exp.AlgorithmA, exp.AlgorithmB, exp.UserPercent, exp.IsActive, exp.Tags).Scan(&exp.ID, &exp.StartDate)

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
	var args []any

	if res.Clicked {
		if res.ClickedAt != nil {
			// если время клика указано явно
			sql = `INSERT INTO results (user_id, recommendation_id, clicked, clicked_at, rating) 
                   VALUES ($1, $2, $3, $4, $5)`
			args = []any{res.UserId, res.RecommendationId, res.Clicked, res.ClickedAt, res.Rating}
		} else {
			// если время клика не указано - используем текущее время
			sql = `INSERT INTO results (user_id, recommendation_id, clicked, clicked_at, rating) 
                   VALUES ($1, $2, $3, CURRENT_TIMESTAMP, $4)`
			args = []any{res.UserId, res.RecommendationId, res.Clicked, res.Rating}
		}
	} else {
		sql = `INSERT INTO results (user_id, recommendation_id, clicked, rating) 
               VALUES ($1, $2, $3, $4)`
		args = []any{res.UserId, res.RecommendationId, res.Clicked, res.Rating}
	}

	_, err = tx.Exec(ctx, sql, args...)
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

	sql := `SELECT id, name, algorithm_a, algorithm_b, user_percent, start_date, is_active, tags 
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
			&exp.UserPercent, &exp.StartDate, &exp.IsActive, &exp.Tags)
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
func (r *Repository) GetExperimentStats(ctx context.Context, experimentID int) (*models.ExperimentStats, error) {
	logger.Info("Запрос статистики для эксперимента %d", experimentID)

	// получение тегов эксперимента
	var tags []string
	err := r.pool.QueryRow(ctx, "SELECT tags FROM experiments WHERE id = $1", experimentID).Scan(&tags)
	if err != nil {
		logger.Error("Ошибка при получении тегов эксперимента: %v", err)
		return nil, fmt.Errorf("не удалось получить теги эксперимента: %w", err)
	}
	// SQL запрос для агрегации статистики по группам (группировка по группам A/B/общее количество рекомендаций/кол-во кликов/средний рейтинг)
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
	// выполнение запроса к бд
	rows, err := r.pool.Query(ctx, sql, experimentID)
	if err != nil {
		logger.Error("Ошибка при запросе статистики эксперимента: %v", err)
		return nil, fmt.Errorf("не удалось получить статистику эксперимента: %w", err)
	}
	defer rows.Close()

	stats := &models.ExperimentStats{
		ExperimentID: experimentID,
		Groups:       make(map[string]models.GroupStats),
		Tags:         tags,
	}

	var totalRec, totalClicks int
	var totalRating, totalCTR float64
	var groupCount int
	// итерация по результатам запроса (по каждой группе)
	for rows.Next() {
		var group string
		var groupRec, groupClicks int
		var avgRating *float64

		err := rows.Scan(&group, &groupRec, &groupClicks, &avgRating)
		if err != nil {
			logger.Error("Ошибка при сканировании строки статистики: %v", err)
			continue
		}

		groupStats := models.GroupStats{
			Group:                group,
			TotalRecommendations: groupRec,
			TotalClicks:          groupClicks,
		}

		if avgRating != nil {
			groupStats.AvgRating = *avgRating
			totalRating += *avgRating
		} else {
			groupStats.AvgRating = 0
		}

		if groupRec > 0 {
			groupStats.CTR = float64(groupClicks) / float64(groupRec)
			totalCTR += groupStats.CTR
		} else {
			groupStats.CTR = 0
		}

		stats.Groups[group] = groupStats

		// сумма общей статистики
		totalRec += groupRec
		totalClicks += groupClicks
		groupCount++
	}

	// рассчет общей статистики
	if groupCount > 0 {
		stats.TotalStats = models.GroupStats{
			Group:                "total",
			TotalRecommendations: totalRec,
			TotalClicks:          totalClicks,
			AvgRating:            totalRating / float64(groupCount),
			CTR:                  totalCTR / float64(groupCount),
		}
	}

	logger.Info("Статистика для эксперимента %d успешно получена", experimentID)
	return stats, nil
}
