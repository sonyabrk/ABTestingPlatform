package ui

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing-platform/db"
	"testing-platform/db/models"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

type StringFunctionsWindow struct {
	window     fyne.Window
	repository *db.Repository
	mainWindow fyne.Window

	tableSelect    *widget.Select
	columnSelect   *widget.Select
	functionSelect *widget.Select

	trimCharInput   *widget.Entry // Для TRIM, LTRIM, RTRIM
	substringStart  *widget.Entry // Для SUBSTRING
	substringLength *widget.Entry // Для SUBSTRING
	replaceOld      *widget.Entry // Для REPLACE
	replaceNew      *widget.Entry // Для REPLACE
	concatSeparator *widget.Entry // Для CONCAT_WS
	concatSecond    *widget.Entry // Для CONCAT и ||
	concatThird     *widget.Entry // Для CONCAT и ||
	lpadLength      *widget.Entry // Для LPAD, RPAD
	lpadFillChar    *widget.Entry // Для LPAD, RPAD

	// Контейнеры для группировки полей
	trimContainer      *fyne.Container
	substringContainer *fyne.Container
	replaceContainer   *fyne.Container
	concatContainer    *fyne.Container
	lpadContainer      *fyne.Container

	previewLabel *widget.Label
	resultTable  *widget.Table
	resultLabel  *widget.Label

	currentColumns []models.ColumnInfo
}

func NewStringFunctionsWindow(repo *db.Repository, mainWindow fyne.Window) *StringFunctionsWindow {
	s := &StringFunctionsWindow{
		repository: repo,
		mainWindow: mainWindow,
		window:     fyne.CurrentApp().NewWindow("Функции работы со строками"),
	}

	s.buildUI()
	s.loadTables()
	return s
}

func (s *StringFunctionsWindow) buildUI() {
	s.tableSelect = widget.NewSelect([]string{}, s.onTableSelected)
	s.tableSelect.PlaceHolder = "Выберите таблицу"

	s.columnSelect = widget.NewSelect([]string{}, nil)
	s.columnSelect.PlaceHolder = "Выберите столбец"

	s.functionSelect = widget.NewSelect([]string{
		"UPPER", "LOWER", "LENGTH", "TRIM", "LTRIM", "RTRIM",
		"SUBSTRING", "REPLACE", "CONCAT", "CONCAT_WS", "LPAD", "RPAD",
		"CONCAT (operator ||)",
	}, s.onFunctionSelected)
	s.functionSelect.PlaceHolder = "Выберите функцию"

	// Создаем специфичные поля ввода
	s.createFunctionSpecificInputs()

	s.previewLabel = widget.NewLabel("Выберите функцию для просмотра примера")
	s.previewLabel.Wrapping = fyne.TextWrapWord

	s.resultLabel = widget.NewLabel("")
	s.resultLabel.Wrapping = fyne.TextWrapWord

	// Таблица с переносом текста
	s.resultTable = widget.NewTable(
		func() (int, int) { return 0, 0 },
		func() fyne.CanvasObject {
			label := widget.NewLabel("")
			label.Wrapping = fyne.TextWrapWord
			return label
		},
		func(i widget.TableCellID, o fyne.CanvasObject) {},
	)

	applyBtn := widget.NewButton("Применить функцию", s.applyFunction)
	previewBtn := widget.NewButton("Показать SQL", s.previewSQL)
	clearBtn := widget.NewButton("Очистить", s.clearForm)

	// Компоновка
	form := container.NewVBox(
		widget.NewLabel("Таблица:"),
		s.tableSelect,
		widget.NewLabel("Столбец:"),
		s.columnSelect,
		widget.NewLabel("Функция:"),
		s.functionSelect,

		// Контейнеры для специфичных полей (изначально скрыты)
		s.trimContainer,
		s.substringContainer,
		s.replaceContainer,
		s.concatContainer,
		s.lpadContainer,

		container.NewHBox(applyBtn, previewBtn, clearBtn),
		s.previewLabel,
		s.resultLabel,
	)

	content := container.NewBorder(
		form, nil, nil, nil,
		container.NewScroll(s.resultTable),
	)

	s.window.SetContent(content)
	s.window.Resize(fyne.NewSize(1000, 700))
}

