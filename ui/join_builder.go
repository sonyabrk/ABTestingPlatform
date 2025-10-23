package ui

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"testing-platform/db"
	"testing-platform/db/models"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

type JoinBuilderWindow struct {
	window     fyne.Window
	repository *db.Repository
	mainWindow fyne.Window

	mainTableSelect  *widget.Select
	joinTypeSelect   *widget.Select
	joinTableSelect  *widget.Select
	mainColumnSelect *widget.Select
	joinColumnSelect *widget.Select

	additionalJoins *widget.Accordion
	resultTable     *widget.Table
	resultLabel     *widget.Label
	sqlPreview      *widget.Entry

	// Новые поля для сортировки и фильтрации
	sortColumnSelect    *widget.Select
	sortDirectionSelect *widget.Select
	filterInput         *widget.Entry
	filterColumnSelect  *widget.Select

	tables       []string
	tableColumns map[string][]models.ColumnInfo

	// Данные для сортировки и фильтрации
	currentResult *models.QueryResult
	filteredRows  []map[string]interface{}
	sortColumn    string
	sortAscending bool
	filterText    string
	filterColumn  string
}

func NewJoinBuilderWindow(repo *db.Repository, mainWindow fyne.Window) *JoinBuilderWindow {
	j := &JoinBuilderWindow{
		repository:    repo,
		mainWindow:    mainWindow,
		window:        fyne.CurrentApp().NewWindow("Мастер JOIN"),
		tableColumns:  make(map[string][]models.ColumnInfo),
		sortAscending: true,
		filterColumn:  "Все столбцы",
	}

	j.buildUI()
	j.loadTables()
	return j
}

