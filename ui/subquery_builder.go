// subquery_builder.go (полная исправленная версия)
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

type SubqueryBuilder struct {
	window     fyne.Window
	repository *db.Repository
	onApply    func(condition *models.SubqueryCondition)

	// Элементы управления
	mainTableSelect  *widget.Select
	mainColumnSelect *widget.Select
	operatorSelect   *widget.Select
	typeSelect       *widget.Select
	subqueryTable    *widget.Select
	subqueryColumn   *widget.Select
	whereContainer   *fyne.Container
	sqlPreview       *widget.Entry

	// Данные
	tables          []string
	tableColumns    map[string][]models.ColumnInfo
	whereConditions []models.WhereCondition
}

func NewSubqueryBuilder(repo *db.Repository, app fyne.App, onApply func(condition *models.SubqueryCondition)) *SubqueryBuilder {
	s := &SubqueryBuilder{
		repository:   repo,
		window:       app.NewWindow("Построитель подзапросов"),
		onApply:      onApply,
		tableColumns: make(map[string][]models.ColumnInfo),
	}

	s.buildUI()
	s.loadTables()
	return s
}

// SetMainTable устанавливает основную таблицу в построителе подзапросов
func (s *SubqueryBuilder) SetMainTable(table string) {
	if s.mainTableSelect != nil {
		s.mainTableSelect.SetSelected(table)
	}
}

func (s *SubqueryBuilder) buildUI() {
	s.mainTableSelect = widget.NewSelect([]string{}, s.onMainTableSelected)
	s.mainTableSelect.PlaceHolder = "Основная таблица"

	s.mainColumnSelect = widget.NewSelect([]string{}, nil)
	s.mainColumnSelect.PlaceHolder = "Столбец основной таблицы"

	s.typeSelect = widget.NewSelect([]string{"ANY", "ALL", "EXISTS"}, s.onTypeSelected)
	s.typeSelect.SetSelected("ANY")
	s.typeSelect.PlaceHolder = "Тип подзапроса"

	s.operatorSelect = widget.NewSelect([]string{"=", "!=", ">", "<", ">=", "<="}, nil)
	s.operatorSelect.SetSelected("=")
	s.operatorSelect.PlaceHolder = "Оператор сравнения"

	s.subqueryTable = widget.NewSelect([]string{}, s.onSubqueryTableSelected)
	s.subqueryTable.PlaceHolder = "Таблица подзапроса"

	s.subqueryColumn = widget.NewSelect([]string{}, nil)
	s.subqueryColumn.PlaceHolder = "Столбец подзапроса"

	s.sqlPreview = widget.NewMultiLineEntry()
	s.sqlPreview.Wrapping = fyne.TextWrapOff
	s.sqlPreview.SetPlaceHolder("Здесь будет показан SQL подзапроса...")

	s.whereContainer = container.NewVBox()

	// Кнопки
	addWhereBtn := widget.NewButton("Добавить условие WHERE", s.addWhereCondition)
	applyBtn := widget.NewButton("Применить подзапрос", s.applySubquery)
	cancelBtn := widget.NewButton("Отмена", func() { s.window.Close() })
	previewBtn := widget.NewButton("Показать SQL", s.previewSQL)

	// Компоновка
	mainSection := container.NewVBox(
		widget.NewLabel("Основная таблица:"),
		s.mainTableSelect,
		s.mainColumnSelect,
	)

	subquerySection := container.NewVBox(
		widget.NewLabel("Тип подзапроса:"),
		s.typeSelect,
		s.operatorSelect,
		s.subqueryTable,
		s.subqueryColumn,
	)

	whereSection := container.NewVBox(
		widget.NewLabel("Условия WHERE в подзапросе:"),
		addWhereBtn,
		container.NewScroll(s.whereContainer),
	)

	leftPanel := container.NewVBox(
		mainSection,
		widget.NewSeparator(),
		subquerySection,
	)

	rightPanel := container.NewVBox(
		whereSection,
		widget.NewSeparator(),
		s.sqlPreview,
		container.NewHBox(applyBtn, previewBtn, cancelBtn),
	)

	split := container.NewHSplit(leftPanel, rightPanel)
	split.SetOffset(0.5)

	s.window.SetContent(split)
	s.window.Resize(fyne.NewSize(900, 600))
}

func (s *SubqueryBuilder) onTypeSelected(selected string) {
	// Показываем/скрываем элементы в зависимости от типа
	if selected == "EXISTS" {
		s.operatorSelect.Hide()
		s.subqueryColumn.Hide()
	} else {
		s.operatorSelect.Show()
		s.subqueryColumn.Show()
	}
	s.previewSQL()
}

func (s *SubqueryBuilder) onMainTableSelected(table string) {
	if table == "" {
		return
	}
	s.loadTableColumns(table, s.mainColumnSelect)
	s.previewSQL()
}

func (s *SubqueryBuilder) onSubqueryTableSelected(table string) {
	if table == "" {
		return
	}
	s.loadTableColumns(table, s.subqueryColumn)
	s.previewSQL()
}

func (s *SubqueryBuilder) loadTables() {
	ctx := context.Background()
	tables, err := s.repository.GetTableNames(ctx)
	if err != nil {
		s.showError(fmt.Errorf("не удалось загрузить список таблиц: %v", err))
		return
	}

	s.tables = tables
	s.mainTableSelect.Options = tables
	s.subqueryTable.Options = tables
	s.mainTableSelect.Refresh()
	s.subqueryTable.Refresh()
}

