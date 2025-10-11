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
	tableColumns map[string][]string
}

func NewJoinBuilderWindow(repo *db.Repository, mainWindow fyne.Window) *JoinBuilderWindow {
	j := &JoinBuilderWindow{
		repository:   repo,
		mainWindow:   mainWindow,
		window:       fyne.CurrentApp().NewWindow("Мастер JOIN"),
		tableColumns: make(map[string][]string),
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

	j.resultLabel = widget.NewLabel("Постройте JOIN для просмотра результатов")
	j.resultLabel.Wrapping = fyne.TextWrapWord

	j.resultTable = widget.NewTable(
		func() (int, int) { return 0, 0 },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(i widget.TableCellID, o fyne.CanvasObject) {},
	)

	// Для дополнительных JOIN
	j.additionalJoins = widget.NewAccordion()

	addJoinBtn := widget.NewButton("Добавить еще JOIN", j.addAdditionalJoin)
	executeBtn := widget.NewButton("Выполнить JOIN", j.executeJoin)
	clearBtn := widget.NewButton("Очистить", j.clearForm)

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

	controls := container.NewVBox(
		joinForm,
		addJoinBtn,
		j.additionalJoins,
		container.NewHBox(executeBtn, clearBtn),
		widget.NewLabel("SQL запрос:"),
		j.sqlPreview,
		j.resultLabel,
	)

	content := container.NewBorder(
		container.NewScroll(controls), nil, nil, nil,
		container.NewScroll(j.resultTable),
	)

	j.window.SetContent(content)
	j.window.Resize(fyne.NewSize(1000, 700))
}

func (j *JoinBuilderWindow) loadTables() {
	tables, err := j.repository.GetTables(context.Background())
	if err != nil {
		j.showError(err)
		return
	}
	j.tables = tables
	j.mainTableSelect.Options = tables
	j.joinTableSelect.Options = tables
	j.mainTableSelect.Refresh()
	j.joinTableSelect.Refresh()
}

func (j *JoinBuilderWindow) onMainTableSelected(table string) {
	j.loadTableColumns(table)
}

func (j *JoinBuilderWindow) onJoinTableSelected(table string) {
	j.loadTableColumns(table)
}

func (j *JoinBuilderWindow) loadTableColumns(table string) {
	if table == "" {
		return
	}

	// Если уже загружены, используем кэш
	if _, exists := j.tableColumns[table]; exists {
		j.updateColumnSelectors()
		return
	}

	columns, err := j.repository.GetTableColumns(context.Background(), table)
	if err != nil {
		j.showError(err)
		return
	}

	var columnNames []string
	for _, col := range columns {
		columnNames = append(columnNames, col.Name)
	}

	j.tableColumns[table] = columnNames
	j.updateColumnSelectors()
}

func (j *JoinBuilderWindow) updateColumnSelectors() {
	mainTable := j.mainTableSelect.Selected
	joinTable := j.joinTableSelect.Selected

	if mainTable != "" {
		j.mainColumnSelect.Options = j.tableColumns[mainTable]
		j.mainColumnSelect.Refresh()
	}

	if joinTable != "" {
		j.joinColumnSelect.Options = j.tableColumns[joinTable]
		j.joinColumnSelect.Refresh()
	}
}

func (j *JoinBuilderWindow) addAdditionalJoin() {
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
		if cols, exists := j.tableColumns[table]; exists {
			joinColumn.Options = cols
			joinColumn.Refresh()
		} else {
			j.loadTableColumns(table)
		}
	}

	removeBtn := widget.NewButton("Удалить", nil)

	joinForm := container.NewHBox(
		joinType,
		tableSelect,
		widget.NewLabel("ON"),
		mainColumn,
		widget.NewLabel("="),
		joinColumn,
		removeBtn,
	)

	item := widget.NewAccordionItem(fmt.Sprintf("JOIN %d", len(j.additionalJoins.Items)+1), joinForm)

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
}

func (j *JoinBuilderWindow) buildJoinQuery() (string, error) {
	if j.mainTableSelect.Selected == "" {
		return "", fmt.Errorf("не выбрана основная таблица")
	}
	if j.joinTableSelect.Selected == "" {
		return "", fmt.Errorf("не выбрана таблица для JOIN")
	}
	if j.mainColumnSelect.Selected == "" || j.joinColumnSelect.Selected == "" {
		return "", fmt.Errorf("не выбраны столбцы для соединения")
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
	for _, item := range j.additionalJoins.Items {
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
			}
		}
	}

	query += " LIMIT 100"
	return query, nil
}

func (j *JoinBuilderWindow) executeJoin() {
	query, err := j.buildJoinQuery()
	if err != nil {
		j.showError(err)
		return
	}

	j.sqlPreview.SetText(query)

	result, err := j.repository.ExecuteQuery(context.Background(), query)
	if err != nil {
		j.showError(err)
		return
	}

	if result.Error != "" {
		j.resultLabel.SetText("Ошибка: " + result.Error)
		return
	}

	j.displayResults(result)
	j.resultLabel.SetText(fmt.Sprintf("Найдено %d строк", len(result.Rows)))
}

func (j *JoinBuilderWindow) displayResults(result *models.QueryResult) {
	if len(result.Rows) == 0 {
		j.resultTable.Length = func() (int, int) { return 1, 1 }
		j.resultTable.UpdateCell = func(id widget.TableCellID, obj fyne.CanvasObject) {
			label := obj.(*widget.Label)
			if id.Row == 0 && id.Col == 0 {
				label.SetText("Нет данных")
			}
		}
		return
	}

	j.resultTable.Length = func() (int, int) {
		return len(result.Rows) + 1, len(result.Columns)
	}

	j.resultTable.UpdateCell = func(id widget.TableCellID, obj fyne.CanvasObject) {
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
	j.resultLabel.SetText("")
	j.resultTable.Length = func() (int, int) { return 0, 0 }
	j.resultTable.Refresh()
}

func (j *JoinBuilderWindow) showError(err error) {
	dialog.ShowError(err, j.window)
}

func (j *JoinBuilderWindow) Show() {
	j.window.Show()
}