func (j *JoinBuilderWindow) buildUI() {
	j.mainTableSelect = widget.NewSelect([]string{}, j.onMainTableSelected)
	j.mainTableSelect.PlaceHolder = "Основная таблица"

	j.joinTypeSelect = widget.NewSelect([]string{
		"INNER JOIN", "LEFT JOIN", "RIGHT JOIN", "FULL JOIN",
	}, nil)
	j.joinTypeSelect.SetSelected("INNER JOIN")
	j.joinTypeSelect.PlaceHolder = "Тип JOIN"

	j.joinTableSelect = widget.NewSelect([]string{}, j.onJoinTableSelected)
	j.joinTableSelect.PlaceHolder = "Таблица для JOIN"

	j.mainColumnSelect = widget.NewSelect([]string{}, nil)
	j.mainColumnSelect.PlaceHolder = "Столбец основной таблицы"

	j.joinColumnSelect = widget.NewSelect([]string{}, nil)
	j.joinColumnSelect.PlaceHolder = "Столбец присоединяемой таблицы"

	j.sqlPreview = widget.NewMultiLineEntry()
	j.sqlPreview.Wrapping = fyne.TextWrapOff
	j.sqlPreview.SetPlaceHolder("Здесь будет показан SQL-запрос...")

	j.resultLabel = widget.NewLabel("Постройте JOIN для просмотра результатов")
	j.resultLabel.Wrapping = fyne.TextWrapWord

	// НОВЫЕ ЭЛЕМЕНТЫ ДЛЯ СОРТИРОВКИ И ФИЛЬТРАЦИИ
	j.sortColumnSelect = widget.NewSelect([]string{}, nil)
	j.sortColumnSelect.PlaceHolder = "Столбец для сортировки"
	j.sortColumnSelect.OnChanged = j.onSortColumnChanged

	j.sortDirectionSelect = widget.NewSelect([]string{"По возрастанию ↑", "По убыванию ↓"}, nil)
	j.sortDirectionSelect.SetSelected("По возрастанию ↑")
	j.sortDirectionSelect.OnChanged = j.onSortDirectionChanged

	j.filterInput = widget.NewEntry()
	j.filterInput.SetPlaceHolder("Текст для фильтрации...")
	j.filterInput.OnChanged = j.onFilterTextChanged

	j.filterColumnSelect = widget.NewSelect([]string{"Все столбцы"}, nil)
	j.filterColumnSelect.SetSelected("Все столбцы")
	j.filterColumnSelect.OnChanged = j.onFilterColumnChanged

	mainTableLabel := widget.NewLabel("Основная таблица:")
	mainTableLabel.TextStyle = fyne.TextStyle{Bold: true}

	joinTypeLabel := widget.NewLabel("Тип JOIN:")
	joinTypeLabel.TextStyle = fyne.TextStyle{Bold: true}

	joinTableLabel := widget.NewLabel("Присоединяемая таблица:")
	joinTableLabel.TextStyle = fyne.TextStyle{Bold: true}

	mainColumnLabel := widget.NewLabel("Столбец основной таблицы:")
	mainColumnLabel.TextStyle = fyne.TextStyle{Bold: true}

	joinColumnLabel := widget.NewLabel("Столбец присоединяемой таблицы:")
	joinColumnLabel.TextStyle = fyne.TextStyle{Bold: true}

	sqlLabel := widget.NewLabel("SQL запрос:")
	sqlLabel.TextStyle = fyne.TextStyle{Bold: true}

	additionalJoinsLabel := widget.NewLabel("Дополнительные JOIN:")
	additionalJoinsLabel.TextStyle = fyne.TextStyle{Bold: true}

	// Таблица с переносом текста
	j.resultTable = widget.NewTable(
		func() (int, int) { return 0, 0 },
		func() fyne.CanvasObject {
			label := widget.NewLabel("")
			label.Wrapping = fyne.TextWrapWord
			return label
		},
		func(i widget.TableCellID, o fyne.CanvasObject) {},
	)

	// Для дополнительных JOIN
	j.additionalJoins = widget.NewAccordion()

	addJoinBtn := widget.NewButton("Добавить еще JOIN", j.addAdditionalJoin)
	executeBtn := widget.NewButton("Выполнить JOIN", j.executeJoin)
	clearBtn := widget.NewButton("Очистить", j.clearForm)

	// Кнопки для управления сортировкой и фильтрацией
	sortFilterLabel := widget.NewLabel("Сортировка и фильтрация результатов:")
	sortFilterLabel.TextStyle = fyne.TextStyle{Bold: true}

	resetSortFilterBtn := widget.NewButton("Сбросить сортировку/фильтр", j.resetSortFilter)

	// Контейнер для сортировки
	sortContainer := container.NewHBox(
		widget.NewLabel("Сортировка:"),
		j.sortColumnSelect,
		j.sortDirectionSelect,
	)

	// Контейнер для фильтрации
	filterContainer := container.NewHBox(
		widget.NewLabel("Фильтр:"),
		j.filterColumnSelect,
		j.filterInput,
	)

	// Добавляем подсказки
	hintLabel := widget.NewLabel("💡 Подсказки:\n• Сначала выберите основную таблицу\n• Затем выберите таблицу для JOIN и столбцы для связи\n• INNER JOIN - только совпадающие строки\n• LEFT JOIN - все строки из левой таблицы\n• RIGHT JOIN - все строки из правой таблицы\n• FULL JOIN - все строки из обеих таблиц")
	hintLabel.Wrapping = fyne.TextWrapWord

	// Компоновка
	joinForm := container.NewVBox(
		mainTableLabel,
		j.mainTableSelect,
		joinTypeLabel,
		j.joinTypeSelect,
		joinTableLabel,
		j.joinTableSelect,
		mainColumnLabel,
		j.mainColumnSelect,
		joinColumnLabel,
		j.joinColumnSelect,
	)

	formContainer := container.NewVScroll(container.NewVBox(
		joinForm,
		hintLabel,
		additionalJoinsLabel,
		addJoinBtn,
		j.additionalJoins,
		container.NewHBox(executeBtn, clearBtn),
		sqlLabel,
		j.sqlPreview,
	))

	// Ограничиваем минимальную высоту формы
	formContainer.SetMinSize(fyne.NewSize(0, 300))

	// Создаем контейнер для результатов с элементами сортировки и фильтрации
	resultContainer := container.NewBorder(
		container.NewVBox(
			j.resultLabel,
			sortFilterLabel,
			sortContainer,
			filterContainer,
			resetSortFilterBtn,
		), nil, nil, nil,
		container.NewScroll(j.resultTable),
	)

	// Разделяем экран по вертикали
	split := container.NewVSplit(
		formContainer,
		resultContainer,
	)
	split.SetOffset(0.4) // 40% для формы, 60% для результатов

	j.window.SetContent(split)
	j.window.Resize(fyne.NewSize(1200, 800))
}

