package models

import (
	"errors"
	"slices"
	"time"
)

// Experiment представляет сущность эксперимента A/B тестирования
type Experiment struct {
	ID          int       `db:"id" json:"id"`
	Name        string    `db:"name" json:"name"`
	AlgorithmA  string    `db:"algorithm_a" json:"algorithm_a"`
	AlgorithmB  string    `db:"algorithm_b" json:"algorithm_b"`
	UserPercent int       `db:"user_percent" json:"user_percent"`
	StartDate   time.Time `db:"start_date" json:"start_date"`
	IsActive    bool      `db:"is_active" json:"is_active"`
	Tags        []string  `db:"tags" json:"tags"`
}

// GroupStats представляет статистику для одной группы
type GroupStats struct {
	Group                string  `json:"group"`
	TotalRecommendations int     `json:"total_recommendations"`
	TotalClicks          int     `json:"total_clicks"`
	AvgRating            float64 `json:"avg_rating"`
	CTR                  float64 `json:"ctr"`
}

// ExperimentStats представляет полную статистику эксперимента
type ExperimentStats struct {
	ExperimentID int                   `json:"experiment_id"`
	Groups       map[string]GroupStats `json:"groups"`
	TotalStats   GroupStats            `json:"total_stats"`
	Tags         []string              `json:"tags,omitempty"`
}

// ExperimentFilter представлет структуру для фильтрации
type ExperimentFilter struct {
	AlgorithmA    string
	AlgorithmB    string
	IsActive      *bool // использование указателя для возможности передачи nil
	StartDateFrom time.Time
	StartDateTo   time.Time
}

// ExperimentResult представляет сводные данные эксперимента с JOIN
type ExperimentResult struct {
	ID           int     `json:"id"`
	Name         string  `json:"name"`
	AlgorithmA   string  `json:"algorithm_a"`
	AlgorithmB   string  `json:"algorithm_b"`
	TotalResults int     `json:"total_results"`
	TotalClicks  int     `json:"total_clicks"`
	AvgRating    float64 `json:"avg_rating"`
}

// возврат имени таблицы в БД
func (Experiment) TableName() string {
	return "experiments"
}

// проверка корректности данных эксперимента
func (e *Experiment) Validate() error {
	validAlgorithms := []string{"collaborative", "content_based", "hybrid", "popularity_based"}
	if !slices.Contains(validAlgorithms, e.AlgorithmA) {
		return errors.New("неверный тип алгоритма A")
	}
	if !slices.Contains(validAlgorithms, e.AlgorithmB) {
		return errors.New("неверный тип алгоритма B")
	}
	if e.Name == "" {
		return errors.New("название эксперимента не может быть пустым")
	}
	if len(e.Name) > 255 {
		return errors.New("название эксперимента слишком длинное")
	}
	if e.UserPercent < 1 || e.UserPercent > 100 {
		return errors.New("процент пользователей должен быть от 1 до 100")
	}
	if e.AlgorithmA == "" {
		return errors.New("алгоритм A не может быть пустым")
	}
	if e.AlgorithmB == "" {
		return errors.New("алгоритм B не может быть пустым")
	}
	if e.AlgorithmA == e.AlgorithmB {
		return errors.New("алгоритмы A и B не могут быть одинаковыми")
	}
	if len(e.Tags) > 10 {
		return errors.New("слишком много тегов (максимум 10)")
	}
	for _, tag := range e.Tags {
		if len(tag) > 50 {
			return errors.New("тег слишком длинный (максимум 50 символов)")
		}
	}
	return nil
}

// методы для инкапсулиции логики проверки

// проверка, активен ли эксперимент
func (e *Experiment) IsRunning() bool {
	return e.IsActive
}

// возвращение названия алгоритмов для отображения
func (e *Experiment) GetAlgorithmNames() (string, string) {
	return e.AlgorithmA, e.AlgorithmB
}

// проверка, можно ли добавлять пользователей в эксперимент
func (e *Experiment) CanAcceptMoreUsers(currentCount int) bool {
	return e.IsRunning() && currentCount < e.UserPercent*1000
}

// проверка уникальности имени эксперимента
func (e *Experiment) CheckNameUniqueness(existingNames []string) bool {
	return !slices.Contains(existingNames, e.Name)
}
