package ui

import (
	"context"
	"fmt"
	"strings"
	"testing-platform/db"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

type AlterTableWindow struct {
	window     fyne.Window
	repository *db.Repository
	mainWindow fyne.Window

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
	resultLabel     *widget.Label

	currentTable string
}

func NewAlterTableWindow(repo *db.Repository, mainWindow fyne.Window) *AlterTableWindow {
	a := &AlterTableWindow{
		repository: repo,
		mainWindow: mainWindow,
		window:     fyne.CurrentApp().NewWindow("ALTER TABLE - Изменение структуры таблиц"),
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
	}, nil)
	a.constraintType.PlaceHolder = "Тип ограничения"

	a.constraintValue = widget.NewEntry()
	a.constraintValue.SetPlaceHolder("Значение/условие")

	a.defaultValue = widget.NewEntry()
	a.defaultValue.SetPlaceHolder("Значение по умолчанию")

	a.nullableCheck = widget.NewCheck("NULL", nil)
	a.nullableCheck.SetChecked(true)

	a.resultLabel = widget.NewLabel("")
	a.resultLabel.Wrapping = fyne.TextWrapWord

	// Кнопки
	applyBtn := widget.NewButton("Применить изменения", a.applyChanges)
	previewBtn := widget.NewButton("Показать SQL", a.showSQL)
	closeBtn := widget.NewButton("Закрыть", func() { a.window.Close() })

	// Компоновка
	form := container.NewVBox(
		widget.NewLabel("Таблица:"),
		a.tableSelect,
		widget.NewLabel("Действие:"),
		a.actionSelect,
		widget.NewLabel("Столбец:"),
		a.columnName,
		widget.NewLabel("Новое имя:"),
		a.newColumnName,
		widget.NewLabel("Тип данных:"),
		a.dataType,
		widget.NewLabel("Ограничение:"),
		a.constraintType,
		widget.NewLabel("Значение ограничения:"),
		a.constraintValue,
		widget.NewLabel("Значение по умолчанию:"),
		a.defaultValue,
		a.nullableCheck,
		container.NewHBox(applyBtn, previewBtn, closeBtn),
		a.resultLabel,
	)

	a.window.SetContent(container.NewScroll(form))
	a.window.Resize(fyne.NewSize(600, 500))
}

func (a *AlterTableWindow) loadTables() {
	tables, err := a.repository.GetTables(context.Background())
	if err != nil {
		a.showError(err)
		return
	}
	a.tableSelect.Options = tables
	a.tableSelect.Refresh()
}

func (a *AlterTableWindow) onTableSelected(table string) {
	a.currentTable = table
	a.resultLabel.SetText(fmt.Sprintf("Выбрана таблица: %s", table))
}

func (a *AlterTableWindow) onActionSelected(action string) {
	// Можно добавить логику показа/скрытия полей в зависимости от действия
}

func (a *AlterTableWindow) buildAlterQuery() (string, error) {
	if a.currentTable == "" {
		return "", fmt.Errorf("не выбрана таблица")
	}

	action := a.actionSelect.Selected
	if action == "" {
		return "", fmt.Errorf("не выбрано действие")
	}

	var query string

	switch action {
	case "Добавить столбец":
		if a.columnName.Text == "" {
			return "", fmt.Errorf("не указано имя столбца")
		}
		if a.dataType.Selected == "" {
			return "", fmt.Errorf("не выбран тип данных")
		}

		query = fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s",
			a.currentTable, a.columnName.Text, a.dataType.Selected)

		if a.defaultValue.Text != "" {
			query += " DEFAULT " + a.defaultValue.Text
		}
		if !a.nullableCheck.Checked {
			query += " NOT NULL"
		}

	case "Удалить столбец":
		if a.columnName.Text == "" {
			return "", fmt.Errorf("не указано имя столбца")
		}
		query = fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s",
			a.currentTable, a.columnName.Text)

	case "Переименовать столбец":
		if a.columnName.Text == "" || a.newColumnName.Text == "" {
			return "", fmt.Errorf("не указаны старое и новое имена столбца")
		}
		query = fmt.Sprintf("ALTER TABLE %s RENAME COLUMN %s TO %s",
			a.currentTable, a.columnName.Text, a.newColumnName.Text)

	case "Изменить тип данных":
		if a.columnName.Text == "" {
			return "", fmt.Errorf("не указано имя столбца")
		}
		if a.dataType.Selected == "" {
			return "", fmt.Errorf("не выбран тип данных")
		}
		query = fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s TYPE %s",
			a.currentTable, a.columnName.Text, a.dataType.Selected)

	case "Добавить ограничение":
		if a.constraintType.Selected == "" {
			return "", fmt.Errorf("не выбран тип ограничения")
		}

		constraintName := fmt.Sprintf("%s_%s_%s",
			a.currentTable, a.columnName.Text, strings.ToLower(a.constraintType.Selected))

		switch a.constraintType.Selected {
		case "PRIMARY KEY":
			query = fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s PRIMARY KEY (%s)",
				a.currentTable, constraintName, a.columnName.Text)
		case "FOREIGN KEY":
			if a.constraintValue.Text == "" {
				return "", fmt.Errorf("для FOREIGN KEY укажите таблицу.столбец")
			}
			parts := strings.Split(a.constraintValue.Text, ".")
			if len(parts) != 2 {
				return "", fmt.Errorf("формат: таблица.столбец")
			}
			query = fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s(%s)",
				a.currentTable, constraintName, a.columnName.Text, parts[0], parts[1])
		case "UNIQUE":
			query = fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s UNIQUE (%s)",
				a.currentTable, constraintName, a.columnName.Text)
		case "CHECK":
			if a.constraintValue.Text == "" {
				return "", fmt.Errorf("укажите условие CHECK")
			}
			query = fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s CHECK (%s)",
				a.currentTable, constraintName, a.constraintValue.Text)
		case "NOT NULL":
			query = fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET NOT NULL",
				a.currentTable, a.columnName.Text)
		}

	case "Удалить ограничение":
		if a.constraintValue.Text == "" {
			return "", fmt.Errorf("укажите имя ограничения")
		}
		query = fmt.Sprintf("ALTER TABLE %s DROP CONSTRAINT %s",
			a.currentTable, a.constraintValue.Text)

	case "Переименовать таблицу":
		if a.newColumnName.Text == "" {
			return "", fmt.Errorf("укажите новое имя таблицы")
		}
		query = fmt.Sprintf("ALTER TABLE %s RENAME TO %s",
			a.currentTable, a.newColumnName.Text)
	}

	return query, nil
}

func (a *AlterTableWindow) applyChanges() {
	query, err := a.buildAlterQuery()
	if err != nil {
		a.showError(err)
		return
	}

	err = a.repository.ExecuteAlter(context.Background(), query)
	if err != nil {
		a.showError(err)
		return
	}

	a.resultLabel.SetText("Изменения успешно применены!\nSQL: " + query)
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
}

func (a *AlterTableWindow) Show() {
	a.window.Show()
}
