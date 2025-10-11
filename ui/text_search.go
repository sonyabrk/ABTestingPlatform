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
	t.patternInput.OnChanged = func(s string) {
		// Автоматическое добавление % для LIKE если не POSIX
		if t.searchType.Selected == "LIKE" && !strings.Contains(s, "%") && s != "" {
			t.patternInput.SetText("%" + s + "%")
			t.patternInput.CursorColumn = len(s) + 1
		}
	}

	t.resultLabel = widget.NewLabel("Введите условия поиска")
	t.resultLabel.Wrapping = fyne.TextWrapWord

	t.resultTable = widget.NewTable(
		func() (int, int) { return 0, 0 },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(i widget.TableCellID, o fyne.CanvasObject) {},
	)

	searchBtn := widget.NewButton("Найти", t.executeSearch)
	clearBtn := widget.NewButton("Очистить", t.clearForm)

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
		container.NewHBox(searchBtn, clearBtn),
		t.resultLabel,
	)

	content := container.NewBorder(
		form, nil, nil, nil,
		container.NewScroll(t.resultTable),
	)

	t.window.SetContent(content)
	t.window.Resize(fyne.NewSize(800, 600))
}

func (t *TextSearchWindow) loadTables() {
	tables, err := t.repository.GetTables(context.Background())
	if err != nil {
		t.showError(err)
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
		t.showError(err)
		return
	}

	var textColumns []string
	for _, col := range columns {
		// Показываем только текстовые столбцы
		if strings.Contains(strings.ToLower(col.DataType), "char") ||
			strings.Contains(strings.ToLower(col.DataType), "text") {
			textColumns = append(textColumns, col.Name)
		}
	}

	t.currentColumns = textColumns
	t.columnSelect.Options = textColumns
	if len(textColumns) > 0 {
		t.columnSelect.SetSelected(textColumns[0])
	}
	t.columnSelect.Refresh()
}

func (t *TextSearchWindow) buildSearchQuery() (string, error) {
	if t.tableSelect.Selected == "" {
		return "", fmt.Errorf("не выбрана таблица")
	}
	if t.columnSelect.Selected == "" {
		return "", fmt.Errorf("не выбран столбец")
	}
	if t.patternInput.Text == "" {
		return "", fmt.Errorf("не указан шаблон поиска")
	}

	table := t.tableSelect.Selected
	column := t.columnSelect.Selected
	pattern := t.patternInput.Text

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
	query, err := t.buildSearchQuery()
	if err != nil {
		t.showError(err)
		return
	}

	result, err := t.repository.ExecuteQuery(context.Background(), query)
	if err != nil {
		t.showError(err)
		return
	}

	if result.Error != "" {
		t.resultLabel.SetText("Ошибка: " + result.Error)
		return
	}

	t.displayResults(result)
}

func (t *TextSearchWindow) displayResults(result *models.QueryResult) {
	if len(result.Rows) == 0 {
		t.resultTable.Length = func() (int, int) { return 1, 1 }
		t.resultTable.UpdateCell = func(id widget.TableCellID, obj fyne.CanvasObject) {
			label := obj.(*widget.Label)
			if id.Row == 0 && id.Col == 0 {
				label.SetText("Ничего не найдено")
			}
		}
		t.resultLabel.SetText("По вашему запросу ничего не найдено")
		return
	}

	t.resultTable.Length = func() (int, int) {
		return len(result.Rows) + 1, len(result.Columns)
	}

	t.resultTable.UpdateCell = func(id widget.TableCellID, obj fyne.CanvasObject) {
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
					// Подсветка найденного текста
					text := fmt.Sprintf("%v", value)
					if t.columnSelect.Selected == result.Columns[id.Col] {
						// Можно добавить подсветку, но в Fyne это сложнее
						label.SetText(text)
					} else {
						label.SetText(text)
					}
				} else {
					label.SetText("NULL")
				}
			}
		}
	}

	t.resultLabel.SetText(fmt.Sprintf("Найдено %d строк", len(result.Rows)))
	t.resultTable.Refresh()
}

func (t *TextSearchWindow) clearForm() {
	t.patternInput.SetText("")
	t.resultLabel.SetText("")
	t.resultTable.Length = func() (int, int) { return 0, 0 }
	t.resultTable.Refresh()
}

func (t *TextSearchWindow) showError(err error) {
	dialog.ShowError(err, t.window)
}

func (t *TextSearchWindow) Show() {
	t.window.Show()
}