func (s *StringFunctionsWindow) createFunctionSpecificInputs() {
	// TRIM, LTRIM, RTRIM поля
	s.trimCharInput = widget.NewEntry()
	s.trimCharInput.SetPlaceHolder("Символ для удаления (оставьте пустым для пробела)")
	s.trimCharInput.Validator = s.validateTrimChar

	s.trimContainer = container.NewVBox(
		widget.NewLabel("Настройки TRIM:"),
		s.trimCharInput,
	)
	s.trimContainer.Hide()

	// SUBSTRING поля
	s.substringStart = widget.NewEntry()
	s.substringStart.SetPlaceHolder("Начальная позиция (обязательно)")
	s.substringStart.Validator = s.validateSubstringStart

	s.substringLength = widget.NewEntry()
	s.substringLength.SetPlaceHolder("Длина подстроки (опционально)")
	s.substringLength.Validator = s.validateSubstringLength

	s.substringContainer = container.NewVBox(
		widget.NewLabel("Настройки SUBSTRING:"),
		widget.NewLabel("Начальная позиция:"),
		s.substringStart,
		widget.NewLabel("Длина:"),
		s.substringLength,
	)
	s.substringContainer.Hide()

	// REPLACE поля
	s.replaceOld = widget.NewEntry()
	s.replaceOld.SetPlaceHolder("Заменяемая подстрока")
	s.replaceOld.Validator = s.validateReplaceOld

	s.replaceNew = widget.NewEntry()
	s.replaceNew.SetPlaceHolder("Новая подстрока")
	s.replaceNew.Validator = s.validateReplaceNew

	s.replaceContainer = container.NewVBox(
		widget.NewLabel("Настройки REPLACE:"),
		widget.NewLabel("Заменить:"),
		s.replaceOld,
		widget.NewLabel("На:"),
		s.replaceNew,
	)
	s.replaceContainer.Hide()

	// CONCAT поля
	s.concatSeparator = widget.NewEntry()
	s.concatSeparator.SetPlaceHolder("Разделитель (для CONCAT_WS)")
	s.concatSeparator.Validator = s.validateConcatSeparator

	s.concatSecond = widget.NewEntry()
	s.concatSecond.SetPlaceHolder("Вторая строка")
	s.concatSecond.Validator = s.validateConcatString

	s.concatThird = widget.NewEntry()
	s.concatThird.SetPlaceHolder("Третья строка (опционально)")
	s.concatThird.Validator = s.validateConcatString

	s.concatContainer = container.NewVBox(
		widget.NewLabel("Настройки CONCAT:"),
		widget.NewLabel("Вторая строка:"),
		s.concatSecond,
		widget.NewLabel("Третья строка:"),
		s.concatThird,
		widget.NewLabel("Разделитель (только для CONCAT_WS):"),
		s.concatSeparator,
	)
	s.concatContainer.Hide()

	// LPAD/RPAD поля
	s.lpadLength = widget.NewEntry()
	s.lpadLength.SetPlaceHolder("Длина результирующей строки")
	s.lpadLength.Validator = s.validateLpadLength

	s.lpadFillChar = widget.NewEntry()
	s.lpadFillChar.SetPlaceHolder("Символ заполнения (по умолчанию пробел)")
	s.lpadFillChar.Validator = s.validateLpadFillChar

	s.lpadContainer = container.NewVBox(
		widget.NewLabel("Настройки LPAD/RPAD:"),
		widget.NewLabel("Длина:"),
		s.lpadLength,
		widget.NewLabel("Символ заполнения:"),
		s.lpadFillChar,
	)
	s.lpadContainer.Hide()
}

// Валидаторы для специфичных полей
func (s *StringFunctionsWindow) validateTrimChar(text string) error {
	if text == "" {
		return nil
	}
	if len(text) > 1 {
		return fmt.Errorf("укажите только один символ")
	}
	return s.validateSQLSafety(text)
}

func (s *StringFunctionsWindow) validateSubstringStart(text string) error {
	if text == "" {
		return fmt.Errorf("укажите начальную позицию")
	}
	if _, err := strconv.Atoi(text); err != nil {
		return fmt.Errorf("должно быть целым числом")
	}
	if num, _ := strconv.Atoi(text); num < 1 {
		return fmt.Errorf("позиция должна быть больше 0")
	}
	return nil
}

func (s *StringFunctionsWindow) validateSubstringLength(text string) error {
	if text == "" {
		return nil
	}
	if _, err := strconv.Atoi(text); err != nil {
		return fmt.Errorf("должно быть целым числом")
	}
	if num, _ := strconv.Atoi(text); num < 1 {
		return fmt.Errorf("длина должна быть больше 0")
	}
	return nil
}

