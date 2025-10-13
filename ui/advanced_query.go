package ui

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"testing-platform/db"
	"testing-platform/db/models"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

type AdvancedQueryWindow struct {
	window     fyne.Window
	repository *db.Repository
	mainWindow fyne.Window

	// Элементы SELECT
	tableSelect      *widget.Select
	columnList       *widget.CheckGroup
	whereContainer   *fyne.Container // Контейнер для условий WHERE
	orderByContainer *fyne.Container // Контейнер для ORDER BY условий
	groupByList      *widget.Select  // Список для GROUP BY
	havingContainer  *fyne.Container // Контейнер для HAVING
	limitSlider      *widget.Slider  // Слайдер для LIMIT
	limitLabel       *widget.Label   // Отображение значения LIMIT

	// Результаты
	resultTable *widget.Table
	resultLabel *widget.Label
	sqlPreview  *widget.Entry

	currentColumns    []models.ColumnInfo
	whereConditions   []WhereCondition   // Хранение условий WHERE
	orderByConditions []OrderByCondition // Хранение условий ORDER BY
	havingConditions  []WhereCondition   // Хранение условий HAVING
}

// Структура для хранения условий WHERE/HAVING
type WhereCondition struct {
	Column   string
	Operator string
	Value    string
}

// Структура для хранения условий ORDER BY
type OrderByCondition struct {
	Column    string
	Direction string
}

func NewAdvancedQueryWindow(repo *db.Repository, mainWindow fyne.Window) *AdvancedQueryWindow {
	a := &AdvancedQueryWindow{
		repository:        repo,
		mainWindow:        mainWindow,
		window:            fyne.CurrentApp().NewWindow("Расширенный SELECT"),
		whereConditions:   []WhereCondition{},
		orderByConditions: []OrderByCondition{},
		havingConditions:  []WhereCondition{},
	}

	a.buildUI()
	a.loadTables()
	return a
}

func (a *AdvancedQueryWindow) buildUI() {
	a.tableSelect = widget.NewSelect([]string{}, a.onTableSelected)
	a.tableSelect.PlaceHolder = "Выберите таблицу"

	a.columnList = widget.NewCheckGroup([]string{}, nil)
	a.columnList.Horizontal = false

	// Инициализация контейнеров для условий
	a.whereContainer = container.NewVBox()
	a.orderByContainer = container.NewVBox()
	a.havingContainer = container.NewVBox()

	// GROUP BY элементы
	a.groupByList = widget.NewSelect([]string{}, nil)
	a.groupByList.PlaceHolder = "Выберите столбец для группировки"

	// LIMIT элементы
	a.limitSlider = widget.NewSlider(1, 10000) // Увеличили максимальный лимит
	a.limitSlider.SetValue(100)
	a.limitLabel = widget.NewLabel("LIMIT: 100")
	a.limitSlider.OnChanged = func(value float64) {
		a.limitLabel.SetText(fmt.Sprintf("LIMIT: %d", int(value)))
	}

	a.sqlPreview = widget.NewMultiLineEntry()
	a.sqlPreview.Wrapping = fyne.TextWrapOff
	a.sqlPreview.SetPlaceHolder("Здесь будет показан SQL-запрос...")

	a.resultLabel = widget.NewLabel("Результаты появятся здесь")
	a.resultLabel.Wrapping = fyne.TextWrapWord

	// Таблица для результатов с переносом текста
	a.resultTable = widget.NewTable(
		func() (int, int) { return 0, 0 },
		func() fyne.CanvasObject {
			label := widget.NewLabel("")
			label.Wrapping = fyne.TextWrapWord
			return label
		},
		func(i widget.TableCellID, o fyne.CanvasObject) {},
	)

	// Кнопки для управления условиями
	addWhereBtn := widget.NewButton("Добавить условие WHERE", a.addWhereCondition)
	addOrderByBtn := widget.NewButton("Добавить сортировку ORDER BY", a.addOrderByCondition)
	addHavingBtn := widget.NewButton("Добавить условие HAVING", a.addHavingCondition)
	executeBtn := widget.NewButton("Выполнить запрос", a.executeQuery)
	clearBtn := widget.NewButton("Очистить всё", a.clearForm)
	showSQLBtn := widget.NewButton("Показать SQL", a.previewSQL)

	// Добавляем подсказки
	hintLabel := widget.NewLabel("💡 Подсказки:\n• Для LIKE используйте % для поиска частей текста\n• Для IN перечислите значения через запятую\n• IS NULL и IS NOT NULL не требуют значения")
	hintLabel.Wrapping = fyne.TextWrapWord

	// Компоновка
	leftPanel := container.NewVBox(
		widget.NewLabel("Таблица:"),
		a.tableSelect,
		widget.NewLabel("Столбцы:"),
		container.NewScroll(a.columnList),
	)

	conditionsPanel := container.NewVBox(
		widget.NewLabel("Условия WHERE:"),
		a.whereContainer,
		addWhereBtn,
		widget.NewSeparator(),
		widget.NewLabel("Сортировка ORDER BY:"),
		a.orderByContainer,
		addOrderByBtn,
		widget.NewSeparator(),
		widget.NewLabel("GROUP BY:"),
		a.groupByList,
		widget.NewLabel("Условия HAVING:"),
		a.havingContainer,
		addHavingBtn,
		widget.NewSeparator(),
		a.limitLabel,
		a.limitSlider,
		hintLabel,
	)

	// Создаем HBox для кнопок
	buttonsContainer := container.NewHBox(executeBtn, showSQLBtn, clearBtn)

	rightPanel := container.NewVBox(
		conditionsPanel,
		buttonsContainer,
	)

	// Создаем HBox для основного расположения
	controls := container.NewHBox(leftPanel, rightPanel)

	content := container.NewBorder(
		controls,
		container.NewVBox(a.resultLabel, widget.NewLabel("SQL:"), a.sqlPreview),
		nil, nil,
		container.NewScroll(a.resultTable),
	)

	a.window.SetContent(content)
	a.window.Resize(fyne.NewSize(1200, 800))
}

