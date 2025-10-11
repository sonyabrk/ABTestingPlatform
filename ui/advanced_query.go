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

type AdvancedQueryWindow struct {
	window     fyne.Window
	repository *db.Repository
	mainWindow fyne.Window

	// Элементы SELECT
	tableSelect  *widget.Select
	columnList   *widget.CheckGroup
	whereClause  *widget.Entry
	orderBy      *widget.Entry
	groupBy      *widget.Entry
	havingClause *widget.Entry
	limitInput   *widget.Entry

	// Результаты
	resultTable *widget.Table
	resultLabel *widget.Label
	sqlPreview  *widget.Entry

	currentColumns []models.ColumnInfo
}

func NewAdvancedQueryWindow(repo *db.Repository, mainWindow fyne.Window) *AdvancedQueryWindow {
	a := &AdvancedQueryWindow{
		repository: repo,
		mainWindow: mainWindow,
		window:     fyne.CurrentApp().NewWindow("Расширенный SELECT"),
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

	a.whereClause = widget.NewEntry()
	a.whereClause.SetPlaceHolder("WHERE условие (age > 18 AND name = 'John')")

	a.orderBy = widget.NewEntry()
	a.orderBy.SetPlaceHolder("ORDER BY (name ASC, age DESC)")

	a.groupBy = widget.NewEntry()
	a.groupBy.SetPlaceHolder("GROUP BY (department, year)")

	a.havingClause = widget.NewEntry()
	a.havingClause.SetPlaceHolder("HAVING (COUNT(*) > 5)")

	a.limitInput = widget.NewEntry()
	a.limitInput.SetPlaceHolder("LIMIT (100)")
	a.limitInput.SetText("100")

	a.sqlPreview = widget.NewMultiLineEntry()
	a.sqlPreview.Wrapping = fyne.TextWrapOff

	a.resultLabel = widget.NewLabel("Результаты появятся здесь")
	a.resultLabel.Wrapping = fyne.TextWrapWord

	// Таблица для результатов
	a.resultTable = widget.NewTable(
		func() (int, int) { return 0, 0 },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(i widget.TableCellID, o fyne.CanvasObject) {},
	)
	a.resultTable.SetColumnWidth(0, 150)

	// Кнопки
	executeBtn := widget.NewButton("Выполнить запрос", a.executeQuery)
	clearBtn := widget.NewButton("Очистить", a.clearForm)
	showSQLBtn := widget.NewButton("Показать SQL", a.previewSQL)

	// Компоновка
	leftPanel := container.NewVBox(
		widget.NewLabel("Таблица:"),
		a.tableSelect,
		widget.NewLabel("Столбцы:"),
		container.NewScroll(a.columnList),
	)

	rightPanel := container.NewVBox(
		widget.NewLabel("WHERE:"),
		a.whereClause,
		widget.NewLabel("ORDER BY:"),
		a.orderBy,
		widget.NewLabel("GROUP BY:"),
		a.groupBy,
		widget.NewLabel("HAVING:"),
		a.havingClause,
		widget.NewLabel("LIMIT:"),
		a.limitInput,
		container.NewHBox(executeBtn, showSQLBtn, clearBtn),
	)

	controls := container.NewHBox(leftPanel, rightPanel)

	content := container.NewBorder(
		controls,
		container.NewVBox(a.resultLabel, widget.NewLabel("SQL:"), a.sqlPreview),
		nil, nil,
		container.NewScroll(a.resultTable),
	)

	a.window.SetContent(content)
	a.window.Resize(fyne.NewSize(900, 700))
}

func (a *AdvancedQueryWindow) loadTables() {
	tables, err := a.repository.GetTables(context.Background())
	if err != nil {
		a.showError(err)
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
		a.showError(err)
		return
	}

	a.currentColumns = columns
	var columnNames []string
	for _, col := range columns {
		columnNames = append(columnNames, col.Name)
	}

	a.columnList.Options = columnNames
	a.columnList.Selected = columnNames // Выбираем все по умолчанию
	a.columnList.Refresh()
}

func (a *AdvancedQueryWindow) buildQuery() string {
	table := a.tableSelect.Selected
	if table == "" {
		return ""
	}

	// SELECT часть
	var selectedColumns string
	if len(a.columnList.Selected) == 0 {
		selectedColumns = "*"
	} else {
		selectedColumns = strings.Join(a.columnList.Selected, ", ")
	}

	query := fmt.Sprintf("SELECT %s FROM %s", selectedColumns, table)

	// WHERE
	if strings.TrimSpace(a.whereClause.Text) != "" {
		query += " WHERE " + a.whereClause.Text
	}

	// GROUP BY
	if strings.TrimSpace(a.groupBy.Text) != "" {
		query += " GROUP BY " + a.groupBy.Text
	}

	// HAVING
	if strings.TrimSpace(a.havingClause.Text) != "" {
		query += " HAVING " + a.havingClause.Text
	}

	// ORDER BY
	if strings.TrimSpace(a.orderBy.Text) != "" {
		query += " ORDER BY " + a.orderBy.Text
	}

	// LIMIT
	if strings.TrimSpace(a.limitInput.Text) != "" {
		query += " LIMIT " + a.limitInput.Text
	}

	return query
}

func (a *AdvancedQueryWindow) executeQuery() {
	query := a.buildQuery()
	if query == "" {
		a.showError(fmt.Errorf("не выбрана таблица"))
		return
	}

	a.sqlPreview.SetText(query)

	result, err := a.repository.ExecuteQuery(context.Background(), query)
	if err != nil {
		a.showError(err)
		return
	}

	if result.Error != "" {
		a.resultLabel.SetText("Ошибка: " + result.Error)
		return
	}

	a.displayResults(result)
}

func (a *AdvancedQueryWindow) displayResults(result *models.QueryResult) {
	if len(result.Rows) == 0 {
		a.resultTable.Length = func() (int, int) { return 1, 1 }
		a.resultTable.UpdateCell = func(id widget.TableCellID, obj fyne.CanvasObject) {
			label := obj.(*widget.Label)
			if id.Row == 0 && id.Col == 0 {
				label.SetText("Нет данных")
			}
		}
		a.resultLabel.SetText("Запрос выполнен успешно. Найдено 0 строк.")
		return
	}

	// Настройка таблицы
	a.resultTable.Length = func() (int, int) {
		return len(result.Rows) + 1, len(result.Columns) // +1 для заголовков
	}

	a.resultTable.UpdateCell = func(id widget.TableCellID, obj fyne.CanvasObject) {
		label := obj.(*widget.Label)

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
	query := a.buildQuery()
	a.sqlPreview.SetText(query)
}

func (a *AdvancedQueryWindow) clearForm() {
	a.tableSelect.SetSelected("")
	a.columnList.Selected = []string{}
	a.whereClause.SetText("")
	a.orderBy.SetText("")
	a.groupBy.SetText("")
	a.havingClause.SetText("")
	a.limitInput.SetText("100")
	a.sqlPreview.SetText("")
	a.resultLabel.SetText("")
	a.resultTable.Length = func() (int, int) { return 0, 0 }
	a.resultTable.Refresh()
}

func (a *AdvancedQueryWindow) showError(err error) {
	dialog.ShowError(err, a.window)
}

func (a *AdvancedQueryWindow) Show() {
	a.window.Show()
}