func (s *StringFunctionsWindow) validateReplaceOld(text string) error {
	if text == "" {
		return fmt.Errorf("укажите заменяемую подстроку")
	}
	return s.validateSQLSafety(text)
}

func (s *StringFunctionsWindow) validateReplaceNew(text string) error {
	if text == "" {
		return nil // Новая подстрока может быть пустой
	}
	return s.validateSQLSafety(text)
}

func (s *StringFunctionsWindow) validateConcatSeparator(text string) error {
	if text == "" {
		return nil
	}
	return s.validateSQLSafety(text)
}

func (s *StringFunctionsWindow) validateConcatString(text string) error {
	if text == "" {
		return nil
	}
	return s.validateSQLSafety(text)
}

func (s *StringFunctionsWindow) validateLpadLength(text string) error {
	if text == "" {
		return fmt.Errorf("укажите длину")
	}
	if _, err := strconv.Atoi(text); err != nil {
		return fmt.Errorf("должно быть целым числом")
	}
	if num, _ := strconv.Atoi(text); num < 1 {
		return fmt.Errorf("длина должна быть больше 0")
	}
	return nil
}

func (s *StringFunctionsWindow) validateLpadFillChar(text string) error {
	if text == "" {
		return nil
	}
	if len(text) > 1 {
		return fmt.Errorf("укажите только один символ")
	}
	return s.validateSQLSafety(text)
}

// Проверка безопасности SQL
func (s *StringFunctionsWindow) validateSQLSafety(text string) error {
	dangerousPatterns := []string{
		";", "--", "/*", "*/", "xp_", "sp_", "exec ", "union ", "select ",
		"insert ", "update ", "delete ", "drop ", "create ", "alter ",
		"grant ", "revoke ", "\\",
	}

	lowerText := strings.ToLower(text)
	for _, dangerous := range dangerousPatterns {
		if strings.Contains(lowerText, dangerous) {
			return fmt.Errorf("текст содержит потенциально опасные символы")
		}
	}
	return nil
}

func (s *StringFunctionsWindow) loadTables() {
	tables, err := s.repository.GetTables(context.Background())
	if err != nil {
		s.showError(fmt.Errorf("не удалось загрузить список таблиц: проверьте подключение к базе данных"))
		return
	}

	if len(tables) == 0 {
		s.showError(fmt.Errorf("в базе данных не найдено ни одной таблицы"))
		return
	}

	s.tableSelect.Options = tables
	s.tableSelect.Refresh()
}

func (s *StringFunctionsWindow) onTableSelected(table string) {
	if table == "" {
		return
	}

	columns, err := s.repository.GetTableColumns(context.Background(), table)
	if err != nil {
		s.showError(fmt.Errorf("не удалось загрузить столбцы таблицы '%s': проверьте права доступа", table))
		return
	}

	// Сохраняем полную информацию о столбцах
	s.currentColumns = columns

	// Показываем ВСЕ столбцы, а не только текстовые
	var allColumnNames []string
	for _, col := range columns {
		allColumnNames = append(allColumnNames, col.Name)
	}

	if len(allColumnNames) == 0 {
		s.showError(fmt.Errorf("в таблице '%s' не найдено столбцов", table))
		return
	}

	s.columnSelect.Options = allColumnNames

	// Автоматически выбираем первый столбец, если он есть
	if len(allColumnNames) > 0 {
		s.columnSelect.SetSelected(allColumnNames[0])
	}

	s.columnSelect.Refresh()

	// Показываем информацию о количестве столбцов
	s.resultLabel.SetText(fmt.Sprintf("Загружено %d столбцов из таблицы '%s'", len(allColumnNames), table))
}

