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
	// Существующие кнопки
	createSchemaBtn := widget.NewButton("Создать схему и таблицы", mw.createSchemaHandler)
	addDataBtn := widget.NewButton("Внести данные", mw.showDataInputDialog)
	showDataBtn := widget.NewButton("Показать данные", mw.showDataDisplayWindow)
	showSummaryBtn := widget.NewButton("Сводные данные", mw.showSummaryWindow)

	// Новые кнопки для быстрого доступа к функциям БД
	alterTableBtn := widget.NewButton("ALTER TABLE", mw.showAlterTable)
	advancedQueryBtn := widget.NewButton("Расширенный SELECT", mw.showAdvancedQuery)
	joinBuilderBtn := widget.NewButton("Мастер JOIN", mw.showJoinBuilder)
	textSearchBtn := widget.NewButton("Текстовый поиск", mw.showTextSearch)
	stringFunctionsBtn := widget.NewButton("Функции работы со строками", mw.showStringFunctions)

	mw.titleLabel = widget.NewLabel("А/В Testing Platform")
	mw.titleLabel.Alignment = fyne.TextAlignCenter
	mw.titleLabel.TextStyle = fyne.TextStyle{Bold: true}
	mw.titleLabel.Wrapping = fyne.TextWrapOff

	mw.subtitleLabel = widget.NewLabel("Управление экспериментами А/В тестирования")
	mw.subtitleLabel.Alignment = fyne.TextAlignCenter
	mw.subtitleLabel.Wrapping = fyne.TextWrapOff

	leftColumn := container.NewVBox(
		widget.NewLabelWithStyle("Основные операции", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		createSchemaBtn,
		addDataBtn,
		showDataBtn,
		showSummaryBtn,
	)

	rightColumn := container.NewVBox(
		widget.NewLabelWithStyle("Функции БД", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		alterTableBtn,
		advancedQueryBtn,
		joinBuilderBtn,
		textSearchBtn,
		stringFunctionsBtn,
	)

	// Адаптивный контейнер - на маленьких экранах вертикально, на больших горизонтально
	adaptiveContainer := container.NewAdaptiveGrid(2, leftColumn, rightColumn)

	// Центрируем
	centeredContainer := container.NewCenter(adaptiveContainer)

	// Основной контент
	mainContent := container.NewVBox(
		container.NewCenter(mw.titleLabel),
		container.NewCenter(mw.subtitleLabel),
		widget.NewSeparator(),
		centeredContainer,
	)

	// Создаем меню
	mw.window.SetMainMenu(mw.createMenu())

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

// Создание меню
func (mw *MainWindow) createMenu() *fyne.MainMenu {
	return fyne.NewMainMenu(
		fyne.NewMenu("Файл",
			fyne.NewMenuItem("Создать схему", mw.createSchemaHandler),
			fyne.NewMenuItem("Выход", func() {
				mw.app.Quit()
			}),
		),
		fyne.NewMenu("Данные",
			fyne.NewMenuItem("Внести данные", mw.showDataInputDialog),
			fyne.NewMenuItem("Показать данные", mw.showDataDisplayWindow),
			fyne.NewMenuItem("Сводные данные", mw.showSummaryWindow),
		),
		fyne.NewMenu("База данных",
			fyne.NewMenuItem("ALTER TABLE", mw.showAlterTable),
			fyne.NewMenuItem("Расширенный SELECT", mw.showAdvancedQuery),
			fyne.NewMenuItem("Мастер JOIN", mw.showJoinBuilder),
		),
		fyne.NewMenu("Функции",
			fyne.NewMenuItem("Текстовый поиск", mw.showTextSearch),
			fyne.NewMenuItem("Работа со строками", mw.showStringFunctions),
		),
		fyne.NewMenu("Справка",
			fyne.NewMenuItem("Инструкция", mw.ShowInstructionDialog),
		),
	)
}

// Методы для открытия новых окон
func (mw *MainWindow) showAlterTable() {
	alterWin := NewAlterTableWindow(mw.rep, mw.window)
	alterWin.Show()
}

func (mw *MainWindow) showAdvancedQuery() {
	queryWin := NewAdvancedQueryWindow(mw.rep, mw.window)
	queryWin.Show()
}

func (mw *MainWindow) showJoinBuilder() {
	joinWin := NewJoinBuilderWindow(mw.rep, mw.window)
	joinWin.Show()
}

func (mw *MainWindow) showTextSearch() {
	searchWin := NewTextSearchWindow(mw.rep, mw.window)
	searchWin.Show()
}

func (mw *MainWindow) showStringFunctions() {
	stringWin := NewStringFunctionsWindow(mw.rep, mw.window)
	stringWin.Show()
}

func (mw *MainWindow) updateLayout() {
	size := mw.window.Canvas().Size()

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

// Показ диалога с инструкцией
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
    - Рейтинг: целое число от 0 до 5 (обязательно при клике)

    Новые функции БД:
    - ALTER TABLE: изменение структуры таблиц
    - Расширенный SELECT: сложные запросы с фильтрацией
    - Мастер JOIN: визуальное построение соединений
    - Текстовый поиск: поиск по шаблонам и регулярным выражениям
    - Функции работы со строками: преобразование текстовых данных`

	text := widget.NewLabel(instructionText)
	text.Wrapping = fyne.TextWrapWord
	scroll := container.NewScroll(text)
	scroll.SetMinSize(fyne.NewSize(500, 300))

	dialog.ShowCustom("Инструкция по внесению данных", "Понятно", scroll, mw.window)
}
