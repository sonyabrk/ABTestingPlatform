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

func (Result) TableName() string {
	return "results"
}

func (r *Result) Validate() error {
	if r.RecommendationId == "" {
		return errors.New("рекомендуемый алгоритм должен существовать")
	}
	if r.Rating > 5 || r.Rating < 1 {
		return errors.New("рейтинг должен быть от 1 до 5")
	}
	return nil
}
