// custom_types.go
package ui

import (
	"context"
	"fmt"
	"strings"
	"testing-platform/db"
	"testing-platform/db/models"
	"testing-platform/pkg/logger"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

type CustomTypesWindow struct {
	window     fyne.Window
	repository *db.Repository
	app        fyne.App

	// Элементы управления
	typeSelect      *widget.Select
	typeName        *widget.Entry
	enumValues      *widget.Entry
	compositeFields *widget.Entry
	resultLabel     *widget.Label
	typeList        *widget.List

	// Данные
	customTypes []models.CustomType
	selectedID  int // Для отслеживания выбранного элемента
}

func NewCustomTypesWindow(repo *db.Repository, app fyne.App) *CustomTypesWindow {
	c := &CustomTypesWindow{
		repository: repo,
		app:        app,
		window:     app.NewWindow("Пользовательские типы данных"),
		selectedID: -1,
	}

	c.buildUI()
	c.loadCustomTypes()
	return c
}

func (c *CustomTypesWindow) buildUI() {
	// Сначала создаем все элементы управления
	c.typeName = widget.NewEntry()
	c.typeName.SetPlaceHolder("Имя типа (например: status_type)")

	c.enumValues = widget.NewEntry()
	c.enumValues.SetPlaceHolder("Значения ENUM через запятую (например: active,inactive,pending)")

	c.compositeFields = widget.NewEntry()
	c.compositeFields.SetPlaceHolder("Поля через запятую в формате имя:тип (например: name:text,age:integer)")

	c.resultLabel = widget.NewLabel("")
	c.resultLabel.Wrapping = fyne.TextWrapWord

	// Создаем typeSelect после того как все поля инициализированы
	c.typeSelect = widget.NewSelect([]string{
		string(models.EnumType),
		string(models.CompositeType),
	}, c.onTypeSelected)
	c.typeSelect.PlaceHolder = "Тип данных"

	// Список существующих типов
	c.typeList = widget.NewList(
		func() int {
			return len(c.customTypes)
		},
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewLabel("Тип:"),
				widget.NewLabel("Имя:"),
				widget.NewLabel("Значения:"),
			)
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			cont := o.(*fyne.Container)
			labels := cont.Objects
			t := c.customTypes[i]
			labels[0].(*widget.Label).SetText(string(t.Type))
			labels[1].(*widget.Label).SetText(t.Name)

			if t.Type == models.EnumType {
				labels[2].(*widget.Label).SetText(strings.Join(t.EnumValues, ", "))
			} else {
				var fields []string
				for _, f := range t.CompositeFields {
					fields = append(fields, fmt.Sprintf("%s:%s", f.Name, f.DataType))
				}
				labels[2].(*widget.Label).SetText(strings.Join(fields, ", "))
			}
		},
	)

	// Обработчик выбора элемента в списке
	c.typeList.OnSelected = func(id widget.ListItemID) {
		c.selectedID = id
	}

	// Кнопки
	createBtn := widget.NewButton("Создать тип", c.createType)
	deleteBtn := widget.NewButton("Удалить выбранный тип", c.deleteType)
	refreshBtn := widget.NewButton("Обновить список", c.loadCustomTypes)
	useInTableBtn := widget.NewButton("Использовать в таблице", c.useInTable)

	// Форма создания
	createForm := container.NewVBox(
		widget.NewLabel("Создать новый тип:"),
		c.typeSelect,
		c.typeName,
		c.enumValues,
		c.compositeFields,
		container.NewHBox(createBtn, useInTableBtn),
	)

	// Основной контент
	content := container.NewBorder(
		createForm,
		container.NewVBox(c.resultLabel, container.NewHBox(deleteBtn, refreshBtn)),
		nil, nil,
		container.NewScroll(c.typeList),
	)

	c.window.SetContent(content)
	c.window.Resize(fyne.NewSize(800, 600))

	// Устанавливаем значение по умолчанию ПОСЛЕ того как все инициализировано
	c.typeSelect.SetSelected(string(models.EnumType))
}

func (c *CustomTypesWindow) onTypeSelected(selected string) {
	// Проверяем, что поля инициализированы перед использованием
	if c.enumValues == nil || c.compositeFields == nil {
		return
	}

	switch models.CustomTypeType(selected) {
	case models.EnumType:
		c.enumValues.Show()
		c.compositeFields.Hide()
	case models.CompositeType:
		c.enumValues.Hide()
		c.compositeFields.Show()
	default:
		c.enumValues.Hide()
		c.compositeFields.Hide()
	}
}

