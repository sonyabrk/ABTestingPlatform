// package ui

// import (
// 	"context"
// 	"fmt"
// 	"strings"
// 	"testing-platform/db"
// 	"testing-platform/db/models"

// 	"fyne.io/fyne/v2"
// 	"fyne.io/fyne/v2/container"
// 	"fyne.io/fyne/v2/dialog"
// 	"fyne.io/fyne/v2/widget"
// )

// type TextSearchWindow struct {
// 	window     fyne.Window
// 	repository *db.Repository
// 	mainWindow fyne.Window

// 	tableSelect  *widget.Select
// 	columnSelect *widget.Select
// 	searchType   *widget.Select
// 	patternInput *widget.Entry
// 	resultTable  *widget.Table
// 	resultLabel  *widget.Label

// 	currentColumns []string
// }

// func NewTextSearchWindow(repo *db.Repository, mainWindow fyne.Window) *TextSearchWindow {
// 	t := &TextSearchWindow{
// 		repository: repo,
// 		mainWindow: mainWindow,
// 		window:     fyne.CurrentApp().NewWindow("Текстовый поиск"),
// 	}

// 	t.buildUI()
// 	t.loadTables()
// 	return t
// }

// func (t *TextSearchWindow) buildUI() {
// 	t.tableSelect = widget.NewSelect([]string{}, t.onTableSelected)
// 	t.tableSelect.PlaceHolder = "Выберите таблицу"

// 	t.columnSelect = widget.NewSelect([]string{}, nil)
// 	t.columnSelect.PlaceHolder = "Выберите столбец"

// 	t.searchType = widget.NewSelect([]string{
// 		"LIKE", "NOT LIKE", "POSIX (~)", "POSIX Case Insensitive (~*)",
// 		"NOT POSIX (!~)", "NOT POSIX Case Insensitive (!~*)",
// 	}, nil)
// 	t.searchType.SetSelected("LIKE")
// 	t.searchType.PlaceHolder = "Тип поиска"

// 	t.patternInput = widget.NewEntry()
// 	t.patternInput.SetPlaceHolder("Введите шаблон поиска")
// 	t.patternInput.OnChanged = func(s string) {
// 		// Автоматическое добавление % для LIKE если не POSIX
// 		if t.searchType.Selected == "LIKE" && !strings.Contains(s, "%") && s != "" {
// 			t.patternInput.SetText("%" + s + "%")
// 			t.patternInput.CursorColumn = len(s) + 1
// 		}
// 	}

// 	t.resultLabel = widget.NewLabel("Введите условия поиска")
// 	t.resultLabel.Wrapping = fyne.TextWrapWord

// 	t.resultTable = widget.NewTable(
// 		func() (int, int) { return 0, 0 },
// 		func() fyne.CanvasObject { return widget.NewLabel("") },
// 		func(i widget.TableCellID, o fyne.CanvasObject) {},
// 	)

// 	searchBtn := widget.NewButton("Найти", t.executeSearch)
// 	clearBtn := widget.NewButton("Очистить", t.clearForm)

// 	// Компоновка
// 	form := container.NewVBox(
// 		widget.NewLabel("Таблица:"),
// 		t.tableSelect,
// 		widget.NewLabel("Столбец:"),
// 		t.columnSelect,
// 		widget.NewLabel("Тип поиска:"),
// 		t.searchType,
// 		widget.NewLabel("Шаблон:"),
// 		t.patternInput,
// 		container.NewHBox(searchBtn, clearBtn),
// 		t.resultLabel,
// 	)

// 	content := container.NewBorder(
// 		form, nil, nil, nil,
// 		container.NewScroll(t.resultTable),
// 	)

// 	t.window.SetContent(content)
// 	t.window.Resize(fyne.NewSize(800, 600))
// }

// func (t *TextSearchWindow) loadTables() {
// 	tables, err := t.repository.GetTables(context.Background())
// 	if err != nil {
// 		t.showError(err)
// 		return
// 	}
// 	t.tableSelect.Options = tables
// 	t.tableSelect.Refresh()
// }

// func (t *TextSearchWindow) onTableSelected(table string) {
// 	if table == "" {
// 		return
// 	}

// 	columns, err := t.repository.GetTableColumns(context.Background(), table)
// 	if err != nil {
// 		t.showError(err)
// 		return
// 	}

// 	var textColumns []string
// 	for _, col := range columns {
// 		// Показываем только текстовые столбцы
// 		if strings.Contains(strings.ToLower(col.DataType), "char") ||
// 			strings.Contains(strings.ToLower(col.DataType), "text") {
// 			textColumns = append(textColumns, col.Name)
// 		}
// 	}

