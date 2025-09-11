package models

import (
	"errors"
	"time"
)

type Result struct {
	ID               int       `db:"id" json:"id"`
	UserId           int       `db:"user_id" json:"user_id"`
	RecommendationId string    `db:"recommendation_id" json:"recommendation_id"`
	Clicked          bool      `db:"clicked" json:"clicked"`
	ClickedAt        time.Time `db:"clicked_at" json:"clicked_at"`
	Rating           int       `db:"rating" json:"rating"`
	//User             *User      `db:"-" json:"user,omitempty"`
}

// возврат имени таблицы в БД
func (Result) TableName() string {
	return "results"
}

// проверка корректности данных результата
func (r *Result) Validate() error {
	if r.ID < 0 {
		return errors.New("айди не может быть отрицательным")
	}
	if r.UserId <= 0 {
		return errors.New("айди пользователя не может быть не положительным")
	}
	if r.RecommendationId == "" {
		return errors.New("ID не может быть пустым")
	}
	if len(r.RecommendationId) > 255 {
		return errors.New("айди рекомедуемого эксперимента слишком длинное")
	}
	if r.Rating > 5 || r.Rating < 0 {
		return errors.New("рейтинг должен быть от 0 до 5")
	}
	if r.Rating > 0 && !r.Clicked {
		return errors.New("нельзя поставить рейтинг без клика")
	}
	if !r.Clicked && r.ClickedAt.IsZero() {
		return errors.New("время клика не может быть указано без самого клика")
	}
	if r.Clicked && r.ClickedAt.IsZero() {
		return errors.New("при наличии клика должно быть указано время клика")
	}
	return nil
}

// методы для инкапсулиции логики проверки

// возвращение оценки алгоритма
func (r *Result) HasRating() bool {
	return r.Rating > 0
}

// проверка на положительную оценку рейтинга
func (r *Result) IsPositiveRating() bool {
	return r.Rating > 3
}

// проверка рейтинга
func (r *Result) IsValidRating() bool {
	return r.Rating >= 1 && r.Rating <= 5
}

// разбиение рейтинга на категории
func (r *Result) GetResultCategory() string {
	switch r.Rating {
	case 0:
		return "negative"
	case 1:
		return "negative"
	case 2:
		return "negative"
	case 3:
		return "neutral"
	case 4:
		return "positive"
	case 5:
		return "positive"
	default:
		return "no rating"
	}
}

// проверка клика за последние 24 часа
func (r *Result) WasClickedRecently() bool {
	if r.ClickedAt.IsZero() {
		return false
	}
	return time.Since(r.ClickedAt) <= 24*time.Hour
}