// НОВЫЕ МЕТОДЫ ДЛЯ СОРТИРОВКИ И ФИЛЬТРАЦИИ

// Обновление списка столбцов для сортировки и фильтрации
func (j *JoinBuilderWindow) updateSortFilterColumns(columns []string) {
	j.sortColumnSelect.Options = columns
	j.sortColumnSelect.Refresh()

	// Для фильтрации добавляем опцию "Все столбцы"
	filterOptions := append([]string{"Все столбцы"}, columns...)
	j.filterColumnSelect.Options = filterOptions
	j.filterColumnSelect.Refresh()
}

// Обработчик изменения столбца сортировки
func (j *JoinBuilderWindow) onSortColumnChanged(column string) {
	j.sortColumn = column
	j.applySortAndFilter()
}

// Обработчик изменения направления сортировки
func (j *JoinBuilderWindow) onSortDirectionChanged(direction string) {
	j.sortAscending = direction == "По возрастанию ↑"
	j.applySortAndFilter()
}

// Обработчик изменения текста фильтра
func (j *JoinBuilderWindow) onFilterTextChanged(filterText string) {
	j.filterText = strings.ToLower(filterText)
	j.applySortAndFilter()
}

// Обработчик изменения столбца фильтрации
func (j *JoinBuilderWindow) onFilterColumnChanged(column string) {
	j.filterColumn = column
	j.applySortAndFilter()
}

// Сброс сортировки и фильтрации
func (j *JoinBuilderWindow) resetSortFilter() {
	j.sortColumnSelect.SetSelected("")
	j.sortDirectionSelect.SetSelected("По возрастанию ↑")
	j.filterInput.SetText("")
	j.filterColumnSelect.SetSelected("Все столбцы")

	j.sortColumn = ""
	j.sortAscending = true
	j.filterText = ""
	j.filterColumn = "Все столбцы"

	j.applySortAndFilter()
}

// Применение сортировки и фильтрации
func (j *JoinBuilderWindow) applySortAndFilter() {
	if j.currentResult == nil || len(j.currentResult.Rows) == 0 {
		return
	}

	// Применяем фильтрацию
	j.filteredRows = j.applyFilter(j.currentResult.Rows)

	// Применяем сортировку
	if j.sortColumn != "" {
		j.applySort(j.filteredRows)
	}

	// Обновляем отображение
	j.refreshResultTable()

	// Обновляем label с информацией
	totalRows := len(j.currentResult.Rows)
	filteredRows := len(j.filteredRows)

	if totalRows == filteredRows {
		j.resultLabel.SetText(fmt.Sprintf("Найдено %d строк", totalRows))
	} else {
		j.resultLabel.SetText(fmt.Sprintf("Найдено %d строк (отфильтровано из %d)", filteredRows, totalRows))
	}
}

// Применение фильтрации
func (j *JoinBuilderWindow) applyFilter(rows []map[string]interface{}) []map[string]interface{} {
	if j.filterText == "" {
		return rows
	}

	var filtered []map[string]interface{}
	for _, row := range rows {
		if j.rowMatchesFilter(row) {
			filtered = append(filtered, row)
		}
	}
	return filtered
}

// Проверка соответствия строки фильтру
func (j *JoinBuilderWindow) rowMatchesFilter(row map[string]interface{}) bool {
	if j.filterText == "" {
		return true
	}

	// Если выбран конкретный столбец для фильтрации
	if j.filterColumn != "Все столбцы" {
		value := row[j.filterColumn]
		if value != nil {
			valueStr := strings.ToLower(fmt.Sprintf("%v", value))
			return strings.Contains(valueStr, j.filterText)
		}
		return false
	}

	// Поиск по всем столбцам
	for _, value := range row {
		if value != nil {
			valueStr := strings.ToLower(fmt.Sprintf("%v", value))
			if strings.Contains(valueStr, j.filterText) {
				return true
			}
		}
	}
	return false
}