func (s *StringFunctionsWindow) onFunctionSelected(function string) {
	// Сначала скрываем все контейнеры
	s.hideAllContainers()

	switch function {
	case "UPPER":
		s.previewLabel.SetText("UPPER(column) - преобразует строку в верхний регистр\nПример: UPPER('hello') → 'HELLO'")

	case "LOWER":
		s.previewLabel.SetText("LOWER(column) - преобразует строку в нижний регистр\nПример: LOWER('HELLO') → 'hello'")

	case "LENGTH":
		s.previewLabel.SetText("LENGTH(column) - возвращает длину строки\nПример: LENGTH('hello') → 5")

	case "TRIM", "LTRIM", "RTRIM":
		s.previewLabel.SetText(fmt.Sprintf("%s(column) - удаляет пробелы%s\n%s(column, 'x') - удаляет указанный символ%s",
			function,
			getTrimDescription(function),
			function,
			getTrimDescription(function)))
		s.trimContainer.Show()

	case "SUBSTRING":
		s.previewLabel.SetText("SUBSTRING(column FROM start [FOR length]) - извлекает подстроку\nПример: SUBSTRING('hello' FROM 2 FOR 3) → 'ell'")
		s.substringContainer.Show()

	case "REPLACE":
		s.previewLabel.SetText("REPLACE(column, old, new) - заменяет подстроку\nПример: REPLACE('hello', 'l', 'x') → 'hexxo'")
		s.replaceContainer.Show()

	case "CONCAT":
		s.previewLabel.SetText("CONCAT(col1, col2, ...) - объединяет строки\nПример: CONCAT('Hello', ' ', 'World') → 'Hello World'")
		s.concatContainer.Show()
		s.concatSeparator.Hide()

	case "CONCAT_WS":
		s.previewLabel.SetText("CONCAT_WS(separator, col1, col2, ...) - объединяет с разделителем\nПример: CONCAT_WS('-', '2023', '12', '01') → '2023-12-01'")
		s.concatContainer.Show()
		s.concatSeparator.Show()

	case "CONCAT (operator ||)":
		s.previewLabel.SetText("column || string - объединяет строки через оператор ||\nПример: 'Hello' || ' ' || 'World' → 'Hello World'")
		s.concatContainer.Show()
		s.concatSeparator.Hide()

	case "LPAD":
		s.previewLabel.SetText("LPAD(column, length, fill) - дополняет строку слева\nПример: LPAD('hi', 5, 'x') → 'xxxhi'")
		s.lpadContainer.Show()

	case "RPAD":
		s.previewLabel.SetText("RPAD(column, length, fill) - дополняет строку справа\nПример: RPAD('hi', 5, 'x') → 'hixxx'")
		s.lpadContainer.Show()
	}
}

// Вспомогательная функция для описания TRIM функций
func getTrimDescription(function string) string {
	switch function {
	case "TRIM":
		return " с обоих концов"
	case "LTRIM":
		return " слева"
	case "RTRIM":
		return " справа"
	default:
		return ""
	}
}

func (s *StringFunctionsWindow) hideAllContainers() {
	s.trimContainer.Hide()
	s.substringContainer.Hide()
	s.replaceContainer.Hide()
	s.concatContainer.Hide()
	s.lpadContainer.Hide()
}

func (s *StringFunctionsWindow) validateForm() error {
	if s.tableSelect.Selected == "" {
		return fmt.Errorf("не выбрана таблица")
	}

	if s.columnSelect.Selected == "" {
		return fmt.Errorf("не выбран столбец")
	}

	if s.functionSelect.Selected == "" {
		return fmt.Errorf("не выбрана функция")
	}

	// Валидация специфичных полей в зависимости от выбранной функции
	function := s.functionSelect.Selected
	switch function {
	case "SUBSTRING":
		if err := s.substringStart.Validate(); err != nil {
			return fmt.Errorf("ошибка в начальной позиции: %v", err)
		}
	case "REPLACE":
		if err := s.replaceOld.Validate(); err != nil {
			return fmt.Errorf("ошибка в заменяемой подстроке: %v", err)
		}
		if err := s.replaceNew.Validate(); err != nil {
			return fmt.Errorf("ошибка в новой подстроке: %v", err)
		}
	case "CONCAT_WS":
		if strings.TrimSpace(s.concatSeparator.Text) == "" {
			return fmt.Errorf("для CONCAT_WS укажите разделитель")
		}
		if err := s.concatSeparator.Validate(); err != nil {
			return fmt.Errorf("ошибка в разделителе: %v", err)
		}
	case "CONCAT (operator ||)":
		if strings.TrimSpace(s.concatSecond.Text) == "" {
			return fmt.Errorf("для объединения строк укажите вторую строку")
		}
		if err := s.concatSecond.Validate(); err != nil {
			return fmt.Errorf("ошибка во второй строке: %v", err)
		}
	case "LPAD", "RPAD":
		if strings.TrimSpace(s.lpadLength.Text) == "" {
			return fmt.Errorf("для %s укажите длину", function)
		}
		if err := s.lpadLength.Validate(); err != nil {
			return fmt.Errorf("ошибка в длине: %v", err)
		}
		if err := s.lpadFillChar.Validate(); err != nil {
			return fmt.Errorf("ошибка в символе заполнения: %v", err)
		}
	}

	return nil
}

