package ui

import (
	"testing-platform/db"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

type MainWindow struct {
	app           fyne.App
	window        fyne.Window
	rep           *db.Repository
	titleLabel    *widget.Label
	subtitleLabel *widget.Label
}

func NewMainWindow(app fyne.App, rep *db.Repository) *MainWindow {
	window := app.NewWindow("Testing Platform")
	window.SetFixedSize(false)
	window.Resize(fyne.NewSize(900, 600))

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
	showSummaryBtn := widget.NewButton("Сводные данные", mw.showSummaryWindow)

	mw.titleLabel = widget.NewLabel("А/В Testing Platform")
	mw.titleLabel.Alignment = fyne.TextAlignCenter
	mw.titleLabel.TextStyle = fyne.TextStyle{Bold: true}
	mw.titleLabel.Wrapping = fyne.TextWrapOff // Отключаем перенос

	mw.subtitleLabel = widget.NewLabel("Управление экспериментами А/В тестирования")
	mw.subtitleLabel.Alignment = fyne.TextAlignCenter
	mw.subtitleLabel.Wrapping = fyne.TextWrapOff // Отключаем перенос

	// Создаем контейнер для кнопок
	buttonsContainer := container.NewVBox(
		layout.NewSpacer(),
		createSchemaBtn,
		addDataBtn,
		showDataBtn,
		showSummaryBtn,
		layout.NewSpacer(),
	)

	// Основной контент
	mainContent := container.NewVBox(
		container.NewCenter(mw.titleLabel),
		container.NewCenter(mw.subtitleLabel),
		widget.NewSeparator(),
		buttonsContainer,
	)

	// Добавляем отступы и возможность прокрутки
	paddedContent := container.NewBorder(
		layout.NewSpacer(), layout.NewSpacer(),
		layout.NewSpacer(), layout.NewSpacer(),
		mainContent,
	)

	mw.window.SetContent(container.NewVScroll(paddedContent))

	// Обработчик изменения размера окна
	mw.window.Canvas().SetOnTypedKey(func(e *fyne.KeyEvent) {
		if e.Name == fyne.KeyR {
			mw.updateLayout()
		}
	})
}

func (mw *MainWindow) updateLayout() {
	size := mw.window.Canvas().Size()

	// Адаптируем текст для разных размеров экрана
	if size.Width < 400 {
		mw.titleLabel.SetText("А/В Testing")
	} else {
		mw.titleLabel.SetText("А/В Testing Platform")
	}
}

func (mw *MainWindow) Show() {
	mw.window.Show()
	mw.ShowInstructionDialog()
}

// показ диалога с инструкцией
func (mw *MainWindow) ShowInstructionDialog() {
	instructionText := `Перед внесением данных ознакомьтесь с правилами:

    Эксперимент:
    - Название: обязательно, не длиннее 255 символов
    - Алгоритмы: должны быть разными
    - Процент пользователей: число от 0.1 до 100
    - Теги: через запятую, каждый тег не длинее 50 символов

    Пользователь:
    - ID эксперимента: целое положительное число
    - ID пользователя: обязательно, не длинее 255 символов (только буквы, цифры, дефисы и подчеркивания)
    - Группа: A или B

    Результат:
    - ID пользователя: целое положительное число
    - ID рекомендации: обязательно, не длинее 255 символов (только буквы, цифры, дефисы и подчеркивания)
    - Рейтинг: целое число от 0 до 5 (обязательно при клике)`

	text := widget.NewLabel(instructionText)
	text.Wrapping = fyne.TextWrapWord
	scroll := container.NewScroll(text)
	scroll.SetMinSize(fyne.NewSize(500, 300))

	dialog.ShowCustom("Инструкция по внесению данных", "Понятно", scroll, mw.window)
}