// 	t.currentColumns = textColumns
// 	t.columnSelect.Options = textColumns
// 	if len(textColumns) > 0 {
// 		t.columnSelect.SetSelected(textColumns[0])
// 	}
// 	t.columnSelect.Refresh()
// }

// func (t *TextSearchWindow) buildSearchQuery() (string, error) {
// 	if t.tableSelect.Selected == "" {
// 		return "", fmt.Errorf("не выбрана таблица")
// 	}
// 	if t.columnSelect.Selected == "" {
// 		return "", fmt.Errorf("не выбран столбец")
// 	}
// 	if t.patternInput.Text == "" {
// 		return "", fmt.Errorf("не указан шаблон поиска")
// 	}

// 	table := t.tableSelect.Selected
// 	column := t.columnSelect.Selected
// 	pattern := t.patternInput.Text

// 	var condition string
// 	switch t.searchType.Selected {
// 	case "LIKE":
// 		condition = fmt.Sprintf("%s LIKE '%s'", column, pattern)
// 	case "NOT LIKE":
// 		condition = fmt.Sprintf("%s NOT LIKE '%s'", column, pattern)
// 	case "POSIX (~)":
// 		condition = fmt.Sprintf("%s ~ '%s'", column, pattern)
// 	case "POSIX Case Insensitive (~*)":
// 		condition = fmt.Sprintf("%s ~* '%s'", column, pattern)
// 	case "NOT POSIX (!~)":
// 		condition = fmt.Sprintf("%s !~ '%s'", column, pattern)
// 	case "NOT POSIX Case Insensitive (!~*)":
// 		condition = fmt.Sprintf("%s !~* '%s'", column, pattern)
// 	default:
// 		return "", fmt.Errorf("неизвестный тип поиска")
// 	}

// 	query := fmt.Sprintf("SELECT * FROM %s WHERE %s LIMIT 100", table, condition)
// 	return query, nil
// }

// func (t *TextSearchWindow) executeSearch() {
// 	query, err := t.buildSearchQuery()
// 	if err != nil {
// 		t.showError(err)
// 		return
// 	}

// 	result, err := t.repository.ExecuteQuery(context.Background(), query)
// 	if err != nil {
// 		t.showError(err)
// 		return
// 	}

// 	if result.Error != "" {
// 		t.resultLabel.SetText("Ошибка: " + result.Error)
// 		return
// 	}

// 	t.displayResults(result)
// }

// func (t *TextSearchWindow) displayResults(result *models.QueryResult) {
// 	if len(result.Rows) == 0 {
// 		t.resultTable.Length = func() (int, int) { return 1, 1 }
// 		t.resultTable.UpdateCell = func(id widget.TableCellID, obj fyne.CanvasObject) {
// 			label := obj.(*widget.Label)
// 			if id.Row == 0 && id.Col == 0 {
// 				label.SetText("Ничего не найдено")
// 			}
// 		}
// 		t.resultLabel.SetText("По вашему запросу ничего не найдено")
// 		return
// 	}

// 	t.resultTable.Length = func() (int, int) {
// 		return len(result.Rows) + 1, len(result.Columns)
// 	}

// 	t.resultTable.UpdateCell = func(id widget.TableCellID, obj fyne.CanvasObject) {
// 		label := obj.(*widget.Label)

// 		if id.Row == 0 {
// 			if id.Col < len(result.Columns) {
// 				label.SetText(result.Columns[id.Col])
// 				label.TextStyle = fyne.TextStyle{Bold: true}
// 			}
// 		} else {
// 			rowIndex := id.Row - 1
// 			if rowIndex < len(result.Rows) && id.Col < len(result.Columns) {
// 				value := result.Rows[rowIndex][result.Columns[id.Col]]
// 				if value != nil {
// 					// Подсветка найденного текста
// 					text := fmt.Sprintf("%v", value)
// 					if t.columnSelect.Selected == result.Columns[id.Col] {
// 						// Можно добавить подсветку, но в Fyne это сложнее
// 						label.SetText(text)
// 					} else {
// 						label.SetText(text)
// 					}
// 				} else {
// 					label.SetText("NULL")
// 				}
// 			}
// 		}
// 	}

// 	t.resultLabel.SetText(fmt.Sprintf("Найдено %d строк", len(result.Rows)))
// 	t.resultTable.Refresh()
// }

// func (t *TextSearchWindow) clearForm() {
// 	t.patternInput.SetText("")
// 	t.resultLabel.SetText("")
// 	t.resultTable.Length = func() (int, int) { return 0, 0 }
// 	t.resultTable.Refresh()
// }