// Применение сортировки
// Применение сортировки
func (j *JoinBuilderWindow) applySort(rows []map[string]interface{}) {
	if j.sortColumn == "" {
		return
	}

	sort.Slice(rows, func(a, b int) bool {
		val1 := rows[a][j.sortColumn]
		val2 := rows[b][j.sortColumn]

		// Обработка nil значений
		if val1 == nil && val2 == nil {
			return false
		}
		if val1 == nil {
			return !j.sortAscending
		}
		if val2 == nil {
			return j.sortAscending
		}

		// Преобразование в строку для сравнения
		str1 := fmt.Sprintf("%v", val1)
		str2 := fmt.Sprintf("%v", val2)

		// Попытка численного сравнения
		if num1, err1 := strconv.ParseFloat(str1, 64); err1 == nil {
			if num2, err2 := strconv.ParseFloat(str2, 64); err2 == nil {
				if j.sortAscending {
					return num1 < num2
				}
				return num1 > num2
			}
		}

		// Строковое сравнение
		if j.sortAscending {
			return str1 < str2
		}
		return str1 > str2
	})
}

// Обновление таблицы результатов
func (j *JoinBuilderWindow) refreshResultTable() {
	if len(j.filteredRows) == 0 {
		j.resultTable.Length = func() (int, int) { return 1, 1 }
		j.resultTable.UpdateCell = func(id widget.TableCellID, obj fyne.CanvasObject) {
			label := obj.(*widget.Label)
			label.Wrapping = fyne.TextWrapWord
			if id.Row == 0 && id.Col == 0 {
				label.SetText("Нет данных, соответствующих фильтру")
			}
		}
		j.resultTable.Refresh()
		return
	}

	columns := j.currentResult.Columns
	j.resultTable.Length = func() (int, int) {
		return len(j.filteredRows) + 1, len(columns)
	}

	// Автоматическая настройка ширины колонок
	for col := 0; col < len(columns); col++ {
		maxWidth := float32(150)
		headerWidth := float32(len(columns[col])) * 8
		if headerWidth > maxWidth {
			maxWidth = headerWidth
		}
		for row := 0; row < len(j.filteredRows) && row < 10; row++ {
			if value := j.filteredRows[row][columns[col]]; value != nil {
				text := fmt.Sprintf("%v", value)
				textWidth := float32(len(text)) * 7
				if textWidth > maxWidth {
					maxWidth = textWidth
				}
			}
		}
		if maxWidth > 400 {
			maxWidth = 400
		}
		j.resultTable.SetColumnWidth(col, maxWidth)
	}

	j.resultTable.UpdateCell = func(id widget.TableCellID, obj fyne.CanvasObject) {
		label := obj.(*widget.Label)
		label.Wrapping = fyne.TextWrapWord

		if id.Row == 0 {
			if id.Col < len(columns) {
				columnName := columns[id.Col]
				// Подсвечиваем столбец, по которому идет сортировка
				if columnName == j.sortColumn {
					label.SetText(columnName + j.getSortIndicator())
					label.TextStyle = fyne.TextStyle{Bold: true}
				} else {
					label.SetText(columnName)
					label.TextStyle = fyne.TextStyle{Bold: true}
				}
			}
		} else {
			rowIndex := id.Row - 1
			if rowIndex < len(j.filteredRows) && id.Col < len(columns) {
				value := j.filteredRows[rowIndex][columns[id.Col]]
				if value != nil {
					label.SetText(fmt.Sprintf("%v", value))
				} else {
					label.SetText("NULL")
				}
			}
		}
	}

	j.resultTable.Refresh()
}

// Получение индикатора сортировки
func (j *JoinBuilderWindow) getSortIndicator() string {
	if j.sortAscending {
		return " ↑"
	}
	return " ↓"
}

// ОСНОВНЫЕ МЕТОДЫ ОТОБРАЖЕНИЯ РЕЗУЛЬТАТОВ И ОЧИСТКИ

func (j *JoinBuilderWindow) displayResults(result *models.QueryResult) {
	j.currentResult = result
	j.filteredRows = result.Rows

	// Обновляем списки столбцов для сортировки и фильтрации
	j.updateSortFilterColumns(result.Columns)

	// Сбрасываем состояние сортировки и фильтрации
	j.resetSortFilter()

	if len(result.Rows) == 0 {
		j.resultTable.Length = func() (int, int) { return 1, 1 }
		j.resultTable.UpdateCell = func(id widget.TableCellID, obj fyne.CanvasObject) {
			label := obj.(*widget.Label)
			label.Wrapping = fyne.TextWrapWord
			if id.Row == 0 && id.Col == 0 {
				label.SetText("Нет данных")
			}
		}
		j.resultLabel.SetText("JOIN выполнен успешно, но не найдено совпадающих строк")
		return
	}

	j.applySortAndFilter()
}

