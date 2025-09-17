package ui

import (
	"context"
	"fmt"
	"strings"
	"testing-platform/pkg/logger"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

func (mw *MainWindow) showDataDisplayWindow() {
	// создание нового окна
	dataWin := mw.app.NewWindow("Данные экспериментов")
	dataWin.Resize(fyne.NewSize(800, 600))

	ctx := context.Background()
	experiments, err := mw.rep.GetExperiments(ctx)
	if err != nil {
		logger.Error("Ошибка получения экспериментов: %v", err)
		dialog.ShowError(err, mw.window)
		return
	}

	// создание таблицы для отображения экспериментов
	data := make([][]string, 0)
	for _, exp := range experiments {
		data = append(data, []string{
			fmt.Sprintf("%d", exp.ID),
			exp.Name,
			exp.AlgorithmA,
			exp.AlgorithmB,
			fmt.Sprintf("%d", exp.UserPercent),
			exp.StartDate.Format("2006-01-02"),
			fmt.Sprintf("%t", exp.IsActive),
			strings.Join(exp.Tags, ", "),
		})
	}

	table := widget.NewTable(
		func() (int, int) {
			return len(data), 8
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("template")
		},
		func(i widget.TableCellID, o fyne.CanvasObject) {
			o.(*widget.Label).SetText(data[i.Row][i.Col])
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

	closeBtn := widget.NewButton("Закрыть", func() {
		dataWin.Close()
	})

	// создание контейнера с таблицей и кнопкой
	content := container.NewBorder(nil, closeBtn, nil, nil, table)
	dataWin.SetContent(content)
	dataWin.Show()
}