func (c *CustomTypesWindow) createType() {
	typeName := strings.TrimSpace(c.typeName.Text)
	if typeName == "" {
		c.resultLabel.SetText("❌ Ошибка: имя типа не может быть пустым")
		return
	}

	ctx := context.Background()
	var query string

	switch models.CustomTypeType(c.typeSelect.Selected) {
	case models.EnumType:
		values := strings.Split(c.enumValues.Text, ",")
		if len(values) == 0 {
			c.resultLabel.SetText("❌ Ошибка: укажите значения ENUM")
			return
		}

		// Форматируем значения для SQL
		var formattedValues []string
		for _, v := range values {
			trimmed := strings.TrimSpace(v)
			if trimmed != "" {
				formattedValues = append(formattedValues, "'"+trimmed+"'")
			}
		}

		query = fmt.Sprintf("CREATE TYPE %s AS ENUM (%s)", typeName, strings.Join(formattedValues, ", "))

	case models.CompositeType:
		fields := strings.Split(c.compositeFields.Text, ",")
		if len(fields) == 0 {
			c.resultLabel.SetText("❌ Ошибка: укажите поля составного типа")
			return
		}

		var fieldDefinitions []string
		for _, field := range fields {
			parts := strings.Split(strings.TrimSpace(field), ":")
			if len(parts) != 2 {
				c.resultLabel.SetText("❌ Ошибка: некорректный формат поля. Используйте имя:тип")
				return
			}
			fieldName := strings.TrimSpace(parts[0])
			fieldType := strings.TrimSpace(parts[1])
			fieldDefinitions = append(fieldDefinitions, fmt.Sprintf("%s %s", fieldName, fieldType))
		}

		query = fmt.Sprintf("CREATE TYPE %s AS (%s)", typeName, strings.Join(fieldDefinitions, ", "))
	}

	// Выполняем создание типа
	err := c.repository.ExecuteAlter(ctx, query)
	if err != nil {
		c.resultLabel.SetText("❌ Ошибка создания типа: " + err.Error())
		return
	}

	c.resultLabel.SetText("✅ Тип " + typeName + " успешно создан")
	c.clearForm()
	c.loadCustomTypes()
}

func (c *CustomTypesWindow) deleteType() {
	if c.selectedID == -1 {
		c.resultLabel.SetText("❌ Ошибка: выберите тип для удаления")
		return
	}

	typeName := c.customTypes[c.selectedID].Name

	dialog.ShowConfirm("Удаление типа",
		fmt.Sprintf("Вы уверены, что хотите удалить тип '%s'?", typeName),
		func(confirm bool) {
			if confirm {
				ctx := context.Background()
				query := fmt.Sprintf("DROP TYPE %s", typeName)

				err := c.repository.ExecuteAlter(ctx, query)
				if err != nil {
					c.resultLabel.SetText("❌ Ошибка удаления типа: " + err.Error())
					return
				}

				c.resultLabel.SetText("✅ Тип " + typeName + " успешно удален")
				c.selectedID = -1
				c.loadCustomTypes()
			}
		}, c.window)
}

func (c *CustomTypesWindow) useInTable() {
	if c.selectedID == -1 {
		c.resultLabel.SetText("❌ Ошибка: выберите тип для использования")
		return
	}

	// Открываем окно ALTER TABLE
	alterWin := NewAlterTableWindow(c.repository, c.window, func() {})
	alterWin.Show()

	c.window.Close()
}

func (c *CustomTypesWindow) loadCustomTypes() {
	ctx := context.Background()

	// Загружаем ENUM типы
	enumQuery := `
		SELECT t.typname as type_name, 
		       array_agg(e.enumlabel ORDER BY e.enumsortorder) as enum_values
		FROM pg_type t 
		JOIN pg_enum e ON t.oid = e.enumtypid  
		JOIN pg_catalog.pg_namespace n ON n.oid = t.typnamespace
		WHERE n.nspname = 'public'
		GROUP BY t.typname
	`

	enumResult, err := c.repository.ExecuteQuery(ctx, enumQuery)
	if err != nil {
		logger.Error("Ошибка загрузки ENUM типов: %v", err)
		return
	}

	c.customTypes = []models.CustomType{}

	for _, row := range enumResult.Rows {
		if typeName, ok := row["type_name"].(string); ok {
			enumType := models.CustomType{
				Name: typeName,
				Type: models.EnumType,
			}

			// PostgreSQL возвращает массив как строку в формате {value1,value2}
			if valuesStr, ok := row["enum_values"].(string); ok {
				// Убираем фигурные скобки и разбиваем по запятым
				valuesStr = strings.Trim(valuesStr, "{}")
				enumType.EnumValues = strings.Split(valuesStr, ",")
			}

			c.customTypes = append(c.customTypes, enumType)
		}
	}

	// Загружаем составные типы (можно расширить при необходимости)
	c.typeList.Refresh()
	c.resultLabel.SetText(fmt.Sprintf("Загружено типов: %d", len(c.customTypes)))
}

func (c *CustomTypesWindow) clearForm() {
	c.typeName.SetText("")
	c.enumValues.SetText("")
	c.compositeFields.SetText("")
}

func (c *CustomTypesWindow) Show() {
	c.window.Show()
}
