package models

import "time"

type Experiment struct {
	ID          int       `db:"id" json:"id"`
	Name        string    `db:"name" json:"name"`
	AlgorithmA  string    `db:"algorithm_a" json:"algorithm_a"`
	AlgorithmB  string    `db:"algorithm_b" json:"algorithm_b"`
	UserPercent int       `db:"user_percent" json:"user_percent"`
	StartDate   time.Time `db:"start_date" json:"start_date"`
	IsActive    bool      `db:"is_active" json:"is_active"`
}

func (Experiment) TableName() string {
	return "experiments"
}