func (s *StringFunctionsWindow) buildFunctionExpression() (string, error) {
	// Валидация формы
	if err := s.validateForm(); err != nil {
		return "", err
	}

	column := s.columnSelect.Selected
	function := s.functionSelect.Selected

	// Экранирование параметров для безопасности
	escapeParam := func(text string) string {
		return strings.ReplaceAll(text, "'", "''")
	}

	switch function {
	case "UPPER":
		return fmt.Sprintf("UPPER(%s)", column), nil
	case "LOWER":
		return fmt.Sprintf("LOWER(%s)", column), nil
	case "LENGTH":
		return fmt.Sprintf("LENGTH(%s)", column), nil
	case "TRIM":
		if s.trimCharInput.Text != "" {
			return fmt.Sprintf("TRIM(BOTH '%s' FROM %s)", escapeParam(s.trimCharInput.Text), column), nil
		}
		return fmt.Sprintf("TRIM(%s)", column), nil
	case "LTRIM":
		if s.trimCharInput.Text != "" {
			return fmt.Sprintf("LTRIM(%s, '%s')", column, escapeParam(s.trimCharInput.Text)), nil
		}
		return fmt.Sprintf("LTRIM(%s)", column), nil
	case "RTRIM":
		if s.trimCharInput.Text != "" {
			return fmt.Sprintf("RTRIM(%s, '%s')", column, escapeParam(s.trimCharInput.Text)), nil
		}
		return fmt.Sprintf("RTRIM(%s)", column), nil
	case "SUBSTRING":
		if s.substringLength.Text != "" {
			return fmt.Sprintf("SUBSTRING(%s FROM %s FOR %s)", column, s.substringStart.Text, s.substringLength.Text), nil
		}
		return fmt.Sprintf("SUBSTRING(%s FROM %s)", column, s.substringStart.Text), nil
	case "REPLACE":
		return fmt.Sprintf("REPLACE(%s, '%s', '%s')", column, escapeParam(s.replaceOld.Text), escapeParam(s.replaceNew.Text)), nil
	case "CONCAT":
		expr := fmt.Sprintf("CONCAT(%s", column)
		if s.concatSecond.Text != "" {
			expr += fmt.Sprintf(", '%s'", escapeParam(s.concatSecond.Text))
		}
		if s.concatThird.Text != "" {
			expr += fmt.Sprintf(", '%s'", escapeParam(s.concatThird.Text))
		}
		expr += ")"
		return expr, nil
	case "CONCAT_WS":
		expr := fmt.Sprintf("CONCAT_WS('%s', %s", escapeParam(s.concatSeparator.Text), column)
		if s.concatSecond.Text != "" {
			expr += fmt.Sprintf(", '%s'", escapeParam(s.concatSecond.Text))
		}
		if s.concatThird.Text != "" {
			expr += fmt.Sprintf(", '%s'", escapeParam(s.concatThird.Text))
		}
		expr += ")"
		return expr, nil
	case "CONCAT (operator ||)":
		expr := fmt.Sprintf("%s || '%s'", column, escapeParam(s.concatSecond.Text))
		if s.concatThird.Text != "" {
			expr += fmt.Sprintf(" || '%s'", escapeParam(s.concatThird.Text))
		}
		return expr, nil
	case "LPAD":
		fillChar := " "
		if s.lpadFillChar.Text != "" {
			fillChar = escapeParam(s.lpadFillChar.Text)
		}
		return fmt.Sprintf("LPAD(%s, %s, '%s')", column, s.lpadLength.Text, fillChar), nil
	case "RPAD":
		fillChar := " "
		if s.lpadFillChar.Text != "" {
			fillChar = escapeParam(s.lpadFillChar.Text)
		}
		return fmt.Sprintf("RPAD(%s, %s, '%s')", column, s.lpadLength.Text, fillChar), nil
	default:
		return "", fmt.Errorf("неизвестная функция: %s", function)
	}
}

