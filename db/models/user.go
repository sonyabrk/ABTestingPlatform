package models

import "errors"

type User struct {
	ID           int    `db:"id" json:"id"`
	ExperinentId int    `db:"experiment_id" json:"experiment_id"`
	UserId       string `db:"user_id" json:"user_id"`
	GroupName    string `db:"group_name" json:"group_name"`
}

// возврат имени таблицы в БД
func (User) TableName() string {
	return "users"
}

// проверка корректности данных пользователя
func (u *User) Validate() error {
	if u.ID < 0 {
		return errors.New("айди не может быть отрицательным")
	}
	if u.ExperinentId < 0 {
		return errors.New("айди эксперимента не может быть отрицательным")
	}
	if u.UserId == "" {
		return errors.New("айди пользователя не может быть пустым")
	}

	if len(u.UserId) > 255 {
		return errors.New("айди пользователя не может слишком длинный")
	}
	if u.UserId == "" {
		return errors.New("название группы не может быть пустым")
	}

	if len(u.UserId) > 255 {
		return errors.New("название группы не может слишком длинным")
	}
	return nil
}

// методы для инкапсулиции логики проверки

// возвращение группы эксперимента
func (u *User) HasGroupName() string {
	return u.GroupName
}

// принадлежит ли пользователь к группе А
func (u *User) IsGroupA() bool {
	return u.GroupName == "A"
}

// принадлежит ли пользователь к группе В
func (u *User) IsGroupB() bool {
	return u.GroupName == "B"
}

// получение описания эксперимента
func (u *User) GetGroupDescription() string {
	if u.IsGroupA() {
		return "Группа A (тестовая)"
	} else {
		return "Группа B (тестовая)"
	}
}

// можно ли пользователю присвоить жксперимент
func (u *User) CanBeAssigned(experimentId int) bool {
	return u.UserId != "" && experimentId > 0
}

// проверка, имеет ли пользватель назначенный айди эксперимента
func (u *User) IsAssignedToExperiment(experimentID int) bool {
	return u.ExperinentId == experimentID
}
