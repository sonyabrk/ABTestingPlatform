package ui

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"testing-platform/db/models"
	"testing-platform/pkg/logger"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// вспомогательная функция для показа ошибок пользователю
func showUserError(win fyne.Window, msg string) {
	dialog.ShowError(fmt.Errorf("%s", msg), win)
}

// обработчик кнопки "Создать схему и таблицы"
func (mw *MainWindow) createSchemaHandler() {
    ctx := context.Background()
    
    // Сначала пытаемся починить миграции, если есть проблемы
    if err := mw.rep.FixMigrations(ctx); err != nil {
        logger.Error("Ошибка исправления миграций: %v", err)
        // Продолжаем с созданием схемы, возможно, она уже создана
    }
    
    err := mw.rep.CreateSchema(ctx)
    if err != nil {
        logger.Error("Ошибка создания схемы: %v", err)
        showUserError(mw.window, "Не удалось создать схему БД: проверьте права доступа и соединение с базой данных")
    } else {
        dialog.ShowInformation("Успех", "Схема БД успешно создана", mw.window)
        logger.Info("Схема БД успешно создана")
    }
}

// создание модального окна для ввода данных с вкладками
func (mw *MainWindow) showDataInputDialog() {
	tabs := container.NewAppTabs(
		container.NewTabItem("Эксперимент", mw.createExperimentForm()),
		container.NewTabItem("Пользователь", mw.createUserForm()),
		container.NewTabItem("Результат", mw.createResultForm()),
	)
	dialog.ShowCustom("Внести данные", "Закрыть", tabs, mw.window)
}

