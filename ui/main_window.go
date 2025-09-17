package ui

import (
	"testing-platform/db"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type MainWindow struct {
	app    fyne.App
	window fyne.Window
	rep    *db.Repository
}

func NewMainWindow(app fyne.App, rep *db.Repository) *MainWindow {
	window := app.NewWindow("Testing Platform")
	window.Resize(fyne.NewSize(600, 400))

	return &MainWindow{
		app:    app,
		window: window,
		rep:    rep,
	}
}

func (mw *MainWindow) CreateUI() {
	createSchemaBtn := widget.NewButton("Создать схему и таблицы", mw.createSchemaHandler)
	addDataBtn := widget.NewButton("Внести данные", mw.showDataInputDialog)
	showDataBtn := widget.NewButton("Показать данные", mw.showDataDisplayWindow)

	content := container.NewVBox(
		widget.NewLabel("А/В Testing Platform для рекомендательных систем"),
		widget.NewLabel("Управление экспериментами А/В тестирования"),
		createSchemaBtn,
		addDataBtn,
		showDataBtn,
	)
	mw.window.SetContent(content)
}

func (mw *MainWindow) Show() {
	mw.window.Show()
}
