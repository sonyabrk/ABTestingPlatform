package ui

import (
	"context"
	"fmt"
	"testing-platform/db"
	"testing-platform/db/models"
	"testing-platform/pkg/logger"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

// SimilarToSearchWindow представляет окно поиска с регулярными выражениями SIMILAR TO
type SimilarToSearchWindow struct {
	window          fyne.Window
	rep             *db.Repository
	parent          fyne.Window
	tableSelect     *widget.Select
	columnSelect    *widget.Select
	patternEntry    *widget.Entry
	negateCheck     *widget.Check
	resultContainer *fyne.Container
	resultLabel     *widget.Label
	resultTable     *widget.Table // Добавляем таблицу для результатов
}

// NewSimilarToSearchWindow создает новое окно поиска с SIMILAR TO
func NewSimilarToSearchWindow(rep *db.Repository, parent fyne.Window) *SimilarToSearchWindow {
	w := &SimilarToSearchWindow{
		window: fyne.CurrentApp().NewWindow("Поиск с регулярными выражениями"),
		rep:    rep,
		parent: parent,
	}
	w.window.Resize(fyne.NewSize(1000, 800)) // Увеличиваем размер окна
	w.buildUI()
	return w
}

func (w *SimilarToSearchWindow) buildUI() {
	// Выбор таблицы
	w.tableSelect = widget.NewSelect([]string{}, w.onTableSelected)
	w.tableSelect.PlaceHolder = "Выберите таблицу"

	// Выбор столбца
	w.columnSelect = widget.NewSelect([]string{}, nil)
	w.columnSelect.PlaceHolder = "Выберите столбец"

	// Поле для шаблона
	w.patternEntry = widget.NewEntry()
	w.patternEntry.SetPlaceHolder("Введите шаблон регулярного выражения")

	// Чекбокс для отрицания
	w.negateCheck = widget.NewCheck("NOT SIMILAR TO (отрицание условия)", nil)

	// Контейнер для результатов
	w.resultContainer = container.NewStack()
	w.resultLabel = widget.NewLabel("Результаты появятся здесь")
	w.resultLabel.Wrapping = fyne.TextWrapWord

	// Создаем таблицу для результатов
	w.resultTable = widget.NewTable(
		func() (int, int) { return 0, 0 },
		func() fyne.CanvasObject {
			label := widget.NewLabel("")
			label.Wrapping = fyne.TextWrapWord
			return label
		},
		func(i widget.TableCellID, o fyne.CanvasObject) {},
	)

	// Кнопка выполнения поиска
	searchBtn := widget.NewButton("Выполнить поиск", w.executeSearch)
	clearBtn := widget.NewButton("Очистить", w.clearSearch)
	closeBtn := widget.NewButton("Закрыть", func() {
		w.window.Close()
	})

	// Примеры шаблонов
	examplesBtn := widget.NewButton("Примеры шаблонов", w.showExamples)

	// Форма поиска
	searchForm := widget.NewForm(
		widget.NewFormItem("Таблица:", w.tableSelect),
		widget.NewFormItem("Столбец:", w.columnSelect),
		widget.NewFormItem("Шаблон:", w.patternEntry),
	)

	// Создаем контейнер с разделителем для лучшего использования пространства
	split := container.NewVSplit(
		// Верхняя часть - форма поиска
		container.NewVBox(
			widget.NewLabelWithStyle("Поиск с регулярными выражениями SIMILAR TO",
				fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
			searchForm,
			w.negateCheck,
			container.NewHBox(searchBtn, clearBtn, examplesBtn),
			widget.NewSeparator(),
			w.resultLabel,
		),
		// Нижняя часть - результаты (занимает 70% пространства)
		container.NewScroll(w.resultTable),
	)
	split.SetOffset(0.3) // 30% для формы, 70% для результатов

	// Основной контейнер
	content := container.NewBorder(
		nil, // верх
		container.NewHBox(layout.NewSpacer(), closeBtn), // низ с кнопкой закрытия
		nil, nil,
		split, // центральная часть с разделителем
	)

	w.window.SetContent(container.NewPadded(content))

	// Загружаем список таблиц
	w.loadTables()
}

// loadTables загружает список таблиц
func (w *SimilarToSearchWindow) loadTables() {
	ctx := context.Background()
	tables, err := w.rep.GetTableNames(ctx)
	if err != nil {
		dialog.ShowError(fmt.Errorf("Ошибка загрузки таблиц: %v", err), w.window)
		return
	}
	w.tableSelect.Options = tables
	w.tableSelect.Refresh()
}

// onTableSelected вызывается при выборе таблицы
func (w *SimilarToSearchWindow) onTableSelected(table string) {
	if table == "" {
		return
	}

	ctx := context.Background()
	columns, err := w.rep.GetTableColumns(ctx, table)
	if err != nil {
		dialog.ShowError(fmt.Errorf("Ошибка загрузки столбцов: %v", err), w.window)
		return
	}

	// Преобразуем []models.ColumnInfo в []string
	var columnNames []string
	for _, col := range columns {
		columnNames = append(columnNames, col.Name)
	}

	w.columnSelect.Options = columnNames
	w.columnSelect.Refresh()
}

// executeSearch выполняет поиск с регулярным выражением
func (w *SimilarToSearchWindow) executeSearch() {
	table := w.tableSelect.Selected
	column := w.columnSelect.Selected
	pattern := w.patternEntry.Text

	if table == "" || column == "" || pattern == "" {
		dialog.ShowInformation("Не заполнены поля",
			"Пожалуйста, выберите таблицу, столбец и введите шаблон", w.window)
		return
	}

	// Показываем индикатор загрузки
	w.resultLabel.SetText("Выполняется поиск...")
	w.resultTable.Length = func() (int, int) { return 1, 1 }
	w.resultTable.UpdateCell = func(id widget.TableCellID, obj fyne.CanvasObject) {
		label := obj.(*widget.Label)
		if id.Row == 0 && id.Col == 0 {
			label.SetText("Выполняется поиск...")
		}
	}
	w.resultTable.Refresh()

	// Строим запрос
	var condition string
	if w.negateCheck.Checked {
		condition = fmt.Sprintf("%s NOT SIMILAR TO '%s'", column, pattern)
	} else {
		condition = fmt.Sprintf("%s SIMILAR TO '%s'", column, pattern)
	}

	query := fmt.Sprintf("SELECT * FROM %s WHERE %s", table, condition)

	ctx := context.Background()
	result, err := w.rep.ExecuteQuery(ctx, query)
	if err != nil {
		errorMsg := fmt.Sprintf("Ошибка выполнения поиска: %v", err)
		logger.Error("%s", errorMsg)
		dialog.ShowError(fmt.Errorf("%s", errorMsg), w.window)
		w.resultLabel.SetText("❌ Ошибка выполнения поиска")
		return
	}

	if result.Error != "" {
		errorMsg := fmt.Sprintf("Ошибка БД: %s", result.Error)
		logger.Error("%s", errorMsg)
		dialog.ShowError(fmt.Errorf("%s", errorMsg), w.window)
		w.resultLabel.SetText("❌ Ошибка БД")
		return
	}

	// Отображаем результаты
	w.displayResults(result)
	w.resultLabel.SetText(fmt.Sprintf("Найдено строк: %d. Запрос: %s", len(result.Rows), query))
}

// displayResults отображает результаты в таблице
func (w *SimilarToSearchWindow) displayResults(result *models.QueryResult) {
	if len(result.Rows) == 0 {
		w.resultTable.Length = func() (int, int) { return 1, 1 }
		w.resultTable.UpdateCell = func(id widget.TableCellID, obj fyne.CanvasObject) {
			label := obj.(*widget.Label)
			if id.Row == 0 && id.Col == 0 {
				label.SetText("Нет данных, соответствующих условию поиска")
			}
		}
		w.resultTable.Refresh()
		return
	}

	// Настройка таблицы
	w.resultTable.Length = func() (int, int) {
		return len(result.Rows) + 1, len(result.Columns)
	}

	// Автоматически настраиваем ширину колонок
	for col := 0; col < len(result.Columns); col++ {
		maxWidth := float32(150) // Минимальная ширина

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
		w.resultTable.SetColumnWidth(col, maxWidth)
	}

	w.resultTable.UpdateCell = func(id widget.TableCellID, obj fyne.CanvasObject) {
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

	w.resultTable.Refresh()
}

// clearSearch очищает условия поиска
func (w *SimilarToSearchWindow) clearSearch() {
	w.columnSelect.SetSelected("")
	w.patternEntry.SetText("")
	w.negateCheck.SetChecked(false)
	w.resultLabel.SetText("Результаты появятся здесь")
	w.resultTable.Length = func() (int, int) { return 0, 0 }
	w.resultTable.Refresh()
}

// showExamples показывает примеры шаблонов
func (w *SimilarToSearchWindow) showExamples() {
	examples := `Примеры шаблонов для SIMILAR TO:

1. Начинается с 'A': 'A%'
2. Заканчивается на 'z': '%z'
3. Содержит 'test': '%test%'
4. Ровно 3 символа: '___'
5. Буква, затем цифра: '[A-Za-z][0-9]'
6. Только буквы: '^[A-Za-z]+$'
7. Email: '%@%.%'
8. Дата в формате YYYY-MM-DD: '____-__-__'

Спецсимволы:
% - любая последовательность символов
_ - один любой символ
[] - диапазон символов
^ - начало строки
$ - конец строки

Примечание: 
SIMILAR TO использует стандарт SQL регулярных выражений.
Для более сложных шаблонов используйте оператор ~ (POSIX регулярные выражения).`

	dialog.ShowCustom("Примеры шаблонов", "Закрыть",
		container.NewScroll(widget.NewLabel(examples)), w.window)
}

func (w *SimilarToSearchWindow) Show() {
	w.window.Show()
}
