package models

import (
	"errors"
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
}

// возврат имени таблицы в БД
func (Experiment) TableName() string {
	return "experiments"
}

// проверка корректности данных эксперимента
func (e *Experiment) Validate() error {
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
	return nil
}

// методы (геттеры) ниже добавлены для инкапсуляции и предотвращения дублирования кода

// IsRunning проверяет, активен ли эксперимент
func (e *Experiment) IsRunning() bool {
	return e.IsActive
}

// GetAlgorithmNames возвращает названия алгоритмов для отображения
func (e *Experiment) GetAlgorithmNames() (string, string) {
	return e.AlgorithmA, e.AlgorithmB
}

// CanAcceptMoreUsers проверяет, можно ли добавлять пользователей в эксперимент
func (e *Experiment) CanAcceptMoreUsers(currentCount int) bool {
	return e.IsRunning() && currentCount < e.UserPercent*1000
}
