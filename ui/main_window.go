package ui

import (
	"fyne.io/fyne/v2/app"
)

func main() {
	App := app.New()
	mainWindow := App.NewWindow("Hello Fyne!")

	mainWindow.ShowAndRun()
}