// func (t *TextSearchWindow) showError(err error) {
// 	dialog.ShowError(err, t.window)
// }

//	func (t *TextSearchWindow) Show() {
//		t.window.Show()
//	}
package ui

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing-platform/db"
	"testing-platform/db/models"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

type TextSearchWindow struct {
	window     fyne.Window
	repository *db.Repository
	mainWindow fyne.Window

	tableSelect  *widget.Select
	columnSelect *widget.Select
	searchType   *widget.Select
	patternInput *widget.Entry
	resultTable  *widget.Table
	resultLabel  *widget.Label

	currentColumns []string
}

func NewTextSearchWindow(repo *db.Repository, mainWindow fyne.Window) *TextSearchWindow {
	t := &TextSearchWindow{
		repository: repo,
		mainWindow: mainWindow,
		window:     fyne.CurrentApp().NewWindow("Текстовый поиск"),
	}

	t.buildUI()
	t.loadTables()
	return t
}

func (t *TextSearchWindow) buildUI() {
	t.tableSelect = widget.NewSelect([]string{}, t.onTableSelected)
	t.tableSelect.PlaceHolder = "Выберите таблицу"

	t.columnSelect = widget.NewSelect([]string{}, nil)
	t.columnSelect.PlaceHolder = "Выберите столбец"

	t.searchType = widget.NewSelect([]string{
		"LIKE", "NOT LIKE", "POSIX (~)", "POSIX Case Insensitive (~*)",
		"NOT POSIX (!~)", "NOT POSIX Case Insensitive (!~*)",
	}, nil)
	t.searchType.SetSelected("LIKE")
	t.searchType.PlaceHolder = "Тип поиска"

	t.patternInput = widget.NewEntry()
	t.patternInput.SetPlaceHolder("Введите шаблон поиска")

	// Добавляем валидацию в реальном времени
	t.patternInput.Validator = func(text string) error {
		if text == "" {
			return nil // Пустой ввод разрешен
		}
		return t.validatePattern(text)
	}

	t.resultLabel = widget.NewLabel("Введите условия поиска")
	t.resultLabel.Wrapping = fyne.TextWrapWord

	// Создаем таблицу с поддержкой переноса текста
	t.resultTable = widget.NewTable(
		func() (int, int) { return 0, 0 },
		func() fyne.CanvasObject {
			label := widget.NewLabel("")
			label.Wrapping = fyne.TextWrapWord // Включаем перенос текста
			return label
		},
		func(i widget.TableCellID, o fyne.CanvasObject) {},
	)

	searchBtn := widget.NewButton("Найти", t.executeSearch)
	clearBtn := widget.NewButton("Очистить", t.clearForm)

	// Добавляем подсказки
	hintLabel := widget.NewLabel("💡 Подсказки:\n• Для LIKE используйте % для поиска частей текста\n• Для POSIX используйте стандартные регулярные выражения\n• Избегайте специальных символов без экранирования")
	hintLabel.Wrapping = fyne.TextWrapWord

	// Компоновка
	form := container.NewVBox(
		widget.NewLabel("Таблица:"),
		t.tableSelect,
		widget.NewLabel("Столбец:"),
		t.columnSelect,
		widget.NewLabel("Тип поиска:"),
		t.searchType,
		widget.NewLabel("Шаблон:"),
		t.patternInput,
		hintLabel,
		container.NewHBox(searchBtn, clearBtn),
		t.resultLabel,
	)

	content := container.NewBorder(
		form, nil, nil, nil,
		container.NewScroll(t.resultTable),
	)

	t.window.SetContent(content)
	t.window.Resize(fyne.NewSize(1000, 700))
}

// Валидация шаблона поиска
func (t *TextSearchWindow) validatePattern(pattern string) error {
	// Проверка на слишком длинный шаблон
	if len(pattern) > 500 {
		return fmt.Errorf("слишком длинный шаблон поиска (максимум 500 символов)")
	}

	// Проверка на опасные SQL-инъекции (базовая защита)
	dangerousPatterns := []string{
		";", "--", "/*", "*/", "xp_", "sp_", "exec ", "union ", "select ", "insert ", "update ", "delete ", "drop ", "create ",
	}

	lowerPattern := strings.ToLower(pattern)
	for _, dangerous := range dangerousPatterns {
		if strings.Contains(lowerPattern, dangerous) {
			return fmt.Errorf("шаблон содержит потенциально опасные символы: %s", dangerous)
		}
	}

	// Проверка специальных символов для разных типов поиска
	currentSearchType := t.searchType.Selected

	switch currentSearchType {
	case "LIKE", "NOT LIKE":
		// Для LIKE проверяем корректное использование % и _
		if strings.Count(pattern, "%") > 10 {
			return fmt.Errorf("слишком много символов %% в шаблоне (максимум 10)")
		}
		if strings.Count(pattern, "_") > 20 {
			return fmt.Errorf("слишком много символов _ в шаблоне (максимум 20)")
		}

	case "POSIX (~)", "POSIX Case Insensitive (~*)", "NOT POSIX (!~)", "NOT POSIX Case Insensitive (!~*)":
		// Базовая проверка регулярных выражений
		if err := t.validateRegex(pattern); err != nil {
			return fmt.Errorf("некорректное регулярное выражение: %v", err)
		}
	}

	return nil
}