// создание формы для ввода данных эксперимента
func (mw *MainWindow) createExperimentForm() *widget.Form {
	name := widget.NewEntry()
	name.SetPlaceHolder("Введите название эксперимента")
	algorithmA := widget.NewSelect([]string{"collaborative", "content_based", "hybrid", "popularity_based"}, nil)
	algorithmB := widget.NewSelect([]string{"collaborative", "content_based", "hybrid", "popularity_based"}, nil)
	userPercent := widget.NewEntry()
	userPercent.SetPlaceHolder("Например: 10.5")
	isActive := widget.NewCheck("Активный эксперимент", nil)
	tagsEntry := widget.NewEntry()
	tagsEntry.SetPlaceHolder("Например: тест, рекомендации, основной")

	nameHint := widget.NewLabel("Обязательное поле, максимум 255 символов")
	nameHint.TextStyle = fyne.TextStyle{Italic: true}

	algorithmHint := widget.NewLabel("Выберите два разных алгоритма из списка")
	algorithmHint.TextStyle = fyne.TextStyle{Italic: true}

	userPercentHint := widget.NewLabel("Число от 0.1 до 100 (положительное)")
	userPercentHint.TextStyle = fyne.TextStyle{Italic: true}

	tagsHint := widget.NewLabel("Теги через запятую (каждый до 50 символов)")
	tagsHint.TextStyle = fyne.TextStyle{Italic: true}

	// ошибки
	nameError := widget.NewLabel("")
	nameError.Hide()
	userPercentError := widget.NewLabel("")
	userPercentError.Hide()
	tagsError := widget.NewLabel("")
	tagsError.Hide()

	userPercent.Validator = func(s string) error {
		if s == "" {
			return fmt.Errorf("поле обязательно для заполнения")
		}

		s = strings.TrimSpace(s)
		if s == "" {
			return fmt.Errorf("поле не может содержать только пробелы")
		}

		if strings.Count(s, ".") > 1 {
			return fmt.Errorf("некорректный формат числа (слишком много точек)")
		}

		if strings.HasPrefix(s, "-") {
			return fmt.Errorf("число должно быть положительным")
		}

		val, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return fmt.Errorf("значение должно быть числом (например: 10.5)")
		}

		if math.IsNaN(val) || math.IsInf(val, 0) {
			return fmt.Errorf("введено недопустимое числовое значение")
		}

		if val <= 0 {
			return fmt.Errorf("число должно быть больше 0")
		}

		if val < 0.1 {
			return fmt.Errorf("число должно быть не менее 0.1")
		}

		if val > 100.0 {
			return fmt.Errorf("число должно быть не более 100")
		}

		parts := strings.Split(s, ".")
		if len(parts) == 2 && len(parts[1]) > 2 {
			return fmt.Errorf("можно использовать не более 2 знаков после запятой")
		}
		return nil
	}

	userPercent.OnChanged = func(s string) {
		if err := userPercent.Validator(s); err != nil {
			userPercentError.SetText(err.Error())
			userPercentError.Show()
		} else {
			userPercentError.Hide()
		}
	}

	name.Validator = func(s string) error {
		s = strings.TrimSpace(s)
		if s == "" {
			return fmt.Errorf("поле обязательно для заполнения")
		}

		if len(s) > 255 {
			return fmt.Errorf("название слишком длинное (максимум 255 символов)")
		}

		if matched, _ := regexp.MatchString(`[<>'"\\]`, s); matched {
			return fmt.Errorf("название содержит недопустимые символы")
		}
		return nil
	}

	name.OnChanged = func(s string) {
		if err := name.Validator(s); err != nil {
			nameError.SetText(err.Error())
			nameError.Show()
		} else {
			nameError.Hide()
		}
	}

	var tagRegex = regexp.MustCompile(`^[a-zA-Zа-яА-Я0-9_\-\s]+$`)

	validateTags := func(s string) error {
		if s == "" {
			return nil
		}

		s = strings.TrimSpace(s)
		if s == "" {
			return nil
		}

		tags := parseTags(s)

		if len(tags) > 10 {
			return fmt.Errorf("слишком много тегов (максимум 10)")
		}

		for _, tag := range tags {
			tag = strings.TrimSpace(tag)
			if tag == "" {
				return fmt.Errorf("тег не может быть пустым")
			}

			if len(tag) > 50 {
				return fmt.Errorf("тег '%s' слишком длинный (максимум 50 символов)", tag)
			}

			if !tagRegex.MatchString(tag) {
				return fmt.Errorf("тег '%s' содержит недопустимые символы. Разрешены только буквы, цифры, пробелы, дефисы и подчеркивания", tag)
			}
		}
		return nil
	}

	tagsEntry.OnChanged = func(s string) {
		if err := validateTags(s); err != nil {
			tagsError.SetText(err.Error())
			tagsError.Show()
		} else {
			tagsError.Hide()
		}
	}

	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Название", Widget: container.NewVBox(name, nameError)},
			{Text: "Алгоритм A", Widget: algorithmA},
			{Text: "Алгоритм B", Widget: algorithmB},
			{Text: "Процент пользователей", Widget: container.NewVBox(userPercent, userPercentError)},
			{Text: "Статус", Widget: isActive},
			{Text: "Теги", Widget: container.NewVBox(tagsEntry, tagsError)},
		},
		OnSubmit: func() {
			if err := name.Validator(name.Text); err != nil {
				showUserError(mw.window, "Ошибка в названии: "+err.Error())
				return
			}
			if err := userPercent.Validator(userPercent.Text); err != nil {
				showUserError(mw.window, "Ошибка в проценте пользователей: "+err.Error())
				return
			}
			if err := validateTags(tagsEntry.Text); err != nil {
				showUserError(mw.window, "Ошибка в тегах: "+err.Error())
				return
			}
			if algorithmA.Selected == "" || algorithmB.Selected == "" {
				showUserError(mw.window, "Выберите оба алгоритма")
				return
			}
			if algorithmA.Selected == algorithmB.Selected {
				showUserError(mw.window, "Алгоритмы не могут быть одинаковыми")
				return
			}

			userPercentVal, _ := strconv.ParseFloat(userPercent.Text, 64)
			tags := parseTags(tagsEntry.Text)

			exp := &models.Experiment{
				Name:        name.Text,
				AlgorithmA:  algorithmA.Selected,
				AlgorithmB:  algorithmB.Selected,
				UserPercent: userPercentVal,
				IsActive:    isActive.Checked,
				Tags:        tags,
			}

			ctx := context.Background()
			err := mw.rep.CreateExperiment(ctx, exp)
			if err != nil {
				logger.Error("Ошибка создания эксперимента: %v", err)
				showUserError(mw.window, "Не удалось создать эксперимент: проверьте корректность данных и соединение с БД")
			} else {
				dialog.ShowInformation("Успех", fmt.Sprintf("Эксперимент создан (ID: %d)", exp.ID), mw.window)
				logger.Info("Эксперимент '%s' успешно создан", exp.Name)
			}
		},
	}
	return form
}