func (j *JoinBuilderWindow) clearForm() {
	j.mainTableSelect.SetSelected("")
	j.joinTableSelect.SetSelected("")
	j.mainColumnSelect.SetSelected("")
	j.joinColumnSelect.SetSelected("")
	j.additionalJoins.Items = nil
	j.additionalJoins.Refresh()
	j.sqlPreview.SetText("")
	j.resultLabel.SetText("Постройте JOIN для просмотра результатов")
	j.resultTable.Length = func() (int, int) { return 0, 0 }
	j.resultTable.Refresh()

	// Сбрасываем состояние сортировки и фильтрации
	j.currentResult = nil
	j.filteredRows = nil
	j.sortColumnSelect.Options = []string{}
	j.filterColumnSelect.Options = []string{"Все столбцы"}
	j.sortColumnSelect.Refresh()
	j.filterColumnSelect.Refresh()
	j.resetSortFilter()
}

// Валидация формы JOIN
func (j *JoinBuilderWindow) validateJoinForm() error {
	// Проверка основной таблицы
	if j.mainTableSelect.Selected == "" {
		return fmt.Errorf("не выбрана основная таблица")
	}

	// Проверка типа JOIN
	if j.joinTypeSelect.Selected == "" {
		return fmt.Errorf("не выбран тип JOIN")
	}

	// Проверка присоединяемой таблицы
	if j.joinTableSelect.Selected == "" {
		return fmt.Errorf("не выбрана таблица для JOIN")
	}

	// Проверка что таблицы разные
	if j.mainTableSelect.Selected == j.joinTableSelect.Selected {
		return fmt.Errorf("невозможно выполнить JOIN одной таблицы с собой")
	}

	// Проверка столбцов основной таблицы
	if j.mainColumnSelect.Selected == "" {
		return fmt.Errorf("не выбран столбец основной таблицы")
	}

	// Проверка существования выбранного столбца в основной таблице
	if !j.columnExists(j.mainTableSelect.Selected, j.mainColumnSelect.Selected) {
		return fmt.Errorf("выбранный столбец '%s' не существует в таблице '%s'",
			j.mainColumnSelect.Selected, j.mainTableSelect.Selected)
	}

	// Проверка столбцов присоединяемой таблицы
	if j.joinColumnSelect.Selected == "" {
		return fmt.Errorf("не выбран столбец присоединяемой таблицы")
	}

	// Проверка существования выбранного столбца в присоединяемой таблице
	if !j.columnExists(j.joinTableSelect.Selected, j.joinColumnSelect.Selected) {
		return fmt.Errorf("выбранный столбец '%s' не существует в таблице '%s'",
			j.joinColumnSelect.Selected, j.joinTableSelect.Selected)
	}

	// Проверка дополнительных JOIN
	for i, item := range j.additionalJoins.Items {
		if err := j.validateAdditionalJoin(item, i+1); err != nil {
			return err
		}
	}

	return nil
}

// Валидация дополнительного JOIN
func (j *JoinBuilderWindow) validateAdditionalJoin(item *widget.AccordionItem, joinNumber int) error {
	content, ok := item.Detail.(*fyne.Container)
	if !ok || len(content.Objects) < 7 {
		return fmt.Errorf("ошибка в дополнительном JOIN %d: некорректная структура формы", joinNumber)
	}

	joinTypeWidget, ok := content.Objects[0].(*widget.Select)
	if !ok || joinTypeWidget.Selected == "" {
		return fmt.Errorf("в дополнительном JOIN %d не выбран тип JOIN", joinNumber)
	}

	tableWidget, ok := content.Objects[1].(*widget.Select)
	if !ok || tableWidget.Selected == "" {
		return fmt.Errorf("в дополнительном JOIN %d не выбрана таблица", joinNumber)
	}

	mainColWidget, ok := content.Objects[3].(*widget.Select)
	if !ok || mainColWidget.Selected == "" {
		return fmt.Errorf("в дополнительном JOIN %d не выбран столбец из предыдущей таблицы", joinNumber)
	}

	joinColWidget, ok := content.Objects[5].(*widget.Select)
	if !ok || joinColWidget.Selected == "" {
		return fmt.Errorf("в дополнительном JOIN %d не выбран столбец присоединяемой таблицы", joinNumber)
	}

	// Проверка существования столбцов
	if !j.columnExistsInAnyTable(mainColWidget.Selected) {
		return fmt.Errorf("в дополнительном JOIN %d: столбец '%s' не существует в предыдущих таблицах",
			joinNumber, mainColWidget.Selected)
	}

	if !j.columnExists(tableWidget.Selected, joinColWidget.Selected) {
		return fmt.Errorf("в дополнительном JOIN %d: столбец '%s' не существует в таблице '%s'",
			joinNumber, joinColWidget.Selected, tableWidget.Selected)
	}

	return nil
}