// Валидация регулярных выражений
func (t *TextSearchWindow) validateRegex(pattern string) error {
	// Проверка на слишком сложные/опасные регулярные выражения
	if len(pattern) > 200 {
		return fmt.Errorf("регулярное выражение слишком сложное (максимум 200 символов)")
	}

	// Проверка на экранирование специальных символов
	if strings.Contains(pattern, `\\`) && !strings.Contains(pattern, `\\`) {
		return fmt.Errorf("некорректное экранирование символов - используйте \\\\ для обратного слеша")
	}

	// Проверка сбалансированности скобок
	if strings.Count(pattern, "(") != strings.Count(pattern, ")") {
		return fmt.Errorf("несбалансированные круглые скобки в регулярном выражении")
	}
	if strings.Count(pattern, "[") != strings.Count(pattern, "]") {
		return fmt.Errorf("несбалансированные квадратные скобки в регулярном выражении")
	}
	if strings.Count(pattern, "{") != strings.Count(pattern, "}") {
		return fmt.Errorf("несбалансированные фигурные скобки в регулярном выражении")
	}

	// Попытка компиляции регулярного выражения (базовая проверка)
	_, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("синтаксическая ошибка в регулярном выражении")
	}

	return nil
}

func (t *TextSearchWindow) loadTables() {
	tables, err := t.repository.GetTables(context.Background())
	if err != nil {
		t.showError(fmt.Errorf("не удалось загрузить список таблиц: проверьте подключение к базе данных"))
		return
	}

	if len(tables) == 0 {
		t.showError(fmt.Errorf("в базе данных не найдено ни одной таблицы"))
		return
	}

	t.tableSelect.Options = tables
	t.tableSelect.Refresh()
}

func (t *TextSearchWindow) onTableSelected(table string) {
	if table == "" {
		return
	}

	columns, err := t.repository.GetTableColumns(context.Background(), table)
	if err != nil {
		t.showError(fmt.Errorf("не удалось загрузить столбцы таблицы '%s': проверьте права доступа", table))
		return
	}

	// Показываем ВСЕ столбцы, а не только текстовые
	var allColumns []string
	for _, col := range columns {
		allColumns = append(allColumns, col.Name)
	}

	if len(allColumns) == 0 {
		t.showError(fmt.Errorf("в таблице '%s' не найдено столбцов", table))
		return
	}

	t.currentColumns = allColumns
	t.columnSelect.Options = allColumns
	t.columnSelect.SetSelected(allColumns[0])
	t.columnSelect.Refresh()
}

func (t *TextSearchWindow) validateSearchParams() error {
	if t.tableSelect.Selected == "" {
		return fmt.Errorf("не выбрана таблица для поиска")
	}

	if t.columnSelect.Selected == "" {
		return fmt.Errorf("не выбран столбец для поиска")
	}

	if strings.TrimSpace(t.patternInput.Text) == "" {
		return fmt.Errorf("введите текст для поиска")
	}

	// Дополнительная валидация шаблона
	if err := t.validatePattern(t.patternInput.Text); err != nil {
		return err
	}

	return nil
}