// Валидация значения для условия WHERE/HAVING
func (a *AdvancedQueryWindow) validateConditionValue(operator, value string) error {
	if value == "" {
		return nil // Пустое значение допустимо
	}

	// Проверка длины значения
	if len(value) > 1000 {
		return fmt.Errorf("значение слишком длинное (максимум 1000 символов)")
	}

	// Проверка на опасные SQL-инъекции
	dangerousPatterns := []string{
		";", "--", "/*", "*/", "xp_", "sp_", "exec ", "union ", "select ", "insert ",
		"update ", "delete ", "drop ", "create ", "alter ", "grant ", "revoke ",
	}

	lowerValue := strings.ToLower(value)
	for _, dangerous := range dangerousPatterns {
		if strings.Contains(lowerValue, dangerous) {
			return fmt.Errorf("значение содержит потенциально опасные символы")
		}
	}

	// Специфическая валидация для разных операторов
	switch operator {
	case "IN", "NOT IN":
		// Проверяем, что значение содержит только разрешенные символы для списка
		if matched, _ := regexp.MatchString(`^[a-zA-Z0-9_,\.\s]+$`, value); !matched {
			return fmt.Errorf("для оператора IN используйте только буквы, цифры, запятые и точки")
		}
	case "LIKE", "NOT LIKE":
		// Проверяем корректное использование % и _
		if strings.Count(value, "%") > 10 {
			return fmt.Errorf("слишком много символов %% (максимум 10)")
		}
		if strings.Count(value, "_") > 20 {
			return fmt.Errorf("слишком много символов _ (максимум 20)")
		}
	}

	return nil
}

