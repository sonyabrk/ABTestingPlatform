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

func (mw *MainWindow) showFilterDialog(callback func(models.ExperimentFilter)) {
	// создание элементов управления для фильтра
	algorithmA := widget.NewSelect([]string{"", "collaborative", "content_based", "hybrid", "popularity_based"}, nil)
	algorithmB := widget.NewSelect([]string{"", "collaborative", "content_based", "hybrid", "popularity_based"}, nil)
	isActive := widget.NewCheck("Активные", nil)
	dateFrom := widget.NewEntry()
	dateTo := widget.NewEntry()

	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Алгоритм A", Widget: algorithmA},
			{Text: "Алгоритм B", Widget: algorithmB},
			{Text: "Только активные", Widget: isActive},
			{Text: "Дата от (YYYY-MM-DD)", Widget: dateFrom},
			{Text: "Дата до (YYYY-MM-DD)", Widget: dateTo},
		},
	}

	// показ диалог с обработчиком подтверждения
	dialog.ShowCustomConfirm(
		"Фильтр экспериментов",
		"Применить",
		"Отмена",
		form,
		func(confirmed bool) {
			if !confirmed {
				return
			}

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
			callback(filter)
		},
		mw.window,
	)
}

func (mw *MainWindow) showDataDisplayWindow() {
	// создание окна
	dataWin := mw.app.NewWindow("Данные экспериментов")
	dataWin.Resize(fyne.NewSize(1000, 600))
	// создание контейнера для таблицы
	tableContainer := container.NewStack()
	// фун-ия для обновления таблицы
	updateTable := func(filter models.ExperimentFilter) {
		ctx := context.Background()
		experiments, err := mw.rep.GetExperiments(ctx, filter)
		if err != nil {
			logger.Error("Ошибка получения экспериментов: %v", err)
			dialog.ShowError(fmt.Errorf("не удалось получить эксперименты, проверьте соединение с базой данных"), dataWin)
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
				return widget.NewLabel("template")
			},
			func(i widget.TableCellID, o fyne.CanvasObject) {
				label := o.(*widget.Label)

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
		table.SetColumnWidth(0, 50)  // ID
		table.SetColumnWidth(1, 150) // Name
		table.SetColumnWidth(2, 120) // Algorithm A
		table.SetColumnWidth(3, 120) // Algorithm B
		table.SetColumnWidth(4, 80)  // User Percent
		table.SetColumnWidth(5, 100) // Start Date
		table.SetColumnWidth(6, 80)  // Is Active
		table.SetColumnWidth(7, 150) // Tags

		// обновление контейнера с таблицей
		tableContainer.Objects = []fyne.CanvasObject{table}
		tableContainer.Refresh()
	}

	// кнопка фильтра
	filterBtn := widget.NewButton("Фильтр", func() {
		mw.showFilterDialog(updateTable)
	})

	// кнопка очистки фильтров
	clearFilterBtn := widget.NewButton("Очистить фильтры", func() {
		updateTable(models.ExperimentFilter{})
	})

	// кнопка закрытия
	closeBtn := widget.NewButton("Закрыть", func() {
		dataWin.Close()
	})
	// первоначальная загрузка данных
	updateTable(models.ExperimentFilter{})
	// создание основного контейнера
	content := container.NewBorder(
		container.NewHBox(filterBtn, clearFilterBtn), // верхняя панель с кнопками
		container.NewHBox(closeBtn),                  // нижняя панель с кнопкой закрытия
		nil, nil,
		tableContainer, // центральная область с таблицей
	)

	dataWin.SetContent(content)
	dataWin.Show()
}

// функция создает и отображает окно со сводными данными экспериментов
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
			return widget.NewLabel("template")
		},
		func(i widget.TableCellID, o fyne.CanvasObject) {
			label := o.(*widget.Label)
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
	table.SetColumnWidth(0, 50)  // ID
	table.SetColumnWidth(1, 150) // Name
	table.SetColumnWidth(2, 120) // Algorithm A
	table.SetColumnWidth(3, 120) // Algorithm B
	table.SetColumnWidth(4, 100) // Total Results
	table.SetColumnWidth(5, 100) // Total Clicks
	table.SetColumnWidth(6, 120) // Avg Rating

	// кнопка закрытия
	closeBtn := widget.NewButton("Закрыть", func() {
		summaryWin.Close()
	})

	// создание контейнера с таблицей и кнопкой
	content := container.NewBorder(nil, closeBtn, nil, nil, table)
	summaryWin.SetContent(content)
	summaryWin.Show()
}
