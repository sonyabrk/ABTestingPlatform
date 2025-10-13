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

	tables       []string
	tableColumns map[string][]models.ColumnInfo
}

func NewJoinBuilderWindow(repo *db.Repository, mainWindow fyne.Window) *JoinBuilderWindow {
	j := &JoinBuilderWindow{
		repository:   repo,
		mainWindow:   mainWindow,
		window:       fyne.CurrentApp().NewWindow("Мастер JOIN"),
		tableColumns: make(map[string][]models.ColumnInfo),
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

	// Добавляем подсказки
	hintLabel := widget.NewLabel("💡 Подсказки:\n• Сначала выберите основную таблицу\n• Затем выберите таблицу для JOIN и столбцы для связи\n• INNER JOIN - только совпадающие строки\n• LEFT JOIN - все строки из левой таблицы\n• RIGHT JOIN - все строки из правой таблицы\n• FULL JOIN - все строки из обеих таблиц")
	hintLabel.Wrapping = fyne.TextWrapWord

	// Компоновка
	joinForm := container.NewVBox(
		widget.NewLabel("Основная таблица:"),
		j.mainTableSelect,
		widget.NewLabel("Тип JOIN:"),
		j.joinTypeSelect,
		widget.NewLabel("Присоединяемая таблица:"),
		j.joinTableSelect,
		widget.NewLabel("Столбец основной таблицы:"),
		j.mainColumnSelect,
		widget.NewLabel("Столбец присоединяемой таблицы:"),
		j.joinColumnSelect,
	)

	formContainer := container.NewVScroll(container.NewVBox(
		joinForm,
		hintLabel,
		addJoinBtn,
		j.additionalJoins,
		container.NewHBox(executeBtn, clearBtn),
		widget.NewLabel("SQL запрос:"),
		j.sqlPreview,
	))

	// Ограничиваем минимальную высоту формы
	formContainer.SetMinSize(fyne.NewSize(0, 300))

	// Создаем контейнер для результатов
	resultContainer := container.NewBorder(
		j.resultLabel, nil, nil, nil,
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
		errorMsg := j.formatDatabaseError(fmt.Errorf("result.Error"))
		j.resultLabel.SetText("Ошибка базы данных: " + errorMsg)
		return
	}

	j.displayResults(result)
	j.resultLabel.SetText(fmt.Sprintf("Найдено %d строк", len(result.Rows)))
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

func (j *JoinBuilderWindow) displayResults(result *models.QueryResult) {
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

	j.resultTable.Length = func() (int, int) {
		return len(result.Rows) + 1, len(result.Columns)
	}

	// Автоматическая настройка ширины колонок
	for col := 0; col < len(result.Columns); col++ {
		maxWidth := float32(150)
		headerWidth := float32(len(result.Columns[col])) * 8
		if headerWidth > maxWidth {
			maxWidth = headerWidth
		}
		for row := 0; row < len(result.Rows) && row < 10; row++ {
			if value := result.Rows[row][result.Columns[col]]; value != nil {
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
			if id.Col < len(result.Columns) {
				label.SetText(result.Columns[id.Col])
				label.TextStyle = fyne.TextStyle{Bold: true}
			}
		} else {
			rowIndex := id.Row - 1
			if rowIndex < len(result.Rows) && id.Col < len(result.Columns) {
				value := result.Rows[rowIndex][result.Columns[id.Col]]
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