// Проверка существования столбца в таблице
func (j *JoinBuilderWindow) columnExists(table, column string) bool {
	columns, exists := j.tableColumns[table]
	if !exists {
		return false
	}

	for _, col := range columns {
		if col.Name == column {
			return true
		}
	}
	return false
}

// Проверка существования столбца в любой из таблиц
func (j *JoinBuilderWindow) columnExistsInAnyTable(column string) bool {
	for _, columns := range j.tableColumns {
		for _, col := range columns {
			if col.Name == column {
				return true
			}
		}
	}
	return false
}

func (j *JoinBuilderWindow) loadTables() {
	tables, err := j.repository.GetTables(context.Background())
	if err != nil {
		j.showError(fmt.Errorf("не удалось загрузить список таблиц: проверьте подключение к базе данных"))
		return
	}

	if len(tables) == 0 {
		j.showError(fmt.Errorf("в базе данных не найдено ни одной таблицы"))
		return
	}

	j.tables = tables
	j.mainTableSelect.Options = tables
	j.joinTableSelect.Options = tables
	j.mainTableSelect.Refresh()
	j.joinTableSelect.Refresh()
}

func (j *JoinBuilderWindow) onMainTableSelected(table string) {
	if table == "" {
		return
	}
	j.loadTableColumns(table)
	j.updateColumnSelectors()

	// Обновляем дополнительные JOIN при изменении основной таблицы
	j.updateAdditionalJoins()
}

func (j *JoinBuilderWindow) onJoinTableSelected(table string) {
	if table == "" {
		return
	}
	j.loadTableColumns(table)
	j.updateColumnSelectors()
}

func (j *JoinBuilderWindow) loadTableColumns(table string) {
	if table == "" {
		return
	}

	// Если уже загружены, используем кэш
	if _, exists := j.tableColumns[table]; exists {
		return
	}

	columns, err := j.repository.GetTableColumns(context.Background(), table)
	if err != nil {
		j.showError(fmt.Errorf("не удалось загрузить столбцы таблицы '%s': проверьте права доступа", table))
		return
	}

	j.tableColumns[table] = columns
}

func (j *JoinBuilderWindow) updateColumnSelectors() {
	mainTable := j.mainTableSelect.Selected
	joinTable := j.joinTableSelect.Selected

	if mainTable != "" {
		var mainColumnNames []string
		if columns, exists := j.tableColumns[mainTable]; exists {
			for _, col := range columns {
				mainColumnNames = append(mainColumnNames, col.Name)
			}
		}
		j.mainColumnSelect.Options = mainColumnNames
		j.mainColumnSelect.Refresh()
	}

	if joinTable != "" {
		var joinColumnNames []string
		if columns, exists := j.tableColumns[joinTable]; exists {
			for _, col := range columns {
				joinColumnNames = append(joinColumnNames, col.Name)
			}
		}
		j.joinColumnSelect.Options = joinColumnNames
		j.joinColumnSelect.Refresh()
	}
}

