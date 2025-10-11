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

type StringFunctionsWindow struct {
	window     fyne.Window
	repository *db.Repository
	mainWindow fyne.Window

	tableSelect    *widget.Select
	columnSelect   *widget.Select
	functionSelect *widget.Select
	param1Input    *widget.Entry
	param2Input    *widget.Entry
	previewLabel   *widget.Label
	resultTable    *widget.Table
	resultLabel    *widget.Label

	currentColumns []string
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
	s.columnSelect.PlaceHolder = "Выберите текстовый столбец"

	s.functionSelect = widget.NewSelect([]string{
		"UPPER", "LOWER", "LENGTH", "TRIM", "LTRIM", "RTRIM",
		"SUBSTRING", "REPLACE", "CONCAT", "CONCAT_WS", "LPAD", "RPAD",
	}, s.onFunctionSelected)
	s.functionSelect.PlaceHolder = "Выберите функцию"

	s.param1Input = widget.NewEntry()
	s.param1Input.SetPlaceHolder("Параметр 1")

	s.param2Input = widget.NewEntry()
	s.param2Input.SetPlaceHolder("Параметр 2")

	s.previewLabel = widget.NewLabel("Выберите функцию для просмотра примера")
	s.previewLabel.Wrapping = fyne.TextWrapWord

	s.resultLabel = widget.NewLabel("")
	s.resultLabel.Wrapping = fyne.TextWrapWord

	s.resultTable = widget.NewTable(
		func() (int, int) { return 0, 0 },
		func() fyne.CanvasObject { return widget.NewLabel("") },
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
		widget.NewLabel("Параметр 1:"),
		s.param1Input,
		widget.NewLabel("Параметр 2:"),
		s.param2Input,
		container.NewHBox(applyBtn, previewBtn, clearBtn),
		s.previewLabel,
		s.resultLabel,
	)

	content := container.NewBorder(
		form, nil, nil, nil,
		container.NewScroll(s.resultTable),
	)

	s.window.SetContent(content)
	s.window.Resize(fyne.NewSize(800, 600))
}

func (s *StringFunctionsWindow) loadTables() {
	tables, err := s.repository.GetTables(context.Background())
	if err != nil {
		s.showError(err)
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
		s.showError(err)
		return
	}

	var textColumns []string
	for _, col := range columns {
		if strings.Contains(strings.ToLower(col.DataType), "char") ||
			strings.Contains(strings.ToLower(col.DataType), "text") {
			textColumns = append(textColumns, col.Name)
		}
	}

	s.currentColumns = textColumns
	s.columnSelect.Options = textColumns
	if len(textColumns) > 0 {
		s.columnSelect.SetSelected(textColumns[0])
	}
	s.columnSelect.Refresh()
}

func (s *StringFunctionsWindow) onFunctionSelected(function string) {
	switch function {
	case "UPPER":
		s.previewLabel.SetText("UPPER(column) - преобразует строку в верхний регистр\nПример: UPPER(name) → 'JOHN'")
		s.param1Input.Hide()
		s.param2Input.Hide()
	case "LOWER":
		s.previewLabel.SetText("LOWER(column) - преобразует строку в нижний регистр\nПример: LOWER(name) → 'john'")
		s.param1Input.Hide()
		s.param2Input.Hide()
	case "LENGTH":
		s.previewLabel.SetText("LENGTH(column) - возвращает длину строки\nПример: LENGTH(name) → 4")
		s.param1Input.Hide()
		s.param2Input.Hide()
	case "TRIM":
		s.previewLabel.SetText("TRIM(column) - удаляет пробелы с обоих концов\nTRIM(BOTH 'x' FROM column) - удаляет указанный символ\nПример: TRIM('  hello  ') → 'hello'")
		s.param1Input.SetPlaceHolder("Символ для удаления (опционально)")
		s.param1Input.Show()
		s.param2Input.Hide()
	case "SUBSTRING":
		s.previewLabel.SetText("SUBSTRING(column FROM start [FOR length]) - извлекает подстроку\nПример: SUBSTRING(name FROM 2 FOR 3) → 'ohn'")
		s.param1Input.SetPlaceHolder("Начальная позиция (обязательно)")
		s.param2Input.SetPlaceHolder("Длина (опционально)")
		s.param1Input.Show()
		s.param2Input.Show()
	case "REPLACE":
		s.previewLabel.SetText("REPLACE(column, old, new) - заменяет подстроку\nПример: REPLACE(name, 'o', '0') → 'j0hn'")
		s.param1Input.SetPlaceHolder("Старая подстрока")
		s.param2Input.SetPlaceHolder("Новая подстрока")
		s.param1Input.Show()
		s.param2Input.Show()
	case "CONCAT":
		s.previewLabel.SetText("CONCAT(col1, col2, ...) - объединяет строки\nCONCAT_WS(separator, col1, col2, ...) - объединяет с разделителем")
		s.param1Input.SetPlaceHolder("Вторая строка или разделитель для CONCAT_WS")
		s.param2Input.SetPlaceHolder("Третья строка")
		s.param1Input.Show()
		s.param2Input.Show()
	case "LPAD", "RPAD":
		s.previewLabel.SetText("LPAD(column, length, fill) - дополняет строку слева\nRPAD(column, length, fill) - дополняет строку справа\nПример: LPAD('hi', 5, 'x') → 'xxxhi'")
		s.param1Input.SetPlaceHolder("Длина")
		s.param2Input.SetPlaceHolder("Заполняющий символ")
		s.param1Input.Show()
		s.param2Input.Show()
	}
}

