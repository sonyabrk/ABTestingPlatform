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
}

// возврат имени таблицы в БД
func (Result) TableName() string {
	return "results"
}

// проверка корректности данных результата
func (r *Result) Validate() error {
	if r.Rating < 0 || r.Rating > 5 {
		return errors.New("рейтинг должен быть от 0 до 5")
	}
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
	if r.Rating > 0 && !r.Clicked {
		return errors.New("нельзя поставить рейтинг без клика")
	}
	if !r.Clicked && r.ClickedAt != nil {
		return errors.New("время клика не может быть указано без самого клика")
	}
	return nil
}
