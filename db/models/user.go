package models

import "errors"

type User struct {
	ID           int    `db:"id" json:"id"`
	ExperimentId int    `db:"experiment_id" json:"experiment_id"`
	UserId       string `db:"user_id" json:"user_id"`
	GroupName    string `db:"group_name" json:"group_name"`
}

// возврат имени таблицы в БД
func (User) TableName() string {
	return "users"
}

// проверка корректности данных пользователя
func (u *User) Validate() error {
	if u.ExperimentId < 0 {
		return errors.New("айди эксперимента не может быть отрицательным")
	}
	if u.UserId == "" {
		return errors.New("айди не может быть пустым")
	}
	if len(u.UserId) > 255 {
		return errors.New("айди не может слишком длинным")
	}
	if u.GroupName != "A" && u.GroupName != "B" {
		return errors.New("группа должна быть 'A' или 'B'")
	}
	return nil
}