func (s *StringFunctionsWindow) buildFunctionExpression() (string, error) {
	if s.columnSelect.Selected == "" {
		return "", fmt.Errorf("не выбран столбец")
	}
	if s.functionSelect.Selected == "" {
		return "", fmt.Errorf("не выбрана функция")
	}

	column := s.columnSelect.Selected
	function := s.functionSelect.Selected

	switch function {
	case "UPPER":
		return fmt.Sprintf("UPPER(%s)", column), nil
	case "LOWER":
		return fmt.Sprintf("LOWER(%s)", column), nil
	case "LENGTH":
		return fmt.Sprintf("LENGTH(%s)", column), nil
	case "TRIM":
		if s.param1Input.Text != "" {
			return fmt.Sprintf("TRIM(BOTH '%s' FROM %s)", s.param1Input.Text, column), nil
		}
		return fmt.Sprintf("TRIM(%s)", column), nil
	case "LTRIM":
		if s.param1Input.Text != "" {
			return fmt.Sprintf("LTRIM(%s, '%s')", column, s.param1Input.Text), nil
		}
		return fmt.Sprintf("LTRIM(%s)", column), nil
	case "RTRIM":
		if s.param1Input.Text != "" {
			return fmt.Sprintf("RTRIM(%s, '%s')", column, s.param1Input.Text), nil
		}
		return fmt.Sprintf("RTRIM(%s)", column), nil
	case "SUBSTRING":
		if s.param1Input.Text == "" {
			return "", fmt.Errorf("укажите начальную позицию")
		}
		if s.param2Input.Text != "" {
			return fmt.Sprintf("SUBSTRING(%s FROM %s FOR %s)", column, s.param1Input.Text, s.param2Input.Text), nil
		}
		return fmt.Sprintf("SUBSTRING(%s FROM %s)", column, s.param1Input.Text), nil
	case "REPLACE":
		if s.param1Input.Text == "" || s.param2Input.Text == "" {
			return "", fmt.Errorf("укажите старую и новую подстроку")
		}
		return fmt.Sprintf("REPLACE(%s, '%s', '%s')", column, s.param1Input.Text, s.param2Input.Text), nil
	case "CONCAT":
		expr := fmt.Sprintf("CONCAT(%s", column)
		if s.param1Input.Text != "" {
			expr += ", " + s.param1Input.Text
		}
		if s.param2Input.Text != "" {
			expr += ", " + s.param2Input.Text
		}
		expr += ")"
		return expr, nil
	case "CONCAT_WS":
		if s.param1Input.Text == "" {
			return "", fmt.Errorf("укажите разделитель")
		}
		expr := fmt.Sprintf("CONCAT_WS('%s', %s", s.param1Input.Text, column)
		if s.param2Input.Text != "" {
			expr += ", " + s.param2Input.Text
		}
		expr += ")"
		return expr, nil
	case "LPAD":
		if s.param1Input.Text == "" {
			return "", fmt.Errorf("укажите длину")
		}
		fillChar := " "
		if s.param2Input.Text != "" {
			fillChar = s.param2Input.Text
		}
		return fmt.Sprintf("LPAD(%s, %s, '%s')", column, s.param1Input.Text, fillChar), nil
	case "RPAD":
		if s.param1Input.Text == "" {
			return "", fmt.Errorf("укажите длину")
		}
		fillChar := " "
		if s.param2Input.Text != "" {
			fillChar = s.param2Input.Text
		}
		return fmt.Sprintf("RPAD(%s, %s, '%s')", column, s.param1Input.Text, fillChar), nil
	default:
		return "", fmt.Errorf("неизвестная функция")
	}
}

func (s *StringFunctionsWindow) applyFunction() {
	if s.tableSelect.Selected == "" {
		s.showError(fmt.Errorf("не выбрана таблица"))
		return
	}

	funcExpr, err := s.buildFunctionExpression()
	if err != nil {
		s.showError(err)
		return
	}

	originalColumn := s.columnSelect.Selected
	query := fmt.Sprintf("SELECT %s as original, %s as result FROM %s LIMIT 50",
		originalColumn, funcExpr, s.tableSelect.Selected)

	result, err := s.repository.ExecuteQuery(context.Background(), query)
	if err != nil {
		s.showError(err)
		return
	}

	if result.Error != "" {
		s.resultLabel.SetText("Ошибка: " + result.Error)
		return
	}

	s.displayResults(result)
	s.resultLabel.SetText(fmt.Sprintf("Функция применена к %d строкам", len(result.Rows)))
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
			if id.Row == 0 && id.Col == 0 {
				label.SetText("Нет данных")
			}
		}
		return
	}

	s.resultTable.Length = func() (int, int) {
		return len(result.Rows) + 1, len(result.Columns)
	}

	s.resultTable.UpdateCell = func(id widget.TableCellID, obj fyne.CanvasObject) {
		label := obj.(*widget.Label)

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
	s.param1Input.SetText("")
	s.param2Input.SetText("")
	s.previewLabel.SetText("")
	s.resultLabel.SetText("")
	s.resultTable.Length = func() (int, int) { return 0, 0 }
	s.resultTable.Refresh()
}

func (s *StringFunctionsWindow) showError(err error) {
	dialog.ShowError(err, s.window)
}

func (s *StringFunctionsWindow) Show() {
	s.window.Show()
}
