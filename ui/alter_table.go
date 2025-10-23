package ui

import (
	"context"
	"fmt"
	"strings"
	"testing-platform/db"
	"testing-platform/pkg/logger"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

type AlterTableWindow struct {
	window         fyne.Window
	repository     *db.Repository
	mainWindow     fyne.Window
	onTableChanged func() // Callback для обновления главного окна

	// Элементы управления
	tableSelect     *widget.Select
	actionSelect    *widget.Select
	columnName      *widget.Entry
	newColumnName   *widget.Entry
	dataType        *widget.Select
	constraintType  *widget.Select
	constraintValue *widget.Entry
	defaultValue    *widget.Entry
	nullableCheck   *widget.Check
	referenceTable  *widget.Entry
	referenceColumn *widget.Entry
	resultLabel     *widget.Label

	// Контейнеры для группировки полей
	columnNameContainer      *fyne.Container
	newColumnNameContainer   *fyne.Container
	dataTypeContainer        *fyne.Container
	constraintTypeContainer  *fyne.Container
	constraintValueContainer *fyne.Container
	defaultValueContainer    *fyne.Container
	nullableCheckContainer   *fyne.Container
	referenceContainer       *fyne.Container

	currentTable string
}

func NewAlterTableWindow(repo *db.Repository, mainWindow fyne.Window, onTableChanged func()) *AlterTableWindow {
	a := &AlterTableWindow{
		repository:     repo,
		mainWindow:     mainWindow,
		onTableChanged: onTableChanged,
		window:         fyne.CurrentApp().NewWindow("ALTER TABLE - Изменение структуры таблиц"),
	}

	a.buildUI()
	a.loadTables()
	return a
}

func (a *AlterTableWindow) buildUI() {
	// Выбор таблицы
	a.tableSelect = widget.NewSelect([]string{}, a.onTableSelected)
	a.tableSelect.PlaceHolder = "Выберите таблицу"

	// Выбор действия
	a.actionSelect = widget.NewSelect([]string{
		"Добавить столбец",
		"Удалить столбец",
		"Переименовать столбец",
		"Изменить тип данных",
		"Добавить ограничение",
		"Удалить ограничение",
		"Переименовать таблицу",
	}, a.onActionSelected)
	a.actionSelect.PlaceHolder = "Выберите действие"

	// Поля ввода
	a.columnName = widget.NewEntry()
	a.columnName.SetPlaceHolder("Имя столбца")

	a.newColumnName = widget.NewEntry()
	a.newColumnName.SetPlaceHolder("Новое имя")

	a.dataType = widget.NewSelect([]string{
		"INTEGER", "SERIAL", "BIGINT", "VARCHAR(255)", "TEXT", "BOOLEAN",
		"DATE", "TIMESTAMP", "NUMERIC(10,2)", "JSONB",
	}, nil)
	a.dataType.PlaceHolder = "Тип данных"

	a.constraintType = widget.NewSelect([]string{
		"PRIMARY KEY", "FOREIGN KEY", "UNIQUE", "CHECK", "NOT NULL",
	}, a.onConstraintTypeChanged)
	a.constraintType.PlaceHolder = "Тип ограничения"

	a.constraintValue = widget.NewEntry()
	a.constraintValue.SetPlaceHolder("Условие CHECK")

	a.defaultValue = widget.NewEntry()
	a.defaultValue.SetPlaceHolder("Значение по умолчанию")

	a.nullableCheck = widget.NewCheck("Разрешить NULL", nil)
	a.nullableCheck.SetChecked(true)

	a.referenceTable = widget.NewEntry()
	a.referenceTable.SetPlaceHolder("Таблица для ссылки")

	a.referenceColumn = widget.NewEntry()
	a.referenceColumn.SetPlaceHolder("Столбец для ссылки")

	a.resultLabel = widget.NewLabel("")
	a.resultLabel.Wrapping = fyne.TextWrapWord

	// Создаем контейнеры для группировки полей
	a.columnNameContainer = container.NewVBox(
		widget.NewLabel("Столбец:"),
		a.columnName,
	)

	a.newColumnNameContainer = container.NewVBox(
		widget.NewLabel("Новое имя:"),
		a.newColumnName,
	)

	a.dataTypeContainer = container.NewVBox(
		widget.NewLabel("Тип данных:"),
		a.dataType,
	)

	a.constraintTypeContainer = container.NewVBox(
		widget.NewLabel("Тип ограничения:"),
		a.constraintType,
	)

	a.constraintValueContainer = container.NewVBox(
		widget.NewLabel("Условие:"),
		a.constraintValue,
	)

	a.defaultValueContainer = container.NewVBox(
		widget.NewLabel("Значение по умолчанию:"),
		a.defaultValue,
	)

	a.nullableCheckContainer = container.NewVBox(
		a.nullableCheck,
	)

	a.referenceContainer = container.NewVBox(
		widget.NewLabel("Ссылка (FOREIGN KEY):"),
		a.referenceTable,
		a.referenceColumn,
	)

	// Кнопки
	applyBtn := widget.NewButton("Применить изменения", a.applyChanges)
	previewBtn := widget.NewButton("Показать SQL", a.showSQL)
	refreshBtn := widget.NewButton("Обновить", a.refreshData)
	closeBtn := widget.NewButton("Закрыть", func() { a.window.Close() })

	// Основная компоновка
	form := container.NewVBox(
		widget.NewLabel("Таблица:"),
		a.tableSelect,
		widget.NewLabel("Действие:"),
		a.actionSelect,
		a.columnNameContainer,
		a.newColumnNameContainer,
		a.dataTypeContainer,
		a.constraintTypeContainer,
		a.constraintValueContainer,
		a.defaultValueContainer,
		a.nullableCheckContainer,
		a.referenceContainer,
		container.NewHBox(applyBtn, previewBtn, refreshBtn, closeBtn),
		a.resultLabel,
	)

	a.window.SetContent(container.NewScroll(form))
	a.window.Resize(fyne.NewSize(600, 600))

	// Скрываем все дополнительные поля при запуске
	a.hideAllContainers()
}

func (a *AlterTableWindow) hideAllContainers() {
	a.columnNameContainer.Hide()
	a.newColumnNameContainer.Hide()
	a.dataTypeContainer.Hide()
	a.constraintTypeContainer.Hide()
	a.constraintValueContainer.Hide()
	a.defaultValueContainer.Hide()
	a.nullableCheckContainer.Hide()
	a.referenceContainer.Hide()
}

func (a *AlterTableWindow) refreshData() {
	a.loadTables()
	a.resultLabel.SetText("Данные обновлены")

	// Вызываем callback для обновления главного окна
	if a.onTableChanged != nil {
		a.onTableChanged()
	}
}

func (a *AlterTableWindow) loadTables() {
	tables, err := a.repository.GetTables(context.Background())
	if err != nil {
		a.showError(err)
		return
	}

	// Сохраняем текущее выделение
	currentSelection := a.tableSelect.Selected

	a.tableSelect.Options = tables
	a.tableSelect.Refresh()

	// Восстанавливаем выделение, если таблица еще существует
	if currentSelection != "" {
		for _, table := range tables {
			if table == currentSelection {
				a.tableSelect.SetSelected(currentSelection)
				break
			}
		}
	}
}

func (a *AlterTableWindow) onTableSelected(table string) {
	a.currentTable = table
	a.resultLabel.SetText(fmt.Sprintf("Выбрана таблица: %s", table))
}

func (a *AlterTableWindow) onActionSelected(action string) {
	// Скрываем все поля сначала
	a.hideAllContainers()

	// Очищаем поля
	a.columnName.SetText("")
	a.newColumnName.SetText("")
	a.dataType.SetSelected("")
	a.constraintType.SetSelected("")
	a.constraintValue.SetText("")
	a.defaultValue.SetText("")
	a.referenceTable.SetText("")
	a.referenceColumn.SetText("")
	a.nullableCheck.SetChecked(true)

	switch action {
	case "Добавить столбец":
		a.columnNameContainer.Show()
		a.dataTypeContainer.Show()
		a.defaultValueContainer.Show()
		a.nullableCheckContainer.Show()
		a.resultLabel.SetText("Введите данные для нового столбца")

	case "Удалить столбец":
		a.columnNameContainer.Show()
		a.resultLabel.SetText("Выберите столбец для удаления")

	case "Переименовать столбец":
		a.columnNameContainer.Show()
		a.newColumnNameContainer.Show()
		a.resultLabel.SetText("Введите старое и новое имя столбца")

	case "Изменить тип данных":
		a.columnNameContainer.Show()
		a.dataTypeContainer.Show()
		a.resultLabel.SetText("Выберите столбец и новый тип данных")

	case "Добавить ограничение":
		a.columnNameContainer.Show()
		a.constraintTypeContainer.Show()
		a.resultLabel.SetText("Выберите тип ограничения")

	case "Удалить ограничение":
		a.constraintValueContainer.Show()
		a.constraintValue.SetPlaceHolder("Имя ограничения")
		a.resultLabel.SetText("Введите имя ограничения для удаления")

	case "Переименовать таблицу":
		a.newColumnNameContainer.Show()
		a.newColumnName.SetPlaceHolder("Новое имя таблицы")
		a.resultLabel.SetText("Введите новое имя таблицы")
	}
}

func (a *AlterTableWindow) onConstraintTypeChanged(constraintType string) {
	// Показываем/скрываем дополнительные поля в зависимости от типа ограничения
	a.constraintValueContainer.Hide()
	a.referenceContainer.Hide()

	switch constraintType {
	case "CHECK":
		a.constraintValueContainer.Show()
		a.constraintValue.SetPlaceHolder("Условие CHECK (например: salary > 0)")
	case "FOREIGN KEY":
		a.referenceContainer.Show()
	case "NOT NULL":
		// Для NOT NULL не нужны дополнительные поля
	default:
		// Для PRIMARY KEY, UNIQUE не нужны дополнительные поля
	}
}

func (a *AlterTableWindow) validateInput() error {
	if a.currentTable == "" {
		return fmt.Errorf("не выбрана таблица")
	}

	action := a.actionSelect.Selected
	if action == "" {
		return fmt.Errorf("не выбрано действие")
	}

	switch action {
	case "Добавить столбец":
		if strings.TrimSpace(a.columnName.Text) == "" {
			return fmt.Errorf("не указано имя столбца")
		}
		if a.dataType.Selected == "" {
			return fmt.Errorf("не выбран тип данных")
		}

	case "Удалить столбец":
		if strings.TrimSpace(a.columnName.Text) == "" {
			return fmt.Errorf("не указано имя столбца")
		}

	case "Переименовать столбец":
		if strings.TrimSpace(a.columnName.Text) == "" {
			return fmt.Errorf("не указано текущее имя столбца")
		}
		if strings.TrimSpace(a.newColumnName.Text) == "" {
			return fmt.Errorf("не указано новое имя столбца")
		}

	case "Изменить тип данных":
		if strings.TrimSpace(a.columnName.Text) == "" {
			return fmt.Errorf("не указано имя столбца")
		}
		if a.dataType.Selected == "" {
			return fmt.Errorf("не выбран тип данных")
		}

	case "Добавить ограничение":
		if a.constraintType.Selected == "" {
			return fmt.Errorf("не выбран тип ограничения")
		}
		if strings.TrimSpace(a.columnName.Text) == "" && a.constraintType.Selected != "CHECK" {
			return fmt.Errorf("не указано имя столбца")
		}

		// Валидация для конкретных типов ограничений
		switch a.constraintType.Selected {
		case "FOREIGN KEY":
			if strings.TrimSpace(a.referenceTable.Text) == "" {
				return fmt.Errorf("для FOREIGN KEY не указана таблица для ссылки")
			}
			if strings.TrimSpace(a.referenceColumn.Text) == "" {
				return fmt.Errorf("для FOREIGN KEY не указан столбец для ссылки")
			}
		case "CHECK":
			if strings.TrimSpace(a.constraintValue.Text) == "" {
				return fmt.Errorf("для CHECK не указано условие")
			}
		}

	case "Удалить ограничение":
		if strings.TrimSpace(a.constraintValue.Text) == "" {
			return fmt.Errorf("не указано имя ограничения")
		}

	case "Переименовать таблицу":
		if strings.TrimSpace(a.newColumnName.Text) == "" {
			return fmt.Errorf("не указано новое имя таблицы")
		}
	}

	return nil
}

func (a *AlterTableWindow) buildAlterQuery() (string, error) {
	if err := a.validateInput(); err != nil {
		return "", err
	}

	action := a.actionSelect.Selected
	var query string

	switch action {
	case "Добавить столбец":
		query = fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s",
			a.currentTable, a.columnName.Text, a.dataType.Selected)

		if a.defaultValue.Text != "" {
			query += " DEFAULT " + a.defaultValue.Text
		}
		if !a.nullableCheck.Checked {
			query += " NOT NULL"
		}

	case "Удалить столбец":
		query = fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s",
			a.currentTable, a.columnName.Text)

	case "Переименовать столбец":
		query = fmt.Sprintf("ALTER TABLE %s RENAME COLUMN %s TO %s",
			a.currentTable, a.columnName.Text, a.newColumnName.Text)

	case "Изменить тип данных":
		query = fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s TYPE %s",
			a.currentTable, a.columnName.Text, a.dataType.Selected)

	case "Добавить ограничение":
		constraintName := fmt.Sprintf("%s_%s_%s",
			a.currentTable, a.columnName.Text, strings.ToLower(strings.ReplaceAll(a.constraintType.Selected, " ", "_")))

		switch a.constraintType.Selected {
		case "PRIMARY KEY":
			query = fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s PRIMARY KEY (%s)",
				a.currentTable, constraintName, a.columnName.Text)
		case "FOREIGN KEY":
			query = fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s(%s)",
				a.currentTable, constraintName, a.columnName.Text, a.referenceTable.Text, a.referenceColumn.Text)
		case "UNIQUE":
			query = fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s UNIQUE (%s)",
				a.currentTable, constraintName, a.columnName.Text)
		case "CHECK":
			query = fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s CHECK (%s)",
				a.currentTable, constraintName, a.constraintValue.Text)
		case "NOT NULL":
			query = fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET NOT NULL",
				a.currentTable, a.columnName.Text)
		}

	case "Удалить ограничение":
		query = fmt.Sprintf("ALTER TABLE %s DROP CONSTRAINT %s",
			a.currentTable, a.constraintValue.Text)

	case "Переименовать таблицу":
		query = fmt.Sprintf("ALTER TABLE %s RENAME TO %s",
			a.currentTable, a.newColumnName.Text)
	}

	return query, nil
}

