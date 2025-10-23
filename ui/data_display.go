package ui

import (
	"context"
	"fmt"
	"strings"
	"testing-platform/db/models"
	"testing-platform/pkg/logger"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// DataDisplayWindow представляет окно отображения данных
type DataDisplayWindow struct {
	window         fyne.Window
	mainWindow     *MainWindow
	tableContainer *fyne.Container
	filterPanel    *fyne.Container
	currentFilter  models.ExperimentFilter // Сохраняем текущий фильтр
}

// NewDataDisplayWindow создает новое окно отображения данных
func NewDataDisplayWindow(mw *MainWindow) *DataDisplayWindow {
	logger.Info("Создание нового окна данных. Полученное главное окно: %p", mw)

	d := &DataDisplayWindow{
		mainWindow: mw,
		window:     mw.app.NewWindow("Данные экспериментов"),
	}

	d.window.Resize(fyne.NewSize(1000, 700))

	// Сразу регистрируем окно в главном окне
	logger.Info("Регистрация нового окна данных %p в главном окне %p", d, mw)
	mw.addDataWindow(d)

	d.buildUI()

	// Устанавливаем обработчик закрытия окна
	d.window.SetOnClosed(func() {
		logger.Info("Закрытие окна данных %p, удаление из главного окна %p", d, mw)
		mw.removeDataWindow(d)
	})

	return d
}

func (d *DataDisplayWindow) buildUI() {
	// создание контейнера для таблицы
	d.tableContainer = container.NewStack()

	// Элементы управления для фильтра (выпадающие списки и поля ввода)
	algorithmA := widget.NewSelect([]string{"", "collaborative", "content_based", "hybrid", "popularity_based"}, nil)
	algorithmB := widget.NewSelect([]string{"", "collaborative", "content_based", "hybrid", "popularity_based"}, nil)
	isActive := widget.NewCheck("Только активные", nil)
	dateFrom := widget.NewEntry()
	dateTo := widget.NewEntry()

	// Устанавливаем плейсхолдеры для полей дат
	dateFrom.SetPlaceHolder("YYYY-MM-DD")
	dateTo.SetPlaceHolder("YYYY-MM-DD")

	// Функция применения фильтра
	applyFilter := func() {
		filter := models.ExperimentFilter{}

		// Заполняем фильтр из значений формы
		if algorithmA.Selected != "" {
			filter.AlgorithmA = algorithmA.Selected
		}
		if algorithmB.Selected != "" {
			filter.AlgorithmB = algorithmB.Selected
		}
		if isActive.Checked {
			filter.IsActive = &isActive.Checked
		}

		// Парсим даты
		if dateFrom.Text != "" {
			if t, err := time.Parse("2006-01-02", dateFrom.Text); err == nil {
				filter.StartDateFrom = t
			}
		}
		if dateTo.Text != "" {
			if t, err := time.Parse("2006-01-02", dateTo.Text); err == nil {
				filter.StartDateTo = t
			}
		}

		d.updateTable(filter)
	}

	// Функция очистки фильтров
	clearFilters := func() {
		algorithmA.SetSelected("")
		algorithmB.SetSelected("")
		isActive.SetChecked(false)
		dateFrom.SetText("")
		dateTo.SetText("")
		d.updateTable(models.ExperimentFilter{})
	}

	refreshBtn := widget.NewButton("Обновить данные", func() {
		d.refreshData()
	})

	// Создаем панель фильтров
	d.filterPanel = container.NewVBox(
		widget.NewLabel("Фильтры:"),
		container.NewHBox(
			container.NewVBox(
				widget.NewLabel("Алгоритм A:"),
				algorithmA,
			),
			container.NewVBox(
				widget.NewLabel("Алгоритм B:"),
				algorithmB,
			),
			container.NewVBox(
				widget.NewLabel(" "), // Пустой label для выравнивания
				isActive,
			),
			container.NewVBox(
				widget.NewLabel("Дата от:"),
				dateFrom,
			),
			container.NewVBox(
				widget.NewLabel("Дата до:"),
				dateTo,
			),
		),
		container.NewHBox(
			widget.NewButton("Применить фильтр", applyFilter),
			widget.NewButton("Очистить фильтры", clearFilters),
			refreshBtn,
		),
	)

	// Добавляем отступы для лучшего вида
	d.filterPanel = container.NewPadded(d.filterPanel)

	// кнопка закрытия
	closeBtn := widget.NewButton("Закрыть", func() {
		d.window.Close()
	})

	// первоначальная загрузка данных
	d.updateTable(models.ExperimentFilter{})

	// создание основного контейнера
	content := container.NewBorder(
		d.filterPanel,               // верхняя панель с фильтрами
		container.NewHBox(closeBtn), // нижняя панель с кнопкой закрытия
		nil, nil,
		d.tableContainer, // центральная область с таблицей
	)

	d.window.SetContent(content)
}

// RefreshData принудительно обновляет данные в окне
func (d *DataDisplayWindow) RefreshData() {
	logger.Info("Обновление данных в окне отображения")
	d.refreshDataSilent()
}

// refreshDataSilent обновляет данные без показа диалога
func (d *DataDisplayWindow) refreshDataSilent() {
	d.updateTable(d.currentFilter) // Используем текущий фильтр
}

// Обновим существующий refreshData
func (d *DataDisplayWindow) refreshData() {
	d.refreshDataSilent()
	dialog.ShowInformation("Обновлено", "Данные успешно обновлены", d.window)
}

// convertValueToString конвертирует любое значение в строку для отображения
func convertValueToString(value interface{}) string {
	if value == nil {
		return ""
	}

	switch v := value.(type) {
	case string:
		return v
	case int, int32, int64:
		return fmt.Sprintf("%d", v)
	case float32, float64:
		return fmt.Sprintf("%.2f", v)
	case bool:
		return fmt.Sprintf("%t", v)
	case []string:
		return strings.Join(v, ", ")
	case []byte:
		return string(v)
	case time.Time:
		return v.Format("2006-01-02 15:04:05")
	default:
		return fmt.Sprintf("%v", v)
	}
}

func (d *DataDisplayWindow) updateTable(filter models.ExperimentFilter) {
	// Сохраняем текущий фильтр
	d.currentFilter = filter

	ctx := context.Background()

	// Получаем динамические данные из таблицы experiments
	query := "SELECT * FROM experiments"
	if filter.AlgorithmA != "" || filter.AlgorithmB != "" || filter.IsActive != nil || !filter.StartDateFrom.IsZero() || !filter.StartDateTo.IsZero() {
		query += " WHERE 1=1"
		var conditions []string
		var args []interface{}
		argCount := 1

		if filter.AlgorithmA != "" {
			conditions = append(conditions, fmt.Sprintf("algorithm_a = $%d", argCount))
			args = append(args, filter.AlgorithmA)
			argCount++
		}
		if filter.AlgorithmB != "" {
			conditions = append(conditions, fmt.Sprintf("algorithm_b = $%d", argCount))
			args = append(args, filter.AlgorithmB)
			argCount++
		}
		if filter.IsActive != nil {
			conditions = append(conditions, fmt.Sprintf("is_active = $%d", argCount))
			args = append(args, *filter.IsActive)
			argCount++
		}
		if !filter.StartDateFrom.IsZero() {
			conditions = append(conditions, fmt.Sprintf("start_date >= $%d", argCount))
			args = append(args, filter.StartDateFrom)
			argCount++
		}
		if !filter.StartDateTo.IsZero() {
			conditions = append(conditions, fmt.Sprintf("start_date <= $%d", argCount))
			args = append(args, filter.StartDateTo)
			argCount++
		}

		if len(conditions) > 0 {
			query += " AND " + strings.Join(conditions, " AND ")
		}

		query += " ORDER BY start_date DESC"

		// Выполняем запрос с фильтрами
		result, err := d.mainWindow.rep.ExecuteQuery(ctx, query)
		if err != nil {
			logger.Error("Ошибка получения экспериментов: %v", err)
			dialog.ShowError(fmt.Errorf("не удалось получить эксперименты, проверьте соединение с базой данных"), d.window)
			return
		}

		d.createDynamicTable(result)
	} else {
		// Без фильтров - простой запрос
		result, err := d.mainWindow.rep.ExecuteQuery(ctx, query+" ORDER BY start_date DESC")
		if err != nil {
			logger.Error("Ошибка получения экспериментов: %v", err)
			dialog.ShowError(fmt.Errorf("не удалось получить эксперименты, проверьте соединение с базой данных"), d.window)
			return
		}

		d.createDynamicTable(result)
	}
}

// createDynamicTable создает таблицу с динамическими столбцами
func (d *DataDisplayWindow) createDynamicTable(result *models.QueryResult) {
	if result.Error != "" {
		logger.Error("Ошибка в результате запроса: %s", result.Error)
		dialog.ShowError(fmt.Errorf("ошибка выполнения запроса: %s", result.Error), d.window)
		return
	}

	// Подготавливаем данные для таблицы
	data := make([][]string, 0)

	// Добавляем строки данных
	for _, row := range result.Rows {
		rowData := make([]string, 0)
		for _, col := range result.Columns {
			value := row[col]
			rowData = append(rowData, convertValueToString(value))
		}
		data = append(data, rowData)
	}

	// Создаем таблицу с динамическим количеством столбцов
	table := widget.NewTable(
		func() (int, int) {
			return len(data) + 1, len(result.Columns) // +1 для заголовков
		},
		func() fyne.CanvasObject {
			label := widget.NewLabel("")
			label.Alignment = fyne.TextAlignCenter
			return label
		},
		func(i widget.TableCellID, o fyne.CanvasObject) {
			label := o.(*widget.Label)
			label.Alignment = fyne.TextAlignCenter

			// Первая строка - заголовки
			if i.Row == 0 {
				if i.Col < len(result.Columns) {
					label.SetText(result.Columns[i.Col])
					label.TextStyle = fyne.TextStyle{Bold: true}
				}
			} else {
				// Данные
				if i.Row-1 < len(data) && i.Col < len(data[i.Row-1]) {
					label.SetText(data[i.Row-1][i.Col])
					label.TextStyle = fyne.TextStyle{}
				}
			}
		})

	// Устанавливаем размеры столбцов (можно сделать адаптивными)
	for i := 0; i < len(result.Columns); i++ {
		table.SetColumnWidth(i, 120) // Базовая ширина для всех столбцов
	}

	// Обновление контейнера с таблицей
	d.tableContainer.Objects = []fyne.CanvasObject{table}
	d.tableContainer.Refresh()

	logger.Info("Таблица обновлена: %d строк, %d столбцов", len(data), len(result.Columns))
}

func (d *DataDisplayWindow) Show() {
	d.window.Show()
}
