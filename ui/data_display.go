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
}

// NewDataDisplayWindow создает новое окно отображения данных
func NewDataDisplayWindow(mw *MainWindow) *DataDisplayWindow {
	d := &DataDisplayWindow{
		mainWindow: mw,
		window:     mw.app.NewWindow("Данные экспериментов"),
	}
	
	d.window.Resize(fyne.NewSize(1000, 700))
	d.buildUI()
	
	// Добавляем окно в список открытых окон главного окна
	mw.addDataWindow(d)
	
	// Устанавливаем обработчик закрытия окна
	d.window.SetOnClosed(func() {
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

	// Кнопка обновления данных
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
		d.filterPanel,                 // верхняя панель с фильтрами
		container.NewHBox(closeBtn), // нижняя панель с кнопкой закрытия
		nil, nil,
		d.tableContainer, // центральная область с таблицей
	)

	d.window.SetContent(content)
}

// Обновление данных в таблице
func (d *DataDisplayWindow) refreshData() {
	d.updateTable(models.ExperimentFilter{})
	dialog.ShowInformation("Обновлено", "Данные успешно обновлены", d.window)
}

func (d *DataDisplayWindow) updateTable(filter models.ExperimentFilter) {
	ctx := context.Background()
	experiments, err := d.mainWindow.rep.GetExperiments(ctx, filter)
	if err != nil {
		logger.Error("Ошибка получения экспериментов: %v", err)
		dialog.ShowError(fmt.Errorf("не удалось получить эксперименты, проверьте соединение с базой данных"), d.window)
		return
	}

	// подготовка данных для таблицы
	data := make([][]string, 0)
	for _, exp := range experiments {
		data = append(data, []string{
			fmt.Sprintf("%d", exp.ID),
			exp.Name,
			exp.AlgorithmA,
			exp.AlgorithmB,
			fmt.Sprintf("%.2f", exp.UserPercent),
			exp.StartDate.Format("2006-01-02"),
			fmt.Sprintf("%t", exp.IsActive),
			strings.Join(exp.Tags, ", "),
		})
	}

	// создание новой таблицы
	table := widget.NewTable(
		func() (int, int) {
			return len(data) + 1, 8
		},
		func() fyne.CanvasObject {
			// Создаем label с выравниванием по центру
			label := widget.NewLabel("")
			label.Alignment = fyne.TextAlignCenter
			return label
		},
		func(i widget.TableCellID, o fyne.CanvasObject) {
			label := o.(*widget.Label)
			label.Alignment = fyne.TextAlignCenter // Выравнивание по центру

			// 1ая строка - заголовки
			if i.Row == 0 {
				headers := []string{"ID", "Название", "Алгоритм A", "Алгоритм B", "Пользователи %", "Дата начала", "Активен", "Теги"}
				if i.Col < len(headers) {
					label.SetText(headers[i.Col])
					label.TextStyle = fyne.TextStyle{Bold: true}
				}
			} else {
				// смещение данных на одну строку вниз
				if i.Row-1 < len(data) && i.Col < len(data[i.Row-1]) {
					label.SetText(data[i.Row-1][i.Col])
					label.TextStyle = fyne.TextStyle{}
				}
			}
		})

	// установка размеров столбцов
	table.SetColumnWidth(0, 60)  // ID
	table.SetColumnWidth(1, 160) // Name
	table.SetColumnWidth(2, 130) // Algorithm A
	table.SetColumnWidth(3, 130) // Algorithm B
	table.SetColumnWidth(4, 130) // User Percent
	table.SetColumnWidth(5, 120) // Start Date
	table.SetColumnWidth(6, 90)  // Is Active
	table.SetColumnWidth(7, 160) // Tags

	// обновление контейнера с таблицей
	d.tableContainer.Objects = []fyne.CanvasObject{table}
	d.tableContainer.Refresh()
}

func (d *DataDisplayWindow) Show() {
	d.window.Show()
}

// Обновленная функция showDataDisplayWindow в MainWindow
func (mw *MainWindow) showDataDisplayWindow() {
	dataWin := NewDataDisplayWindow(mw)
	dataWin.Show()
}

// Обновленная функция showSummaryWindow в MainWindow
func (mw *MainWindow) showSummaryWindow() {
	// создание нового окна
	summaryWin := mw.app.NewWindow("Сводные данные экспериментов")
	summaryWin.Resize(fyne.NewSize(1000, 600))

	// получение данных из репозитория
	ctx := context.Background()
	results, err := mw.rep.GetExperimentResultsWithDetails(ctx)
	if err != nil {
		logger.Error("Ошибка получения сводных данных: %v", err)
		dialog.ShowError(fmt.Errorf("не удалось получить сводные данные, проверьте соединение с базой данных"), mw.window)
		return
	}

	// подготовка данных для таблицы
	data := make([][]string, 0)
	for _, res := range results {
		avgRatingStr := fmt.Sprintf("%.2f", res.AvgRating)

		data = append(data, []string{
			fmt.Sprintf("%d", res.ID),
			res.Name,
			res.AlgorithmA,
			res.AlgorithmB,
			fmt.Sprintf("%d", res.TotalResults),
			fmt.Sprintf("%d", res.TotalClicks),
			avgRatingStr,
		})
	}

	// создание таблицы
	table := widget.NewTable(
		func() (int, int) {
			return len(data) + 1, 7
		},
		func() fyne.CanvasObject {
			// Создаем label с выравниванием по центру
			label := widget.NewLabel("")
			label.Alignment = fyne.TextAlignCenter
			return label
		},
		func(i widget.TableCellID, o fyne.CanvasObject) {
			label := o.(*widget.Label)
			label.Alignment = fyne.TextAlignCenter // Выравнивание по центру

			if i.Row == 0 {
				headers := []string{"ID", "Название", "Алгоритм A", "Алгоритм B", "Результаты", "Клики", "Средний рейтинг"}
				if i.Col < len(headers) {
					label.SetText(headers[i.Col])
					label.TextStyle = fyne.TextStyle{Bold: true}
				}
			} else {
				if i.Row-1 < len(data) && i.Col < len(data[i.Row-1]) {
					label.SetText(data[i.Row-1][i.Col])
					label.TextStyle = fyne.TextStyle{}
				}
			}
		})

	// настройка размеров столбцов
	table.SetColumnWidth(0, 60)  // ID
	table.SetColumnWidth(1, 160) // Name
	table.SetColumnWidth(2, 130) // Algorithm A
	table.SetColumnWidth(3, 130) // Algorithm B
	table.SetColumnWidth(4, 120) // Total Results
	table.SetColumnWidth(5, 120) // Total Clicks
	table.SetColumnWidth(6, 130) // Avg Rating

	// кнопка закрытия
	closeBtn := widget.NewButton("Закрыть", func() {
		summaryWin.Close()
	})
	
	// кнопка обновления
	refreshBtn := widget.NewButton("Обновить", func() {
		// Закрываем и открываем заново для обновления данных
		summaryWin.Close()
		mw.showSummaryWindow()
	})

	// создание контейнера с таблицей и кнопкой
	content := container.NewBorder(nil, container.NewHBox(refreshBtn, closeBtn), nil, nil, table)
	summaryWin.SetContent(content)
	summaryWin.Show()
}