func (a *AlterTableWindow) executeQuery(query string) {
    err := a.repository.ExecuteAlter(context.Background(), query)
    if err != nil {
        a.showError(err)
        return
    }

    // Обновляем интерфейс
    a.refreshData()
    a.resultLabel.SetText("✅ Изменения успешно применены!\nSQL: " + query)

    // Специальная обработка для переименования таблицы
    if a.actionSelect.Selected == "Переименовать таблицу" {
        a.currentTable = a.newColumnName.Text
    }

    // Очищаем поля
    a.columnName.SetText("")
    a.newColumnName.SetText("")
    a.constraintValue.SetText("")
    a.defaultValue.SetText("")
    a.referenceTable.SetText("")
    a.referenceColumn.SetText("")

    // ВАЖНО: Вызываем callback для обновления всех окон данных
    logger.Info("Вызов callback onTableChanged. Функция установлена: %t", a.onTableChanged != nil)
    if a.onTableChanged != nil {
        a.onTableChanged()
    } else {
        logger.Error("Callback onTableChanged НЕ установлен!")
    }
}

// func (a *AlterTableWindow) executeQuery(query string) {
// 	err := a.repository.ExecuteAlter(context.Background(), query)
// 	if err != nil {
// 		a.showError(err)
// 		return
// 	}