// Получение всех доступных столбцов для дополнительных JOIN
func (j *JoinBuilderWindow) getAllAvailableColumns() []string {
	var allColumns []string

	// Добавляем столбцы из основной таблицы
	if mainTable := j.mainTableSelect.Selected; mainTable != "" {
		if columns, exists := j.tableColumns[mainTable]; exists {
			for _, col := range columns {
				allColumns = append(allColumns, col.Name)
			}
		}
	}

	// Добавляем столбцы из присоединяемой таблицы
	if joinTable := j.joinTableSelect.Selected; joinTable != "" {
		if columns, exists := j.tableColumns[joinTable]; exists {
			for _, col := range columns {
				allColumns = append(allColumns, col.Name)
			}
		}
	}

	// Добавляем столбцы из дополнительных JOIN
	for _, item := range j.additionalJoins.Items {
		content := item.Detail.(*fyne.Container)
		if len(content.Objects) >= 2 {
			tableWidget := content.Objects[1].(*widget.Select)
			if tableWidget.Selected != "" {
				if columns, exists := j.tableColumns[tableWidget.Selected]; exists {
					for _, col := range columns {
						allColumns = append(allColumns, col.Name)
					}
				}
			}
		}
	}

	return allColumns
}

func (j *JoinBuilderWindow) addAdditionalJoin() {
	// Проверяем, что есть основная таблица
	if j.mainTableSelect.Selected == "" {
		j.showError(fmt.Errorf("сначала выберите основную таблицу"))
		return
	}

	// Создаем форму для дополнительного JOIN
	joinType := widget.NewSelect([]string{"INNER JOIN", "LEFT JOIN", "RIGHT JOIN", "FULL JOIN"}, nil)
	joinType.SetSelected("INNER JOIN")

	tableSelect := widget.NewSelect(j.tables, nil)
	tableSelect.PlaceHolder = "Таблица"

	mainColumn := widget.NewSelect([]string{}, nil)
	mainColumn.PlaceHolder = "Столбец из предыдущей таблицы"

	joinColumn := widget.NewSelect([]string{}, nil)
	joinColumn.PlaceHolder = "Столбец присоединяемой таблицы"

	// Загружаем столбцы когда выбирается таблица
	tableSelect.OnChanged = func(table string) {
		if table == "" {
			return
		}
		j.loadTableColumns(table)
		if cols, exists := j.tableColumns[table]; exists {
			var columnNames []string
			for _, col := range cols {
				columnNames = append(columnNames, col.Name)
			}
			joinColumn.Options = columnNames
			joinColumn.Refresh()
		}
	}

	// Обновляем список доступных столбцов при изменении
	updateMainColumns := func() {
		availableColumns := j.getAllAvailableColumns()
		mainColumn.Options = availableColumns
		mainColumn.Refresh()
	}

	// Обновляем при изменении основной таблицы или других JOIN
	j.mainTableSelect.OnChanged = func(string) { updateMainColumns() }
	j.joinTableSelect.OnChanged = func(string) { updateMainColumns() }

	removeBtn := widget.NewButton("✕", nil)

	joinForm := container.NewHBox(
		joinType,
		tableSelect,
		widget.NewLabel("ON"),
		mainColumn,
		widget.NewLabel("="),
		joinColumn,
		removeBtn,
	)

	item := widget.NewAccordionItem(fmt.Sprintf("Дополнительный JOIN %d", len(j.additionalJoins.Items)+1), joinForm)

	// Настраиваем кнопку удаления
	removeBtn.OnTapped = func() {
		items := j.additionalJoins.Items
		for i, it := range items {
			if it == item {
				j.additionalJoins.Items = append(items[:i], items[i+1:]...)
				j.additionalJoins.Refresh()
				break
			}
		}
	}

	j.additionalJoins.Append(item)
	j.additionalJoins.Refresh()

	// Инициализируем список столбцов
	updateMainColumns()
}

// Обновление дополнительных JOIN при изменении основной таблицы
func (j *JoinBuilderWindow) updateAdditionalJoins() {
	for _, item := range j.additionalJoins.Items {
		content := item.Detail.(*fyne.Container)
		if len(content.Objects) >= 4 {
			mainColWidget := content.Objects[3].(*widget.Select)
			availableColumns := j.getAllAvailableColumns()
			mainColWidget.Options = availableColumns
			mainColWidget.Refresh()
		}
	}
}

