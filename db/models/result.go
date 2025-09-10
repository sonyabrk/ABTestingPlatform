package models

import (
	"errors"
	"time"
)

type Result struct {
	ID               int        `db:"id" json:"id"`
	UserId           int        `db:"user_id" json:"user_id"`
	RecommendationId string     `db:"recommendation_id" json:"recommendation_id"`
	Clicked          bool       `db:"clicked" json:"clicked"`
	ClickedAt        *time.Time `db:"clicked_at" json:"clicked_at"`
	Rating           int        `db:"rating" json:"rating"`
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
	if r.UserId < 0 {
		return errors.New("айди пользователя не может быть отрицательным")
	}
	if r.RecommendationId == "" {
		return errors.New("рекомендуемый алгоритм должен существовать")
	}
	if len(r.RecommendationId) > 255 {
		return errors.New("айди рекомедуемого эксперимента слишком длинное")
	}
	if r.ClickedAt != nil && r.Clicked == false {
		return errors.New("не может быть времени без клика")
	}
	if r.ClickedAt == nil && r.Clicked == true {
		return errors.New("не может быть клика без времени")
	}
	if r.Rating > 5 || r.Rating < 1 {
		return errors.New("рейтинг должен быть от 1 до 5")
	}
	return nil
}

// методы для инкапсулиции логики проверки

// возвращение оценки алгоритма
func (r *Result) HasRating() int {
	return r.Rating
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
	twentyFourHoursAgo := time.Now().Add(-24 * time.Hour)
	return r.ClickedAt.After(twentyFourHoursAgo)
}