// 	// Обновляем интерфейс
// 	a.refreshData()
// 	a.resultLabel.SetText("✅ Изменения успешно применены!\nSQL: " + query)

// 	// Специальная обработка для переименования таблицы
// 	if a.actionSelect.Selected == "Переименовать таблицу" {
// 		a.currentTable = a.newColumnName.Text
// 	}

// 	// Очищаем поля
// 	a.columnName.SetText("")
// 	a.newColumnName.SetText("")
// 	a.constraintValue.SetText("")
// 	a.defaultValue.SetText("")
// 	a.referenceTable.SetText("")
// 	a.referenceColumn.SetText("")

// 	// ВАЖНО: Вызываем callback для обновления всех окон данных
// 	if a.onTableChanged != nil {
// 		logger.Info("Вызов callback для обновления интерфейса")
// 		a.onTableChanged()
// 	} else {
// 		logger.Error("Callback onTableChanged не установлен!")
// 	}
// }

func (a *AlterTableWindow) applyChanges() {
	query, err := a.buildAlterQuery()
	if err != nil {
		a.showError(err)
		return
	}

	// Подтверждение для деструктивных операций
	action := a.actionSelect.Selected
	if action == "Удалить столбец" || action == "Удалить ограничение" {
		dialog.ShowConfirm("Подтверждение",
			fmt.Sprintf("Вы уверены, что хотите выполнить действие: %s?\n\nSQL: %s", action, query),
			func(confirmed bool) {
				if confirmed {
					a.executeQuery(query)
				}
			}, a.window)
	} else {
		a.executeQuery(query)
	}
}

func (a *AlterTableWindow) showSQL() {
	query, err := a.buildAlterQuery()
	if err != nil {
		a.showError(err)
		return
	}

	a.resultLabel.SetText("SQL запрос:\n" + query)
}

func (a *AlterTableWindow) showError(err error) {
	dialog.ShowError(err, a.window)
	a.resultLabel.SetText("❌ Ошибка: " + err.Error())
}

func (a *AlterTableWindow) Show() {
	a.window.Show()
}
