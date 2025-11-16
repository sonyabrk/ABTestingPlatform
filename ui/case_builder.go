package ui

import (
	"context"
	"fmt"
	"strings"
	"testing-platform/db"
	"testing-platform/db/models"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

type CaseBuilderWindow struct {
	window fyne.Window
	rep    *db.Repository
	parent fyne.Window

	// Элементы интерфейса для CASE
	tableSelect   *widget.Select
	columnSelect  *widget.Select
	aliasEntry    *widget.Entry
	caseContainer *fyne.Container
	elseEntry     *widget.Entry

	// Элементы для COALESCE
	coalesceTableSelect     *widget.Select
	coalesceColumnSelect    *widget.Select
	coalesceValuesContainer *fyne.Container
	coalesceAliasEntry      *widget.Entry

	// Элементы для NULLIF
	nullifTableSelect  *widget.Select
	nullifColumnSelect *widget.Select
	nullifValueEntry   *widget.Entry
	nullifAliasEntry   *widget.Entry

	// Общие элементы
	resultLabel *widget.Label
	sqlPreview  *widget.Entry
}

func NewCaseBuilderWindow(rep *db.Repository, parent fyne.Window) *CaseBuilderWindow {
	window := fyne.CurrentApp().NewWindow("Конструктор CASE и NULL-функции")
	window.Resize(fyne.NewSize(1000, 700))
	window.SetFixedSize(false)

	cb := &CaseBuilderWindow{
		window:      window,
		rep:         rep,
		parent:      parent,
		resultLabel: widget.NewLabel(""),
		sqlPreview:  widget.NewMultiLineEntry(),
	}

	cb.sqlPreview.Wrapping = fyne.TextWrapWord
	cb.createUI()

	return cb
}

func (cb *CaseBuilderWindow) createUI() {
	// Создаем вкладки
	tabs := container.NewAppTabs(
		container.NewTabItem("Конструктор CASE", cb.createCaseTab()),
		container.NewTabItem("Функция COALESCE", cb.createCoalesceTab()),
		container.NewTabItem("Функция NULLIF", cb.createNullifTab()),
	)

	cb.window.SetContent(tabs)
}

func (cb *CaseBuilderWindow) createCaseTab() fyne.CanvasObject {
	// Выбор таблицы
	cb.tableSelect = widget.NewSelect([]string{}, func(value string) {
		cb.loadColumns(value)
	})
	cb.tableSelect.PlaceHolder = "Выберите таблицу"

	// Выбор столбца
	cb.columnSelect = widget.NewSelect([]string{}, func(value string) {})
	cb.columnSelect.PlaceHolder = "Выберите столбец для условий"

	// Алиас для результата
	cb.aliasEntry = widget.NewEntry()
	cb.aliasEntry.SetPlaceHolder("Название нового столбца")

	// Контейнер для условий WHEN...THEN
	cb.caseContainer = container.NewVBox()

	// ELSE значение
	cb.elseEntry = widget.NewEntry()
	cb.elseEntry.SetPlaceHolder("Значение по умолчанию (ELSE)")

	// Кнопки управления
	addConditionBtn := widget.NewButton("Добавить условие WHEN", cb.addCaseCondition)
	clearConditionsBtn := widget.NewButton("Очистить условия", cb.clearCaseConditions)
	executeBtn := widget.NewButton("Выполнить запрос", cb.executeCaseQuery)
	showSqlBtn := widget.NewButton("Показать SQL", cb.showCaseSQL)

	// Загружаем список таблиц
	cb.loadTables()

	// Собираем интерфейс
	form := container.NewVBox(
		widget.NewLabel("Конструктор выражений CASE"),
		widget.NewSeparator(),

		widget.NewLabel("Таблица:"),
		cb.tableSelect,

		widget.NewLabel("Базовый столбец (опционально):"),
		cb.columnSelect,

		widget.NewLabel("Название результирующего столбца:"),
		cb.aliasEntry,

		widget.NewSeparator(),
		widget.NewLabel("Условия WHEN...THEN:"),
		cb.caseContainer,

		container.NewHBox(addConditionBtn, clearConditionsBtn),

		widget.NewLabel("Значение ELSE:"),
		cb.elseEntry,

		widget.NewSeparator(),
		container.NewHBox(executeBtn, showSqlBtn),

		cb.resultLabel,
	)

	return container.NewScroll(form)
}

func (cb *CaseBuilderWindow) createCoalesceTab() fyne.CanvasObject {
	// Выбор таблицы
	cb.coalesceTableSelect = widget.NewSelect([]string{}, func(value string) {
		cb.loadCoalesceColumns(value)
	})
	cb.coalesceTableSelect.PlaceHolder = "Выберите таблицу"

	// Выбор столбца
	cb.coalesceColumnSelect = widget.NewSelect([]string{}, func(value string) {})
	cb.coalesceColumnSelect.PlaceHolder = "Выберите столбец для проверки NULL"

	// Контейнер для значений
	cb.coalesceValuesContainer = container.NewVBox()

	// Алиас
	cb.coalesceAliasEntry = widget.NewEntry()
	cb.coalesceAliasEntry.SetPlaceHolder("Название нового столбца")

	// Кнопки
	addValueBtn := widget.NewButton("Добавить значение", cb.addCoalesceValue)
	clearValuesBtn := widget.NewButton("Очистить значения", cb.clearCoalesceValues)
	executeBtn := widget.NewButton("Выполнить запрос", cb.executeCoalesceQuery)
	showSqlBtn := widget.NewButton("Показать SQL", cb.showCoalesceSQL)

	// Загружаем таблицы
	cb.loadCoalesceTables()

	form := container.NewVBox(
		widget.NewLabel("Функция COALESCE - подстановка значений вместо NULL"),
		widget.NewSeparator(),

		widget.NewLabel("Таблица:"),
		cb.coalesceTableSelect,

		widget.NewLabel("Проверяемый столбец:"),
		cb.coalesceColumnSelect,

		widget.NewLabel("Значения для подстановки (в порядке приоритета):"),
		cb.coalesceValuesContainer,

		container.NewHBox(addValueBtn, clearValuesBtn),

		widget.NewLabel("Название результирующего столбца:"),
		cb.coalesceAliasEntry,

		widget.NewSeparator(),
		container.NewHBox(executeBtn, showSqlBtn),

		cb.resultLabel,
	)

	return container.NewScroll(form)
}

func (cb *CaseBuilderWindow) createNullifTab() fyne.CanvasObject {
	// Выбор таблицы
	cb.nullifTableSelect = widget.NewSelect([]string{}, func(value string) {
		cb.loadNullifColumns(value)
	})
	cb.nullifTableSelect.PlaceHolder = "Выберите таблицу"

	// Выбор столбца
	cb.nullifColumnSelect = widget.NewSelect([]string{}, func(value string) {})
	cb.nullifColumnSelect.PlaceHolder = "Выберите столбец"

	// Значение для сравнения
	cb.nullifValueEntry = widget.NewEntry()
	cb.nullifValueEntry.SetPlaceHolder("Значение для сравнения")

	// Алиас
	cb.nullifAliasEntry = widget.NewEntry()
	cb.nullifAliasEntry.SetPlaceHolder("Название нового столбца")

	// Кнопки
	executeBtn := widget.NewButton("Выполнить запрос", cb.executeNullifQuery)
	showSqlBtn := widget.NewButton("Показать SQL", cb.showNullifSQL)

	// Загружаем таблицы
	cb.loadNullifTables()

	form := container.NewVBox(
		widget.NewLabel("Функция NULLIF - замена совпадающих значений на NULL"),
		widget.NewSeparator(),

		widget.NewLabel("Таблица:"),
		cb.nullifTableSelect,

		widget.NewLabel("Столбец:"),
		cb.nullifColumnSelect,

		widget.NewLabel("Значение для сравнения:"),
		cb.nullifValueEntry,

		widget.NewLabel("Название результирующего столбца:"),
		cb.nullifAliasEntry,

		widget.NewSeparator(),
		container.NewHBox(executeBtn, showSqlBtn),

		cb.resultLabel,
	)

	return container.NewScroll(form)
}

func (cb *CaseBuilderWindow) addCaseCondition() {
	whenEntry := widget.NewEntry()
	whenEntry.SetPlaceHolder("Значение условия")
	thenEntry := widget.NewEntry()
	thenEntry.SetPlaceHolder("Результат")

	// Создаем строку условия
	conditionRow := container.NewHBox(
		widget.NewLabel("WHEN"),
		whenEntry,
		widget.NewLabel("THEN"),
		thenEntry,
	)

	// Создаем кнопку удаления отдельно
	deleteBtn := widget.NewButton("✕", nil)

	// Добавляем кнопку в строку
	conditionRow.Add(deleteBtn)

	// Устанавливаем обработчик для кнопки удаления
	deleteBtn.OnTapped = func() {
		cb.caseContainer.Remove(conditionRow)
		cb.caseContainer.Refresh()
	}

	cb.caseContainer.Add(conditionRow)
	cb.caseContainer.Refresh()
}

func (cb *CaseBuilderWindow) clearCaseConditions() {
	cb.caseContainer.Objects = nil
	cb.caseContainer.Refresh()
}

func (cb *CaseBuilderWindow) addCoalesceValue() {
	valueEntry := widget.NewEntry()
	valueEntry.SetPlaceHolder("Значение")

	// Создаем строку значения
	valueRow := container.NewHBox(
		valueEntry,
	)

	// Создаем кнопку удаления отдельно
	deleteBtn := widget.NewButton("✕", nil)

	// Добавляем кнопку в строку
	valueRow.Add(deleteBtn)

	// Устанавливаем обработчик для кнопки удаления
	deleteBtn.OnTapped = func() {
		cb.coalesceValuesContainer.Remove(valueRow)
		cb.coalesceValuesContainer.Refresh()
	}

	cb.coalesceValuesContainer.Add(valueRow)
	cb.coalesceValuesContainer.Refresh()
}

func (cb *CaseBuilderWindow) clearCoalesceValues() {
	cb.coalesceValuesContainer.Objects = nil
	cb.coalesceValuesContainer.Refresh()
}

func (cb *CaseBuilderWindow) loadTables() {
	ctx := context.Background()
	tables, err := cb.rep.GetTableNames(ctx)
	if err != nil {
		cb.resultLabel.SetText("Ошибка загрузки таблиц: " + err.Error())
		return
	}

	cb.tableSelect.Options = tables
	cb.tableSelect.Refresh()
}

func (cb *CaseBuilderWindow) loadCoalesceTables() {
	ctx := context.Background()
	tables, err := cb.rep.GetTableNames(ctx)
	if err != nil {
		cb.resultLabel.SetText("Ошибка загрузки таблиц: " + err.Error())
		return
	}

	cb.coalesceTableSelect.Options = tables
	cb.coalesceTableSelect.Refresh()
}

func (cb *CaseBuilderWindow) loadNullifTables() {
	ctx := context.Background()
	tables, err := cb.rep.GetTableNames(ctx)
	if err != nil {
		cb.resultLabel.SetText("Ошибка загрузки таблиц: " + err.Error())
		return
	}

	cb.nullifTableSelect.Options = tables
	cb.nullifTableSelect.Refresh()
}

func (cb *CaseBuilderWindow) loadColumns(tableName string) {
	if tableName == "" {
		return
	}

	ctx := context.Background()
	schema, err := cb.rep.GetTableSchema(ctx, tableName)
	if err != nil {
		cb.resultLabel.SetText("Ошибка загрузки столбцов: " + err.Error())
		return
	}

	columns := make([]string, 0, len(schema))
	for _, colInfo := range schema {
		columns = append(columns, colInfo.Name)
	}

	cb.columnSelect.Options = columns
	cb.columnSelect.Refresh()
}

func (cb *CaseBuilderWindow) loadCoalesceColumns(tableName string) {
	if tableName == "" {
		return
	}

	ctx := context.Background()
	schema, err := cb.rep.GetTableSchema(ctx, tableName)
	if err != nil {
		cb.resultLabel.SetText("Ошибка загрузки столбцов: " + err.Error())
		return
	}

	columns := make([]string, 0, len(schema))
	for _, colInfo := range schema {
		columns = append(columns, colInfo.Name)
	}

	cb.coalesceColumnSelect.Options = columns
	cb.coalesceColumnSelect.Refresh()
}

func (cb *CaseBuilderWindow) loadNullifColumns(tableName string) {
	if tableName == "" {
		return
	}

	ctx := context.Background()
	schema, err := cb.rep.GetTableSchema(ctx, tableName)
	if err != nil {
		cb.resultLabel.SetText("Ошибка загрузки столбцов: " + err.Error())
		return
	}

	columns := make([]string, 0, len(schema))
	for _, colInfo := range schema {
		columns = append(columns, colInfo.Name)
	}

	cb.nullifColumnSelect.Options = columns
	cb.nullifColumnSelect.Refresh()
}

func (cb *CaseBuilderWindow) executeCaseQuery() {
	sql := cb.generateCaseSQL()
	if sql == "" {
		cb.resultLabel.SetText("Ошибка: не заполнены обязательные поля")
		return
	}

	cb.executeQuery(sql)
}

func (cb *CaseBuilderWindow) executeCoalesceQuery() {
	sql := cb.generateCoalesceSQL()
	if sql == "" {
		cb.resultLabel.SetText("Ошибка: не заполнены обязательные поля")
		return
	}

	cb.executeQuery(sql)
}

func (cb *CaseBuilderWindow) executeNullifQuery() {
	sql := cb.generateNullifSQL()
	if sql == "" {
		cb.resultLabel.SetText("Ошибка: не заполнены обязательные поля")
		return
	}

	cb.executeQuery(sql)
}

func (cb *CaseBuilderWindow) showCaseSQL() {
	sql := cb.generateCaseSQL()
	if sql == "" {
		cb.resultLabel.SetText("Ошибка: не заполнены обязательные поля")
		return
	}

	cb.showSQLDialog(sql)
}

func (cb *CaseBuilderWindow) showCoalesceSQL() {
	sql := cb.generateCoalesceSQL()
	if sql == "" {
		cb.resultLabel.SetText("Ошибка: не заполнены обязательные поля")
		return
	}

	cb.showSQLDialog(sql)
}

func (cb *CaseBuilderWindow) showNullifSQL() {
	sql := cb.generateNullifSQL()
	if sql == "" {
		cb.resultLabel.SetText("Ошибка: не заполнены обязательные поля")
		return
	}

	cb.showSQLDialog(sql)
}

func (cb *CaseBuilderWindow) generateCaseSQL() string {
	if cb.tableSelect.Selected == "" || cb.aliasEntry.Text == "" {
		return ""
	}

	var caseBuilder strings.Builder
	caseBuilder.WriteString("CASE\n")

	// Добавляем условия WHEN...THEN
	for _, obj := range cb.caseContainer.Objects {
		if row, ok := obj.(*fyne.Container); ok && len(row.Objects) >= 5 {
			whenEntry := row.Objects[1].(*widget.Entry)
			thenEntry := row.Objects[3].(*widget.Entry)
			whenValue := whenEntry.Text
			thenValue := thenEntry.Text
			if whenValue != "" && thenValue != "" {
				if cb.columnSelect.Selected != "" {
					caseBuilder.WriteString(fmt.Sprintf("    WHEN %s = '%s' THEN '%s'\n",
						cb.columnSelect.Selected, whenValue, thenValue))
				} else {
					caseBuilder.WriteString(fmt.Sprintf("    WHEN '%s' THEN '%s'\n", whenValue, thenValue))
				}
			}
		}
	}

	// Добавляем ELSE
	if cb.elseEntry.Text != "" {
		caseBuilder.WriteString(fmt.Sprintf("    ELSE '%s'\n", cb.elseEntry.Text))
	}

	caseBuilder.WriteString("END")

	return fmt.Sprintf("SELECT *, %s AS %s FROM %s",
		caseBuilder.String(), cb.aliasEntry.Text, cb.tableSelect.Selected)
}

func (cb *CaseBuilderWindow) generateCoalesceSQL() string {
	if cb.coalesceTableSelect.Selected == "" || cb.coalesceColumnSelect.Selected == "" ||
		cb.coalesceAliasEntry.Text == "" {
		return ""
	}

	var values []string
	values = append(values, cb.coalesceColumnSelect.Selected)

	// Собираем значения из контейнера
	for _, obj := range cb.coalesceValuesContainer.Objects {
		if row, ok := obj.(*fyne.Container); ok && len(row.Objects) > 0 {
			if entry, ok := row.Objects[0].(*widget.Entry); ok && entry.Text != "" {
				// Проверяем, число ли это
				if _, isNum := tryParseNumber(entry.Text); isNum {
					values = append(values, entry.Text)
				} else {
					values = append(values, "'"+entry.Text+"'")
				}
			}
		}
	}

	if len(values) == 1 {
		values = append(values, "'N/A'") // значение по умолчанию
	}

	coalesceExpr := "COALESCE(" + strings.Join(values, ", ") + ")"

	return fmt.Sprintf("SELECT *, %s AS %s FROM %s",
		coalesceExpr, cb.coalesceAliasEntry.Text, cb.coalesceTableSelect.Selected)
}

func (cb *CaseBuilderWindow) generateNullifSQL() string {
	if cb.nullifTableSelect.Selected == "" || cb.nullifColumnSelect.Selected == "" ||
		cb.nullifValueEntry.Text == "" || cb.nullifAliasEntry.Text == "" {
		return ""
	}

	value := cb.nullifValueEntry.Text
	if _, isNum := tryParseNumber(value); !isNum {
		value = "'" + value + "'"
	}

	nullifExpr := fmt.Sprintf("NULLIF(%s, %s)", cb.nullifColumnSelect.Selected, value)

	return fmt.Sprintf("SELECT *, %s AS %s FROM %s",
		nullifExpr, cb.nullifAliasEntry.Text, cb.nullifTableSelect.Selected)
}

func (cb *CaseBuilderWindow) executeQuery(sql string) {
	ctx := context.Background()
	result, err := cb.rep.ExecuteQuery(ctx, sql)
	if err != nil {
		cb.resultLabel.SetText("Ошибка выполнения запроса: " + err.Error())
		return
	}

	if result.Error != "" {
		cb.resultLabel.SetText("Ошибка БД: " + result.Error)
		return
	}

	// Показываем результат в новом окне
	cb.showResultWindow(result, sql)

	cb.resultLabel.SetText(fmt.Sprintf("Запрос выполнен успешно. Найдено %d строк", len(result.Rows)))
}

func (cb *CaseBuilderWindow) showResultWindow(result *models.QueryResult, sql string) {
	window := fyne.CurrentApp().NewWindow("Результаты запроса")
	window.Resize(fyne.NewSize(800, 600))

	// Создаем таблицу для отображения результатов
	table := widget.NewTable(
		func() (int, int) {
			return len(result.Rows) + 1, len(result.Columns) // +1 для заголовков
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("")
		},
		func(i widget.TableCellID, o fyne.CanvasObject) {
			label := o.(*widget.Label)
			if i.Row == 0 {
				// Заголовки
				if i.Col < len(result.Columns) {
					label.SetText(result.Columns[i.Col])
				}
			} else {
				// Данные
				rowIndex := i.Row - 1
				if rowIndex < len(result.Rows) && i.Col < len(result.Columns) {
					value := result.Rows[rowIndex][result.Columns[i.Col]]
					if value == nil {
						label.SetText("NULL")
					} else {
						label.SetText(fmt.Sprintf("%v", value))
					}
				}
			}
		})

	// Настройка ширины столбцов
	for col := 0; col < len(result.Columns); col++ {
		table.SetColumnWidth(col, 150)
	}

	// Показываем SQL запрос
	sqlLabel := widget.NewLabel("SQL: " + sql)
	sqlLabel.Wrapping = fyne.TextWrapWord

	content := container.NewBorder(
		sqlLabel,
		nil, nil, nil,
		container.NewScroll(table),
	)

	window.SetContent(content)
	window.Show()
}

func (cb *CaseBuilderWindow) showSQLDialog(sql string) {
	preview := widget.NewMultiLineEntry()
	preview.SetText(sql)
	preview.Wrapping = fyne.TextWrapWord

	scroll := container.NewScroll(preview)
	scroll.SetMinSize(fyne.NewSize(600, 400))

	dialog.ShowCustom("Сгенерированный SQL", "Закрыть", scroll, cb.window)
}

func (cb *CaseBuilderWindow) Show() {
	cb.window.Show()
}

// Вспомогательная функция для проверки, является ли строка числом
func tryParseNumber(s string) (float64, bool) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err == nil
}