func (t *TextSearchWindow) buildSearchQuery() (string, error) {
	// Предварительная валидация
	if err := t.validateSearchParams(); err != nil {
		return "", err
	}

	table := t.tableSelect.Selected
	column := t.columnSelect.Selected
	pattern := strings.TrimSpace(t.patternInput.Text)

	// Экранирование специальных символов для безопасности
	pattern = strings.ReplaceAll(pattern, "'", "''")
	pattern = strings.ReplaceAll(pattern, `\`, `\\`)

	// Для LIKE автоматически добавляем % если их нет
	if t.searchType.Selected == "LIKE" && !strings.Contains(pattern, "%") {
		pattern = "%" + pattern + "%"
	}

	var condition string
	switch t.searchType.Selected {
	case "LIKE":
		condition = fmt.Sprintf("%s LIKE '%s'", column, pattern)
	case "NOT LIKE":
		condition = fmt.Sprintf("%s NOT LIKE '%s'", column, pattern)
	case "POSIX (~)":
		condition = fmt.Sprintf("%s ~ '%s'", column, pattern)
	case "POSIX Case Insensitive (~*)":
		condition = fmt.Sprintf("%s ~* '%s'", column, pattern)
	case "NOT POSIX (!~)":
		condition = fmt.Sprintf("%s !~ '%s'", column, pattern)
	case "NOT POSIX Case Insensitive (!~*)":
		condition = fmt.Sprintf("%s !~* '%s'", column, pattern)
	default:
		return "", fmt.Errorf("неизвестный тип поиска")
	}

	query := fmt.Sprintf("SELECT * FROM %s WHERE %s LIMIT 100", table, condition)
	return query, nil
}

func (t *TextSearchWindow) executeSearch() {
	// Показываем индикатор загрузки
	t.resultLabel.SetText("Выполняется поиск...")
	t.resultTable.Length = func() (int, int) { return 0, 0 }
	t.resultTable.Refresh()

	query, err := t.buildSearchQuery()
	if err != nil {
		t.showError(err)
		t.resultLabel.SetText("Ошибка в параметрах поиска")
		return
	}

	result, err := t.repository.ExecuteQuery(context.Background(), query)
	if err != nil {
		// Обработка специфических ошибок базы данных
		errorMsg := t.formatDatabaseError(err)
		t.showError(fmt.Errorf("ошибка при выполнении запроса: %s", errorMsg))
		t.resultLabel.SetText("Ошибка при выполнении поиска")
		return
	}

	if result.Error != "" {
		errorMsg := t.formatDatabaseError(fmt.Errorf("result.Error"))
		t.resultLabel.SetText("Ошибка базы данных: " + errorMsg)
		return
	}

	t.displayResults(result)
}

// Форматирование ошибок базы данных в понятный вид
func (t *TextSearchWindow) formatDatabaseError(err error) string {
	errorStr := err.Error()

	// PostgreSQL ошибки
	if strings.Contains(errorStr, "syntax error") {
		return "синтаксическая ошибка в запросе"
	}
	if strings.Contains(errorStr, "invalid regular expression") {
		return "некорректное регулярное выражение"
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

	// Общие ошибки
	if strings.Contains(errorStr, "LIKE") && strings.Contains(errorStr, "pattern") {
		return "некорректный шаблон для поиска LIKE"
	}

	return "внутренняя ошибка базы данных"
}

func (t *TextSearchWindow) displayResults(result *models.QueryResult) {
	if len(result.Rows) == 0 {
		t.resultTable.Length = func() (int, int) { return 1, 1 }
		t.resultTable.UpdateCell = func(id widget.TableCellID, obj fyne.CanvasObject) {
			label := obj.(*widget.Label)
			label.Wrapping = fyne.TextWrapWord
			if id.Row == 0 && id.Col == 0 {
				label.SetText("Ничего не найдено")
			}
		}
		t.resultLabel.SetText("По вашему запросу ничего не найдено. Попробуйте изменить условия поиска.")
		return
	}

	t.resultTable.Length = func() (int, int) {
		return len(result.Rows) + 1, len(result.Columns)
	}

	// Автоматически настраиваем ширину колонок на основе содержимого
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
				// Оцениваем ширину текста (примерно 7 пикселей на символ)
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
		t.resultTable.SetColumnWidth(col, maxWidth)
	}

	t.resultTable.UpdateCell = func(id widget.TableCellID, obj fyne.CanvasObject) {
		label := obj.(*widget.Label)
		label.Wrapping = fyne.TextWrapWord // Убедимся, что перенос включен

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
					text := fmt.Sprintf("%v", value)
					label.SetText(text)
				} else {
					label.SetText("NULL")
				}
			}
		}
	}

	t.resultLabel.SetText(fmt.Sprintf("Найдено %d строк. Для уточнения поиска измените шаблон или тип поиска.", len(result.Rows)))
	t.resultTable.Refresh()
}

func (t *TextSearchWindow) clearForm() {
	t.patternInput.SetText("")
	t.patternInput.Validate() // Сбрасываем состояние валидации
	t.resultLabel.SetText("Введите условия поиска")
	t.resultTable.Length = func() (int, int) { return 0, 0 }
	t.resultTable.Refresh()
}

func (t *TextSearchWindow) showError(err error) {
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
		t.window,
	)
	customDialog.Show()
}

func (t *TextSearchWindow) Show() {
	t.window.Show()
}