func (j *JoinBuilderWindow) buildJoinQuery() (string, error) {
	// Валидация формы
	if err := j.validateJoinForm(); err != nil {
		return "", err
	}

	mainTable := j.mainTableSelect.Selected
	joinTable := j.joinTableSelect.Selected
	joinType := strings.Replace(j.joinTypeSelect.Selected, " JOIN", "", 1)

	// Базовый JOIN
	query := fmt.Sprintf("SELECT * FROM %s %s %s ON %s.%s = %s.%s",
		mainTable, joinType, joinTable,
		mainTable, j.mainColumnSelect.Selected,
		joinTable, j.joinColumnSelect.Selected)

	// Добавляем дополнительные JOIN
	for i, item := range j.additionalJoins.Items {
		content := item.Detail.(*fyne.Container)
		if len(content.Objects) >= 6 {
			joinTypeWidget := content.Objects[0].(*widget.Select)
			tableWidget := content.Objects[1].(*widget.Select)
			mainColWidget := content.Objects[3].(*widget.Select)
			joinColWidget := content.Objects[5].(*widget.Select)

			if joinTypeWidget.Selected != "" && tableWidget.Selected != "" &&
				mainColWidget.Selected != "" && joinColWidget.Selected != "" {
				joinType := strings.Replace(joinTypeWidget.Selected, " JOIN", "", 1)
				query += fmt.Sprintf(" %s %s ON %s = %s.%s",
					joinType, tableWidget.Selected,
					mainColWidget.Selected, tableWidget.Selected, joinColWidget.Selected)
			} else {
				return "", fmt.Errorf("ошибка в дополнительном JOIN %d: не все поля заполнены", i+1)
			}
		}
	}

	query += " LIMIT 100"
	return query, nil
}

func (j *JoinBuilderWindow) executeJoin() {
	// Показываем индикатор загрузки
	j.resultLabel.SetText("Выполняется JOIN...")
	j.resultTable.Length = func() (int, int) { return 0, 0 }
	j.resultTable.Refresh()

	query, err := j.buildJoinQuery()
	if err != nil {
		j.showError(err)
		j.resultLabel.SetText("Ошибка в параметрах JOIN")
		return
	}

	j.sqlPreview.SetText(query)

	result, err := j.repository.ExecuteQuery(context.Background(), query)
	if err != nil {
		errorMsg := j.formatDatabaseError(err)
		j.showError(fmt.Errorf("ошибка при выполнении запроса: %s", errorMsg))
		j.resultLabel.SetText("Ошибка при выполнении JOIN")
		return
	}

	if result.Error != "" {
		errorMsg := j.formatDatabaseError(fmt.Errorf("%s", result.Error))
		j.resultLabel.SetText("Ошибка базы данных: " + errorMsg)
		return
	}

	j.displayResults(result)
}

// Форматирование ошибок базы данных
func (j *JoinBuilderWindow) formatDatabaseError(err error) string {
	errorStr := err.Error()

	if strings.Contains(errorStr, "syntax error") {
		return "синтаксическая ошибка в SQL запросе"
	}
	if strings.Contains(errorStr, "does not exist") {
		return "таблица или столбец не существует"
	}
	if strings.Contains(errorStr, "permission denied") {
		return "недостаточно прав для выполнения операции"
	}
	if strings.Contains(errorStr, "invalid input syntax") {
		return "некорректный синтаксис параметров"
	}
	if strings.Contains(errorStr, "ambiguous column") {
		return "неоднозначное имя столбца (столбец присутствует в нескольких таблицах)"
	}
	if strings.Contains(errorStr, "join") && strings.Contains(errorStr, "missing") {
		return "ошибка в условии JOIN: проверьте правильность указания таблиц и столбцов"
	}
	if strings.Contains(errorStr, "foreign key") {
		return "нарушение целостности внешнего ключа"
	}
	if strings.Contains(errorStr, "timeout") {
		return "превышено время выполнения запроса"
	}

	return "внутренняя ошибка базы данных"
}

func (j *JoinBuilderWindow) showError(err error) {
	customDialog := dialog.NewCustom(
		"Ошибка",
		"Закрыть",
		container.NewVBox(
			widget.NewLabel("❌ Произошла ошибка:"),
			widget.NewLabel(err.Error()),
			widget.NewLabel(""),
			widget.NewLabel("Проверьте введенные параметры и попробуйте снова."),
		),
		j.window,
	)
	customDialog.Show()
}

func (j *JoinBuilderWindow) Show() {
	j.window.Show()
}