func (s *SubqueryBuilder) loadTableColumns(table string, selectWidget *widget.Select) {
	if _, exists := s.tableColumns[table]; !exists {
		ctx := context.Background()
		columns, err := s.repository.GetTableColumns(ctx, table)
		if err != nil {
			s.showError(fmt.Errorf("не удалось загрузить столбцы таблицы %s: %v", table, err))
			return
		}
		s.tableColumns[table] = columns
	}

	var columnNames []string
	for _, col := range s.tableColumns[table] {
		columnNames = append(columnNames, col.Name)
	}

	selectWidget.Options = columnNames
	selectWidget.Refresh()
}

func (s *SubqueryBuilder) addWhereCondition() {
	columnSelect := widget.NewSelect([]string{}, nil)
	operatorSelect := widget.NewSelect([]string{"=", "!=", ">", "<", ">=", "<=", "LIKE", "IN", "IS NULL", "IS NOT NULL"}, nil)
	operatorSelect.SetSelected("=")
	valueEntry := widget.NewEntry()
	valueEntry.SetPlaceHolder("Значение")

	// Загружаем столбцы если таблица выбрана
	if s.subqueryTable.Selected != "" {
		columnSelect.Options = s.getColumnNames(s.subqueryTable.Selected)
		columnSelect.Refresh()
	}

	deleteBtn := widget.NewButton("✕", nil)

	conditionRow := container.NewHBox(
		columnSelect,
		operatorSelect,
		valueEntry,
		deleteBtn,
	)

	// Добавляем условие
	condition := models.WhereCondition{}
	s.whereConditions = append(s.whereConditions, condition)
	conditionIndex := len(s.whereConditions) - 1

	// Обновляем условие при изменении
	updateCondition := func() {
		if conditionIndex < len(s.whereConditions) {
			s.whereConditions[conditionIndex] = models.WhereCondition{
				Column:   columnSelect.Selected,
				Operator: operatorSelect.Selected,
				Value:    valueEntry.Text,
			}
		}
		s.previewSQL()
	}

	columnSelect.OnChanged = func(string) { updateCondition() }
	operatorSelect.OnChanged = func(string) { updateCondition() }
	valueEntry.OnChanged = func(string) { updateCondition() }

	deleteBtn.OnTapped = func() {
		if conditionIndex < len(s.whereConditions) {
			s.whereConditions = append(s.whereConditions[:conditionIndex], s.whereConditions[conditionIndex+1:]...)
			s.whereContainer.Remove(conditionRow)
			s.previewSQL()
		}
	}

	s.whereContainer.Add(conditionRow)
	s.previewSQL()
}

func (s *SubqueryBuilder) getColumnNames(table string) []string {
	var names []string
	if columns, exists := s.tableColumns[table]; exists {
		for _, col := range columns {
			names = append(names, col.Name)
		}
	}
	return names
}

func (s *SubqueryBuilder) buildWhereClause() string {
	if len(s.whereConditions) == 0 {
		return "1=1"
	}

	var conditions []string
	for _, cond := range s.whereConditions {
		if cond.Column == "" || cond.Operator == "" {
			continue
		}

		var conditionStr string
		switch cond.Operator {
		case "IS NULL", "IS NOT NULL":
			conditionStr = fmt.Sprintf("%s %s", cond.Column, cond.Operator)
		default:
			// Экранируем значение
			escapedValue := strings.ReplaceAll(cond.Value, "'", "''")
			conditionStr = fmt.Sprintf("%s %s '%s'", cond.Column, cond.Operator, escapedValue)
		}
		conditions = append(conditions, conditionStr)
	}

	return strings.Join(conditions, " AND ")
}

func (s *SubqueryBuilder) previewSQL() {
	if s.mainTableSelect.Selected == "" || s.mainColumnSelect.Selected == "" ||
		s.typeSelect.Selected == "" || s.subqueryTable.Selected == "" {
		return
	}

	var sql string
	subqueryType := s.typeSelect.Selected
	whereClause := s.buildWhereClause()

	switch subqueryType {
	case "EXISTS":
		sql = fmt.Sprintf("SELECT * FROM %s WHERE EXISTS (SELECT 1 FROM %s WHERE %s)",
			s.mainTableSelect.Selected, s.subqueryTable.Selected, whereClause)
	case "ANY", "ALL":
		if s.subqueryColumn.Selected == "" || s.operatorSelect.Selected == "" {
			return
		}
		sql = fmt.Sprintf("SELECT * FROM %s WHERE %s %s %s (SELECT %s FROM %s WHERE %s)",
			s.mainTableSelect.Selected, s.mainColumnSelect.Selected, s.operatorSelect.Selected,
			subqueryType, s.subqueryColumn.Selected, s.subqueryTable.Selected, whereClause)
	}

	s.sqlPreview.SetText(sql)
}

func (s *SubqueryBuilder) applySubquery() {
	if s.mainTableSelect.Selected == "" || s.mainColumnSelect.Selected == "" ||
		s.typeSelect.Selected == "" || s.subqueryTable.Selected == "" {
		s.showError(fmt.Errorf("заполните все обязательные поля"))
		return
	}

	if s.typeSelect.Selected != "EXISTS" && (s.subqueryColumn.Selected == "" || s.operatorSelect.Selected == "") {
		s.showError(fmt.Errorf("для подзапросов ANY/ALL укажите столбец и оператор"))
		return
	}

	condition := &models.SubqueryCondition{
		Type:       s.typeSelect.Selected,
		MainColumn: s.mainColumnSelect.Selected,
		Operator:   s.operatorSelect.Selected,
		Subquery:   s.sqlPreview.Text,
	}

	if s.onApply != nil {
		s.onApply(condition)
	}
	s.window.Close()
}

func (s *SubqueryBuilder) showError(err error) {
	dialog.ShowError(err, s.window)
}

func (s *SubqueryBuilder) Show() {
	s.window.Show()
}