// создание формы для добавления пользователя в эксперимент
func (mw *MainWindow) createUserForm() *widget.Form {
	experimentId := widget.NewEntry()
	experimentId.SetPlaceHolder("Например: 1")

	userId := widget.NewEntry()
	userId.SetPlaceHolder("Например: user_123")

	groupName := widget.NewSelect([]string{"A", "B"}, nil)

	experimentIdHint := widget.NewLabel("Целое положительное число")
	experimentIdHint.TextStyle = fyne.TextStyle{Italic: true}

	userIdHint := widget.NewLabel("Только буквы, цифры, дефисы и подчеркивания")
	userIdHint.TextStyle = fyne.TextStyle{Italic: true}

	groupHint := widget.NewLabel("Группа A или B для A/B тестирования")
	groupHint.TextStyle = fyne.TextStyle{Italic: true}

	experimentIdError := widget.NewLabel("")
	experimentIdError.Hide()
	userIdError := widget.NewLabel("")
	userIdError.Hide()

	experimentId.Validator = func(s string) error {
		s = strings.TrimSpace(s)
		if s == "" {
			return fmt.Errorf("поле обязательно для заполнения")
		}

		if strings.HasPrefix(s, "-") {
			return fmt.Errorf("число должно быть положительным")
		}

		val, err := strconv.Atoi(s)
		if err != nil {
			return fmt.Errorf("должно быть целым числом (например: 42)")
		}

		if val <= 0 {
			return fmt.Errorf("значение должно быть положительным числом")
		}

		if val > 1000000 {
			return fmt.Errorf("значение слишком большое (максимум 1000000)")
		}
		return nil
	}
	experimentId.OnChanged = func(s string) {
		if err := experimentId.Validator(s); err != nil {
			experimentIdError.SetText(err.Error())
			experimentIdError.Show()
		} else {
			experimentIdError.Hide()
		}
	}

	userId.Validator = func(s string) error {
		s = strings.TrimSpace(s)
		if s == "" {
			return fmt.Errorf("поле обязательно для заполнения")
		}

		if len(s) > 255 {
			return fmt.Errorf("ID пользователя слишком длинный (максимум 255 символов)")
		}

		if matched, _ := regexp.MatchString(`^[a-zA-Z0-9_\-]+$`, s); !matched {
			return fmt.Errorf("ID пользователя может содержать только буквы, цифры, дефисы и подчеркивания")
		}
		return nil
	}
	userId.OnChanged = func(s string) {
		if err := userId.Validator(s); err != nil {
			userIdError.SetText(err.Error())
			userIdError.Show()
		} else {
			userIdError.Hide()
		}
	}

	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "ID эксперимента", Widget: container.NewVBox(experimentId, experimentIdError)},
			{Text: "ID пользователя", Widget: container.NewVBox(userId, userIdError)},
			{Text: "Группа", Widget: groupName},
		},
		OnSubmit: func() {
			if err := experimentId.Validator(experimentId.Text); err != nil {
				showUserError(mw.window, "Ошибка в ID эксперимента: "+err.Error())
				return
			}
			if err := userId.Validator(userId.Text); err != nil {
				showUserError(mw.window, "Ошибка в ID пользователя: "+err.Error())
				return
			}
			if groupName.Selected == "" {
				showUserError(mw.window, "Выберите группу (A или B)")
				return
			}

			experimentIdVal, _ := strconv.Atoi(experimentId.Text)

			// проверяем наличие эксперимента
			ctx := context.Background()
			exists, err := mw.rep.ExperimentExists(ctx, experimentIdVal)
			if err != nil {
				logger.Error("Ошибка проверки эксперимента: %v", err)
				showUserError(mw.window, "Ошибка при проверке ID эксперимента: проверьте соединение с БД")
				return
			}
			if !exists {
				showUserError(mw.window, fmt.Sprintf("Эксперимент с ID %d не найден", experimentIdVal))
				return
			}

			user := &models.User{
				ExperimentId: experimentIdVal,
				UserId:       userId.Text,
				GroupName:    groupName.Selected,
			}

			err = mw.rep.AddUserToExperiment(ctx, user)
			if err != nil {
				logger.Error("Ошибка добавления пользователя: %v", err)
				showUserError(mw.window, "Не удалось добавить пользователя: проверьте корректность данных и соединение с БД")
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
	userId := widget.NewEntry()
	userId.SetPlaceHolder("Например: 1")

	recommendationId := widget.NewEntry()
	recommendationId.SetPlaceHolder("Например: rec_456")

	clicked := widget.NewCheck("Кликнут", nil)

	rating := widget.NewEntry()
	rating.SetPlaceHolder("0-5")

	userIdHint := widget.NewLabel("Целое положительное число")
	userIdHint.TextStyle = fyne.TextStyle{Italic: true}

	recommendationIdHint := widget.NewLabel("Только буквы, цифры, дефисы и подчеркивания")
	recommendationIdHint.TextStyle = fyne.TextStyle{Italic: true}

	ratingHint := widget.NewLabel("Целое число от 0 до 5 (обязательно при клике)")
	ratingHint.TextStyle = fyne.TextStyle{Italic: true}

	userIdError := widget.NewLabel("")
	userIdError.Hide()
	recommendationIdError := widget.NewLabel("")
	recommendationIdError.Hide()
	ratingError := widget.NewLabel("")
	ratingError.Hide()

	userId.Validator = func(s string) error {
		s = strings.TrimSpace(s)
		if s == "" {
			return fmt.Errorf("поле обязательно для заполнения")
		}

		if strings.HasPrefix(s, "-") {
			return fmt.Errorf("число должно быть положительным")
		}

		val, err := strconv.Atoi(s)
		if err != nil {
			return fmt.Errorf("должно быть целым числом (например: 42)")
		}

		if val <= 0 {
			return fmt.Errorf("значение должно быть положительным числом")
		}

		if val > 1000000 {
			return fmt.Errorf("значение слишком большое (максимум 1000000)")
		}
		return nil
	}
	userId.OnChanged = func(s string) {
		if err := userId.Validator(s); err != nil {
			userIdError.SetText(err.Error())
			userIdError.Show()
		} else {
			userIdError.Hide()
		}
	}

	recommendationId.Validator = func(s string) error {
		s = strings.TrimSpace(s)
		if s == "" {
			return fmt.Errorf("поле обязательно для заполнения")
		}

		if len(s) > 255 {
			return fmt.Errorf("ID рекомендации слишком длинный (максимум 255 символов)")
		}

		if matched, _ := regexp.MatchString(`^[a-zA-Z0-9_\-]+$`, s); !matched {
			return fmt.Errorf("ID рекомендации может содержать только буквы, цифры, дефисы и подчеркивания")
		}
		return nil
	}
	recommendationId.OnChanged = func(s string) {
		if err := recommendationId.Validator(s); err != nil {
			recommendationIdError.SetText(err.Error())
			recommendationIdError.Show()
		} else {
			recommendationIdError.Hide()
		}
	}

	rating.Validator = func(s string) error {
		s = strings.TrimSpace(s)

		if s == "" && clicked.Checked {
			return fmt.Errorf("рейтинг обязателен при клике")
		}

		if s != "" {
			if strings.HasPrefix(s, "-") {
				return fmt.Errorf("рейтинг не может быть отрицательным")
			}

			val, err := strconv.Atoi(s)
			if err != nil {
				return fmt.Errorf("рейтинг должен быть целым числом от 0 до 5")
			}

			if val < 0 || val > 5 {
				return fmt.Errorf("рейтинг должен быть от 0 до 5")
			}

			if clicked.Checked && (val < 0 || val > 5) {
				return fmt.Errorf("при клике рейтинг должен быть от 0 до 5")
			}
		}
		return nil
	}
	rating.OnChanged = func(s string) {
		if err := rating.Validator(s); err != nil {
			ratingError.SetText(err.Error())
			ratingError.Show()
		} else {
			ratingError.Hide()
		}
	}

	clicked.OnChanged = func(checked bool) {
		if !checked {
			rating.SetText("")
			rating.Disable()
		} else {
			rating.Enable()
		}
		rating.OnChanged(rating.Text)
	}
	if !clicked.Checked {
		rating.Disable()
	}

	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "ID пользователя", Widget: container.NewVBox(userId, userIdError)},
			{Text: "ID рекомендации", Widget: container.NewVBox(recommendationId, recommendationIdError)},
			{Text: "Кликнут", Widget: clicked},
			{Text: "Рейтинг", Widget: container.NewVBox(rating, ratingError)},
		},
		OnSubmit: func() {
			if err := userId.Validator(userId.Text); err != nil {
				showUserError(mw.window, "Ошибка в ID пользователя: "+err.Error())
				return
			}
			if err := recommendationId.Validator(recommendationId.Text); err != nil {
				showUserError(mw.window, "Ошибка в ID рекомендации: "+err.Error())
				return
			}
			if err := rating.Validator(rating.Text); err != nil {
				showUserError(mw.window, "Ошибка в рейтинге: "+err.Error())
				return
			}

			userIdVal, _ := strconv.Atoi(userId.Text)

			// проверяем наличие пользователя
			ctx := context.Background()
			exists, err := mw.rep.UserExists(ctx, userIdVal)
			if err != nil {
				logger.Error("Ошибка проверки пользователя: %v", err)
				showUserError(mw.window, "Ошибка при проверке ID пользователя: проверьте соединение с БД")
				return
			}
			if !exists {
				showUserError(mw.window, fmt.Sprintf("Пользователь с ID %d не найден", userIdVal))
				return
			}

			var ratingVal int
			if rating.Text != "" {
				ratingVal, _ = strconv.Atoi(rating.Text)
			}

			result := &models.Result{
				UserId:           userIdVal,
				RecommendationId: recommendationId.Text,
				Clicked:          clicked.Checked,
				ClickedAt:        nil,
				Rating:           ratingVal,
			}

			err = mw.rep.AddResult(ctx, result)
			if err != nil {
				logger.Error("Ошибка добавления результата: %v", err)
				showUserError(mw.window, "Не удалось добавить результат: проверьте корректность данных и соединение с БД")
			} else {
				dialog.ShowInformation("Успех", "Результат успешно добавлен", mw.window)
				logger.Info("Результат для пользователя %d успешно добавлен", result.UserId)
			}
		},
	}
	return form
}

func parseTags(tagsStr string) []string {
	if tagsStr == "" {
		return []string{}
	}
	tags := strings.Split(tagsStr, ",")
	for i, tag := range tags {
		tags[i] = strings.TrimSpace(tag)
	}
	return tags
}