// Валидация всего запроса перед выполнением
func (a *AdvancedQueryWindow) validateQuery() error {
	// Проверка выбора таблицы
	if a.tableSelect.Selected == "" {
		return fmt.Errorf("не выбрана таблица для запроса")
	}

	// Проверка условий WHERE
	for i, condition := range a.whereConditions {
		if condition.Column == "" {
			return fmt.Errorf("в условии WHERE №%d не выбран столбец", i+1)
		}
		if condition.Operator == "" {
			return fmt.Errorf("в условии WHERE №%d не выбран оператор", i+1)
		}
		// Для операторов, требующих значения
		if condition.Operator != "IS NULL" && condition.Operator != "IS NOT NULL" {
			if strings.TrimSpace(condition.Value) == "" {
				return fmt.Errorf("в условии WHERE №%d не указано значение", i+1)
			}
			if err := a.validateConditionValue(condition.Operator, condition.Value); err != nil {
				return fmt.Errorf("ошибка в условии WHERE №%d: %v", i+1, err)
			}
		}
	}

	// Проверка условий HAVING
	for i, condition := range a.havingConditions {
		if condition.Column == "" {
			return fmt.Errorf("в условии HAVING №%d не выбран столбец", i+1)
		}
		if condition.Operator == "" {
			return fmt.Errorf("в условии HAVING №%d не выбран оператор", i+1)
		}
		// Для операторов, требующих значения
		if condition.Operator != "IS NULL" && condition.Operator != "IS NOT NULL" {
			if strings.TrimSpace(condition.Value) == "" {
				return fmt.Errorf("в условии HAVING №%d не указано значение", i+1)
			}
			if err := a.validateConditionValue(condition.Operator, condition.Value); err != nil {
				return fmt.Errorf("ошибка в условии HAVING №%d: %v", i+1, err)
			}
		}
	}

	// Проверка условий ORDER BY
	for i, condition := range a.orderByConditions {
		if condition.Column == "" {
			return fmt.Errorf("в условии сортировки №%d не выбран столбец", i+1)
		}
	}

	// Проверка GROUP BY
	if a.groupByList.Selected != "" {
		// Проверяем, что выбранный столбец существует в текущих столбцах
		found := false
		for _, col := range a.currentColumns {
			if col.Name == a.groupByList.Selected {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("выбранный столбец для GROUP BY не существует в таблице")
		}
	}

	return nil
}

func (a *AdvancedQueryWindow) addWhereCondition() {
	a.addCondition(a.whereContainer, &a.whereConditions, "WHERE")
}

func (a *AdvancedQueryWindow) addOrderByCondition() {
	a.addOrderBy()
}

func (a *AdvancedQueryWindow) addHavingCondition() {
	a.addCondition(a.havingContainer, &a.havingConditions, "HAVING")
}

func (a *AdvancedQueryWindow) addCondition(cont *fyne.Container, conditions *[]WhereCondition, conditionType string) {
	// Создаем элементы для одного условия
	columnSelect := widget.NewSelect([]string{}, nil)
	columnSelect.PlaceHolder = "Столбец"

	// Если столбцы уже загружены, обновляем список
	if len(a.currentColumns) > 0 {
		columnSelect.Options = a.getColumnNames()
		columnSelect.Refresh()
	}

	operatorSelect := widget.NewSelect([]string{
		"=", "!=", ">", "<", ">=", "<=", "LIKE", "NOT LIKE",
		"IN", "NOT IN", "IS NULL", "IS NOT NULL",
	}, nil)
	operatorSelect.SetSelected("=")

	valueEntry := widget.NewEntry()
	valueEntry.SetPlaceHolder("Значение")

	// Добавляем валидацию для поля значения
	valueEntry.Validator = func(text string) error {
		selectedOperator := operatorSelect.Selected
		if selectedOperator == "" {
			selectedOperator = "=" // Значение по умолчанию
		}
		return a.validateConditionValue(selectedOperator, text)
	}

	// Кнопка удаления условия
	deleteBtn := widget.NewButton("✕", nil)

	// Создаем HBox для строки условия
	conditionRow := container.NewHBox(
		columnSelect,
		operatorSelect,
		valueEntry,
		deleteBtn,
	)

	// Создаем условие
	condition := WhereCondition{}
	*conditions = append(*conditions, condition)
	conditionIndex := len(*conditions) - 1

	// Обновляем условие при изменении полей
	updateCondition := func() {
		if conditionIndex < len(*conditions) {
			(*conditions)[conditionIndex] = WhereCondition{
				Column:   columnSelect.Selected,
				Operator: operatorSelect.Selected,
				Value:    valueEntry.Text,
			}
		}
	}

	columnSelect.OnChanged = func(s string) { updateCondition() }
	operatorSelect.OnChanged = func(s string) {
		updateCondition()
		// Обновляем валидацию при изменении оператора
		valueEntry.Validate()
	}
	valueEntry.OnChanged = func(s string) {
		updateCondition()
		valueEntry.Validate()
	}

	// Настраиваем кнопку удаления
	deleteBtn.OnTapped = func() {
		if conditionIndex < len(*conditions) {
			// Удаляем условие из слайса
			*conditions = append((*conditions)[:conditionIndex], (*conditions)[conditionIndex+1:]...)
			// Удаляем строку из контейнера
			cont.Remove(conditionRow)
		}
	}

	cont.Add(conditionRow)
}

func (a *AdvancedQueryWindow) addOrderBy() {
	// Создаем элементы для сортировки
	columnSelect := widget.NewSelect([]string{}, nil)
	columnSelect.PlaceHolder = "Столбец"

	// Если столбцы уже загружены, обновляем список
	if len(a.currentColumns) > 0 {
		columnSelect.Options = a.getColumnNames()
		columnSelect.Refresh()
	}

	directionSelect := widget.NewSelect([]string{
		"По возрастанию (ASC)",
		"По убыванию (DESC)",
		"Случайно (RANDOM)",
		"По длине строки (LENGTH)",
		"Без учета регистра (CASE INSENSITIVE)",
	}, nil)
	directionSelect.SetSelected("По возрастанию (ASC)")

	// Кнопка удаления условия
	deleteBtn := widget.NewButton("✕", nil)

	// Создаем HBox для строки сортировки
	orderByRow := container.NewHBox(
		columnSelect,
		directionSelect,
		deleteBtn,
	)

	// Создаем условие
	condition := OrderByCondition{}
	a.orderByConditions = append(a.orderByConditions, condition)
	conditionIndex := len(a.orderByConditions) - 1

	// Обновляем условие при изменении полей
	updateCondition := func() {
		if conditionIndex < len(a.orderByConditions) {
			// Преобразуем понятное название в SQL направление
			direction := a.getSQLDirection(directionSelect.Selected)
			a.orderByConditions[conditionIndex] = OrderByCondition{
				Column:    columnSelect.Selected,
				Direction: direction,
			}
		}
	}

	columnSelect.OnChanged = func(s string) { updateCondition() }
	directionSelect.OnChanged = func(s string) { updateCondition() }

	// Настраиваем кнопку удаления
	deleteBtn.OnTapped = func() {
		if conditionIndex < len(a.orderByConditions) {
			// Удаляем условие из слайса
			a.orderByConditions = append(a.orderByConditions[:conditionIndex], a.orderByConditions[conditionIndex+1:]...)
			// Удаляем строку из контейнера
			a.orderByContainer.Remove(orderByRow)
		}
	}

	a.orderByContainer.Add(orderByRow)
}

// Преобразует понятное название направления в SQL синтаксис
func (a *AdvancedQueryWindow) getSQLDirection(displayDirection string) string {
	switch displayDirection {
	case "По убыванию (DESC)":
		return "DESC"
	case "Случайно (RANDOM)":
		return "RANDOM()"
	case "По длине строки (LENGTH)":
		return "LENGTH"
	case "Без учета регистра (CASE INSENSITIVE)":
		return "COLLATE NOCASE"
	default: // "По возрастанию (ASC)"
		return "ASC"
	}
}

// Форматирует условие ORDER BY для SQL запроса
func (a *AdvancedQueryWindow) formatOrderByCondition(condition OrderByCondition) string {
	if condition.Column == "" {
		return ""
	}

	switch condition.Direction {
	case "RANDOM()":
		return "RANDOM()"
	case "LENGTH":
		return fmt.Sprintf("LENGTH(%s)", condition.Column)
	case "COLLATE NOCASE":
		return fmt.Sprintf("%s COLLATE NOCASE", condition.Column)
	default:
		return fmt.Sprintf("%s %s", condition.Column, condition.Direction)
	}
}

func (a *AdvancedQueryWindow) getColumnNames() []string {
	var names []string
	for _, col := range a.currentColumns {
		names = append(names, col.Name)
	}
	return names
}

func (a *AdvancedQueryWindow) loadTables() {
	tables, err := a.repository.GetTables(context.Background())
	if err != nil {
		a.showError(fmt.Errorf("не удалось загрузить список таблиц: проверьте подключение к базе данных"))
		return
	}

	if len(tables) == 0 {
		a.showError(fmt.Errorf("в базе данных не найдено ни одной таблицы"))
		return
	}

	a.tableSelect.Options = tables
	a.tableSelect.Refresh()
}

func (a *AdvancedQueryWindow) onTableSelected(table string) {
	if table == "" {
		return
	}

	columns, err := a.repository.GetTableColumns(context.Background(), table)
	if err != nil {
		a.showError(fmt.Errorf("не удалось загрузить столбцы таблицы '%s': проверьте права доступа", table))
		return
	}

	a.currentColumns = columns
	var columnNames []string
	for _, col := range columns {
		columnNames = append(columnNames, col.Name)
	}

	if len(columnNames) == 0 {
		a.showError(fmt.Errorf("в таблице '%s' не найдено столбцов", table))
		return
	}

	// Обновляем списки выбора
	a.columnList.Options = columnNames
	a.columnList.Selected = columnNames // Выбираем все по умолчанию
	a.columnList.Refresh()

	a.groupByList.Options = columnNames
	a.groupByList.Refresh()

	// Обновляем существующие условия
	a.updateExistingConditions()
}

// Метод для обновления существующих условий
func (a *AdvancedQueryWindow) updateExistingConditions() {
	columnNames := a.getColumnNames()

	// Обновляем условия WHERE
	for i := range a.whereConditions {
		if i < len(a.whereContainer.Objects) {
			if conditionRow, ok := a.whereContainer.Objects[i].(*fyne.Container); ok {
				if columnSelect, ok := conditionRow.Objects[0].(*widget.Select); ok {
					columnSelect.Options = columnNames
					columnSelect.Refresh()
				}
			}
		}
	}

	// Обновляем условия ORDER BY
	for i := range a.orderByConditions {
		if i < len(a.orderByContainer.Objects) {
			if orderByRow, ok := a.orderByContainer.Objects[i].(*fyne.Container); ok {
				if columnSelect, ok := orderByRow.Objects[0].(*widget.Select); ok {
					columnSelect.Options = columnNames
					columnSelect.Refresh()
				}
			}
		}
	}

	// Обновляем условия HAVING
	for i := range a.havingConditions {
		if i < len(a.havingContainer.Objects) {
			if conditionRow, ok := a.havingContainer.Objects[i].(*fyne.Container); ok {
				if columnSelect, ok := conditionRow.Objects[0].(*widget.Select); ok {
					columnSelect.Options = columnNames
					columnSelect.Refresh()
				}
			}
		}
	}
}

func (a *AdvancedQueryWindow) buildQuery() (string, error) {
	// Валидация перед построением запроса
	if err := a.validateQuery(); err != nil {
		return "", err
	}

	table := a.tableSelect.Selected

	// SELECT часть
	var selectedColumns string
	if len(a.columnList.Selected) == 0 {
		selectedColumns = "*"
	} else {
		selectedColumns = strings.Join(a.columnList.Selected, ", ")
	}

	query := fmt.Sprintf("SELECT %s FROM %s", selectedColumns, table)

	// WHERE условия
	whereClause, err := a.buildConditions(a.whereConditions)
	if err != nil {
		return "", fmt.Errorf("ошибка в условиях WHERE: %v", err)
	}
	if whereClause != "" {
		query += " WHERE " + whereClause
	}

	// GROUP BY
	if a.groupByList.Selected != "" {
		query += " GROUP BY " + a.groupByList.Selected
	}

	// HAVING условия
	havingClause, err := a.buildConditions(a.havingConditions)
	if err != nil {
		return "", fmt.Errorf("ошибка в условиях HAVING: %v", err)
	}
	if havingClause != "" {
		query += " HAVING " + havingClause
	}

	// ORDER BY условия
	orderByClause := a.buildOrderByConditions()
	if orderByClause != "" {
		query += " ORDER BY " + orderByClause
	}

	// LIMIT
	limitValue := int(a.limitSlider.Value)
	if limitValue > 0 {
		query += " LIMIT " + strconv.Itoa(limitValue)
	}

	return query, nil
}

func (a *AdvancedQueryWindow) buildConditions(conditions []WhereCondition) (string, error) {
	if len(conditions) == 0 {
		return "", nil
	}

	var conditionStrings []string
	for i, cond := range conditions {
		if cond.Column == "" || cond.Operator == "" {
			return "", fmt.Errorf("условие №%d: не указан столбец или оператор", i+1)
		}

		// Форматируем значение в зависимости от оператора
		var valueStr string
		switch cond.Operator {
		case "IS NULL", "IS NOT NULL":
			valueStr = "" // Эти операторы не требуют значения
		case "IN", "NOT IN":
			// Предполагаем, что значение - это список, разделенный запятыми
			// Экранируем значения
			values := strings.Split(cond.Value, ",")
			var escapedValues []string
			for _, v := range values {
				trimmed := strings.TrimSpace(v)
				if trimmed != "" {
					// Проверяем, является ли значение числом
					if _, err := strconv.Atoi(trimmed); err == nil {
						escapedValues = append(escapedValues, trimmed)
					} else {
						// Экранируем строковые значения
						escapedValue := strings.ReplaceAll(trimmed, "'", "''")
						escapedValues = append(escapedValues, "'"+escapedValue+"'")
					}
				}
			}
			if len(escapedValues) == 0 {
				return "", fmt.Errorf("условие №%d: для оператора IN укажите значения через запятую", i+1)
			}
			valueStr = "(" + strings.Join(escapedValues, ", ") + ")"
		default:
			// Для строковых значений добавляем кавычки
			if _, err := strconv.Atoi(cond.Value); err != nil {
				// Если не число, обрамляем кавычками и экранируем
				escapedValue := strings.ReplaceAll(cond.Value, "'", "''")
				valueStr = "'" + escapedValue + "'"
			} else {
				valueStr = cond.Value
			}
		}

		conditionStr := cond.Column + " " + cond.Operator
		if valueStr != "" {
			conditionStr += " " + valueStr
		}
		conditionStrings = append(conditionStrings, conditionStr)
	}

	return strings.Join(conditionStrings, " AND "), nil
}

func (a *AdvancedQueryWindow) buildOrderByConditions() string {
	if len(a.orderByConditions) == 0 {
		return ""
	}

	var orderByStrings []string
	for _, cond := range a.orderByConditions {
		if cond.Column == "" {
			continue
		}
		orderByStrings = append(orderByStrings, a.formatOrderByCondition(cond))
	}

	return strings.Join(orderByStrings, ", ")
}

func (a *AdvancedQueryWindow) executeQuery() {
	// Показываем индикатор загрузки
	a.resultLabel.SetText("Выполняется запрос...")
	a.resultTable.Length = func() (int, int) { return 0, 0 }
	a.resultTable.Refresh()

	query, err := a.buildQuery()
	if err != nil {
		a.showError(err)
		a.resultLabel.SetText("Ошибка в параметрах запроса")
		return
	}

	a.sqlPreview.SetText(query)

	result, err := a.repository.ExecuteQuery(context.Background(), query)
	if err != nil {
		errorMsg := a.formatDatabaseError(err)
		a.showError(fmt.Errorf("ошибка при выполнении запроса: %s", errorMsg))
		a.resultLabel.SetText("Ошибка при выполнении запроса")
		return
	}

	if result.Error != "" {
		errorMsg := a.formatDatabaseError(fmt.Errorf("result.Error"))
		a.resultLabel.SetText("Ошибка базы данных: " + errorMsg)
		return
	}

	a.displayResults(result)
}

// Форматирование ошибок базы данных в понятный вид
func (a *AdvancedQueryWindow) formatDatabaseError(err error) string {
	errorStr := err.Error()

	// PostgreSQL ошибки
	if strings.Contains(errorStr, "syntax error") {
		return "синтаксическая ошибка в запросе"
	}
	if strings.Contains(errorStr, "does not exist") {
		return "таблица или столбец не существует"
	}
	if strings.Contains(errorStr, "permission denied") {
		return "недостаточно прав для выполнения операции"
	}
	if strings.Contains(errorStr, "timeout") {
		return "превышено время ожидания запроса"
	}
	if strings.Contains(errorStr, "connection") {
		return "проблема с подключением к базе данных"
	}
	if strings.Contains(errorStr, "division by zero") {
		return "деление на ноль в запросе"
	}
	if strings.Contains(errorStr, "grouping error") {
		return "ошибка в GROUP BY: выбранные столбцы должны быть в GROUP BY или агрегатных функциях"
	}
	if strings.Contains(errorStr, "aggregate functions") {
		return "неправильное использование агрегатных функций"
	}

	return "внутренняя ошибка базы данных"
}

func (a *AdvancedQueryWindow) displayResults(result *models.QueryResult) {
	if len(result.Rows) == 0 {
		a.resultTable.Length = func() (int, int) { return 1, 1 }
		a.resultTable.UpdateCell = func(id widget.TableCellID, obj fyne.CanvasObject) {
			label := obj.(*widget.Label)
			label.Wrapping = fyne.TextWrapWord
			if id.Row == 0 && id.Col == 0 {
				label.SetText("Нет данных")
			}
		}
		a.resultLabel.SetText("Запрос выполнен успешно. Найдено 0 строк.")
		return
	}

	// Настройка таблицы
	a.resultTable.Length = func() (int, int) {
		return len(result.Rows) + 1, len(result.Columns)
	}

	// Автоматически настраиваем ширину колонок
	for col := 0; col < len(result.Columns); col++ {
		maxWidth := float32(120) // Минимальная ширина

		// Учитываем ширину заголовка
		headerWidth := float32(len(result.Columns[col])) * 8
		if headerWidth > maxWidth {
			maxWidth = headerWidth
		}

		// Проверяем данные в первых 20 строках для определения ширины
		for row := 0; row < len(result.Rows) && row < 20; row++ {
			if value := result.Rows[row][result.Columns[col]]; value != nil {
				text := fmt.Sprintf("%v", value)
				textWidth := float32(len(text)) * 7
				if textWidth > maxWidth {
					maxWidth = textWidth
				}
			}
		}

		// Ограничиваем максимальную ширину
		if maxWidth > 400 {
			maxWidth = 400
		}
		a.resultTable.SetColumnWidth(col, maxWidth)
	}

	a.resultTable.UpdateCell = func(id widget.TableCellID, obj fyne.CanvasObject) {
		label := obj.(*widget.Label)
		label.Wrapping = fyne.TextWrapWord

		if id.Row == 0 {
			// Заголовки
			if id.Col < len(result.Columns) {
				label.SetText(result.Columns[id.Col])
				label.TextStyle = fyne.TextStyle{Bold: true}
			}
		} else {
			// Данные
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

	a.resultLabel.SetText(fmt.Sprintf("Найдено %d строк", len(result.Rows)))
	a.resultTable.Refresh()
}

func (a *AdvancedQueryWindow) previewSQL() {
	query, err := a.buildQuery()
	if err != nil {
		a.showError(err)
		a.sqlPreview.SetText("Невозможно показать SQL: " + err.Error())
		return
	}
	a.sqlPreview.SetText(query)
}

func (a *AdvancedQueryWindow) clearForm() {
	a.tableSelect.SetSelected("")
	a.columnList.Selected = []string{}
	a.whereContainer.Objects = nil
	a.orderByContainer.Objects = nil
	a.havingContainer.Objects = nil
	a.groupByList.SetSelected("")
	a.limitSlider.SetValue(100)
	a.limitLabel.SetText("LIMIT: 100")
	a.sqlPreview.SetText("")
	a.resultLabel.SetText("Результаты появятся здесь")
	a.resultTable.Length = func() (int, int) { return 0, 0 }
	a.resultTable.Refresh()

	a.whereConditions = []WhereCondition{}
	a.orderByConditions = []OrderByCondition{}
	a.havingConditions = []WhereCondition{}

	a.currentColumns = []models.ColumnInfo{}
}

func (a *AdvancedQueryWindow) showError(err error) {
	// Используем кастомный диалог с более понятным сообщением
	customDialog := dialog.NewCustom(
		"Ошибка",
		"Закрыть",
		container.NewVBox(
			widget.NewLabel("❌ Произошла ошибка:"),
			widget.NewLabel(err.Error()),
			widget.NewLabel(""),
			widget.NewLabel("Проверьте введенные данные и попробуйте снова."),
		),
		a.window,
	)
	customDialog.Show()
}

func (a *AdvancedQueryWindow) Show() {
	a.window.Show()
}