func (s *StringFunctionsWindow) applyFunction() {
	// Показываем индикатор загрузки
	s.resultLabel.SetText("Применяем функцию...")
	s.resultTable.Length = func() (int, int) { return 0, 0 }
	s.resultTable.Refresh()

	funcExpr, err := s.buildFunctionExpression()
	if err != nil {
		s.showError(err)
		s.resultLabel.SetText("Ошибка в параметрах функции")
		return
	}

	originalColumn := s.columnSelect.Selected
	query := fmt.Sprintf("SELECT %s as original, %s as result FROM %s LIMIT 50",
		originalColumn, funcExpr, s.tableSelect.Selected)

	result, err := s.repository.ExecuteQuery(context.Background(), query)
	if err != nil {
		errorMsg := s.formatDatabaseError(err)
		s.showError(fmt.Errorf("ошибка при выполнении запроса: %s", errorMsg))
		s.resultLabel.SetText("Ошибка при выполнении запроса")
		return
	}

	if result.Error != "" {
		errorMsg := s.formatDatabaseError(fmt.Errorf("result.Error"))
		s.resultLabel.SetText("Ошибка базы данных: " + errorMsg)
		return
	}

	s.displayResults(result)
	s.resultLabel.SetText(fmt.Sprintf("Функция применена к %d строкам", len(result.Rows)))
}

// Форматирование ошибок базы данных
func (s *StringFunctionsWindow) formatDatabaseError(err error) string {
	errorStr := err.Error()

	if strings.Contains(errorStr, "syntax error") {
		return "синтаксическая ошибка в запросе"
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
	if strings.Contains(errorStr, "function does not exist") {
		return "функция не поддерживается вашей версией базы данных"
	}
	if strings.Contains(errorStr, "numeric") {
		return "ошибка в числовом параметре"
	}
	if strings.Contains(errorStr, "substring") {
		return "ошибка в параметрах подстроки: позиция должна быть в пределах длины строки"
	}

	return "внутренняя ошибка базы данных"
}

func (s *StringFunctionsWindow) previewSQL() {
	funcExpr, err := s.buildFunctionExpression()
	if err != nil {
		s.showError(err)
		return
	}

	if s.tableSelect.Selected == "" {
		s.resultLabel.SetText("SQL: " + funcExpr)
	} else {
		query := fmt.Sprintf("SELECT %s as result FROM %s", funcExpr, s.tableSelect.Selected)
		s.resultLabel.SetText("SQL: " + query)
	}
}

func (s *StringFunctionsWindow) displayResults(result *models.QueryResult) {
	if len(result.Rows) == 0 {
		s.resultTable.Length = func() (int, int) { return 1, 1 }
		s.resultTable.UpdateCell = func(id widget.TableCellID, obj fyne.CanvasObject) {
			label := obj.(*widget.Label)
			label.Wrapping = fyne.TextWrapWord
			if id.Row == 0 && id.Col == 0 {
				label.SetText("Нет данных")
			}
		}
		return
	}

	s.resultTable.Length = func() (int, int) {
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
		s.resultTable.SetColumnWidth(col, maxWidth)
	}

	s.resultTable.UpdateCell = func(id widget.TableCellID, obj fyne.CanvasObject) {
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

	s.resultTable.Refresh()
}

func (s *StringFunctionsWindow) clearForm() {
	s.functionSelect.SetSelected("")
	s.hideAllContainers()

	// Очищаем все поля
	s.trimCharInput.SetText("")
	s.substringStart.SetText("")
	s.substringLength.SetText("")
	s.replaceOld.SetText("")
	s.replaceNew.SetText("")
	s.concatSeparator.SetText("")
	s.concatSecond.SetText("")
	s.concatThird.SetText("")
	s.lpadLength.SetText("")
	s.lpadFillChar.SetText("")

	s.previewLabel.SetText("Выберите функцию для просмотра примера")
	s.resultLabel.SetText("")
	s.resultTable.Length = func() (int, int) { return 0, 0 }
	s.resultTable.Refresh()
}

func (s *StringFunctionsWindow) showError(err error) {
	customDialog := dialog.NewCustom(
		"Ошибка",
		"Закрыть",
		container.NewVBox(
			widget.NewLabel("❌ Произошла ошибка:"),
			widget.NewLabel(err.Error()),
			widget.NewLabel(""),
			widget.NewLabel("Проверьте введенные параметры и попробуйте снова."),
		),
		s.window,
	)
	customDialog.Show()
}

func (s *StringFunctionsWindow) Show() {
	s.window.Show()
}
