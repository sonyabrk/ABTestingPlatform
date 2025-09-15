package ui

import (
	"context"
	"fmt"
	"testing-platform/db/models"
	"testing-platform/pkg/logger"

	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// обработчик кнопки "Создать схему и таблицы"
func (mw *MainWindow) createSchemaHandler() {
	// создание базового контекста
	ctx := context.Background()
	err := mw.rep.CreateSchema(ctx)
	if err != nil {
		logger.Error("Ошибка создания схемы: %v", err)
		dialog.ShowError(err, mw.window)
	} else {
		dialog.ShowInformation("Успех", "Схема БД успешно создана", mw.window)
		logger.Info("Схема БД успешно создана")
	}
}

// создание модального окна для ввода данных с вкладками
func (mw *MainWindow) showDataInputDialog() {
	// создание вкладок для разных типов данных
	tabs := container.NewAppTabs(
		container.NewTabItem("Эксперимент", mw.createExperimentForm()),
		container.NewTabItem("Пользователь", mw.createUserForm()),
		container.NewTabItem("Результат", mw.createResultForm()),
	)
	// размещение вкладок в верхней части окна
	tabs.SetTabLocation(container.TabLocationTop)
	dialog.ShowCustom("Внести данные", "Закрыть", tabs, mw.window)
}

// создание формы для ввода данных эксперимента
func (mw *MainWindow) createExperimentForm() *widget.Form {
	// элементы формы
	name := widget.NewEntry()
	algorithmA := widget.NewSelect([]string{"collaborative", "content_based", "hybrid", "popularity_based"}, nil)
	algorithmB := widget.NewSelect([]string{"collaborative", "content_based", "hybrid", "popularity_based"}, nil)
	userPercent := widget.NewEntry()
	isActive := widget.NewCheck("Активный эксперимент", nil)
	// форма с элементами
	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Название", Widget: name},
			{Text: "Алгоритм A", Widget: algorithmA},
			{Text: "Алгоритм B", Widget: algorithmB},
			{Text: "Процент пользователей", Widget: userPercent},
			{Text: "Статус", Widget: isActive},
		},
		OnSubmit: func() {
			// объект модели из данных формы
			exp := &models.Experiment{
				Name:        name.Text,
				AlgorithmA:  algorithmA.Selected,
				AlgorithmB:  algorithmB.Selected,
				UserPercent: parseInt(userPercent.Text),
				IsActive:    isActive.Checked,
			}

			ctx := context.Background()
			err := mw.rep.CreateExperiment(ctx, exp)
			if err != nil {
				logger.Error("Ошибка создания эксперимента: %v", err)
				dialog.ShowError(err, mw.window)
			} else {
				dialog.ShowInformation("Успех", fmt.Sprintf("Эксперимент создан с ID: %d", exp.ID), mw.window)
				logger.Info("Эксперимент '%s' успешно создан", exp.Name)
			}
		},
	}

	return form
}

// создание формы для добавления пользователя в эксперимент
func (mw *MainWindow) createUserForm() *widget.Form {
	// элементы формы для пользователя
	experimentId := widget.NewEntry()
	userId := widget.NewEntry()
	groupName := widget.NewSelect([]string{"A", "B"}, nil)

	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "ID эксперимента", Widget: experimentId},
			{Text: "ID пользователя", Widget: userId},
			{Text: "Группа", Widget: groupName},
		},
		OnSubmit: func() {
			// объект пользоватлея
			user := &models.User{
				ExperimentId: parseInt(experimentId.Text),
				UserId:       userId.Text,
				GroupName:    groupName.Selected,
			}

			ctx := context.Background()
			err := mw.rep.AddUserToExperiment(ctx, user)
			if err != nil {
				logger.Error("Ошибка добавления пользователя: %v", err)
				dialog.ShowError(err, mw.window)
			} else {
				dialog.ShowInformation("Успех", "Пользователь успешно добавлен", mw.window)
				logger.Info("Пользователь %s успешно добавлен", user.UserId)
			}
		},
	}
	return form
}

// создание формы для добавления результатов тестирования
func (mw *MainWindow) createResultForm() *widget.Form {
	// элементы формы для результатов
	userId := widget.NewEntry()
	recommendationId := widget.NewEntry()
	clicked := widget.NewCheck("Кликнут", nil)
	rating := widget.NewEntry()

	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "ID пользователя", Widget: userId},
			{Text: "ID рекомендации", Widget: recommendationId},
			{Text: "Кликнут", Widget: clicked},
			{Text: "Рейтинг", Widget: rating},
		},
		OnSubmit: func() {
			// объект результата
			result := &models.Result{
				UserId:           parseInt(userId.Text),
				RecommendationId: recommendationId.Text,
				Clicked:          clicked.Checked,
				Rating:           parseInt(rating.Text),
			}

			ctx := context.Background()
			err := mw.rep.AddResult(ctx, result)
			if err != nil {
				logger.Error("Ошибка добавления результата: %v", err)
				dialog.ShowError(err, mw.window)
			} else {
				dialog.ShowInformation("Успех", "Результат успешно добавлен", mw.window)
				logger.Info("Результат для пользователя %d успешно добавлен", result.UserId)
			}
		},
	}
	return form
}
