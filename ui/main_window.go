package ui

import (
	"context"
	"fmt"
	"sync"
	"testing-platform/db"
	"testing-platform/pkg/logger"

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

	// Для отслеживания открытых окон данных
	dataWindows []*DataDisplayWindow
	dataMutex   sync.Mutex

	// Для отслеживания открытых сводных окон
	summaryWindows []fyne.Window
	summaryMutex   sync.Mutex
}

func NewMainWindow(app fyne.App, rep *db.Repository) *MainWindow {
	window := app.NewWindow("Testing Platform")
	window.SetFixedSize(false)
	window.Resize(fyne.NewSize(900, 600))

	return &MainWindow{
		app:            app,
		window:         window,
		rep:            rep,
		dataWindows:    make([]*DataDisplayWindow, 0),
		summaryWindows: make([]fyne.Window, 0),
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

	// rightColumn := container.NewVBox(
	// 	widget.NewLabelWithStyle("Функции БД", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
	// 	alterTableBtn,
	// 	advancedQueryBtn,
	// 	joinBuilderBtn,
	// 	textSearchBtn,
	// 	stringFunctionsBtn,
	// )

	// Временная кнопка для отладки
	debugBtn := widget.NewButton("Отладка: Обновить все окна", func() {
		logger.Info("Принудительное обновление всех окон")
		mw.RefreshAllWindows()
	})

	// Добавьте эту кнопку в интерфейс, например:
	rightColumn := container.NewVBox(
		widget.NewLabelWithStyle("Функции БД", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		alterTableBtn,
		advancedQueryBtn,
		joinBuilderBtn,
		textSearchBtn,
		stringFunctionsBtn,
		widget.NewSeparator(),
		debugBtn, // Добавляем кнопку отладки
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

// Добавляем методы для управления окнами данных
func (mw *MainWindow) addDataWindow(dw *DataDisplayWindow) {
	mw.dataMutex.Lock()
	defer mw.dataMutex.Unlock()
	logger.Info("Добавление окна данных в главное окно %p. Теперь окон: %d", mw, len(mw.dataWindows)+1)
	mw.dataWindows = append(mw.dataWindows, dw)

	// Отладочная информация
	logger.Info("Содержимое mw.dataWindows:")
	for i, window := range mw.dataWindows {
		logger.Info("  Окно %d: %p", i, window)
	}
}

// NotifyAllDataWindows уведомляет все открытые окна данных об изменениях
func (mw *MainWindow) NotifyAllDataWindows() {
	mw.dataMutex.Lock()
	defer mw.dataMutex.Unlock()

	logger.Info("Уведомление %d окон данных об обновлении. Главное окно: %p", len(mw.dataWindows), mw)

	for i, dw := range mw.dataWindows {
		logger.Info("  Обновление окна %d: %p", i, dw)
		dw.RefreshData()
	}
}

// // Добавляем методы для управления окнами данных
// func (mw *MainWindow) addDataWindow(dw *DataDisplayWindow) {
// 	mw.dataMutex.Lock()
// 	defer mw.dataMutex.Unlock()
// 	logger.Info("Добавление окна данных в главное окно. Теперь окон: %d", len(mw.dataWindows)+1)
// 	mw.dataWindows = append(mw.dataWindows, dw)
// }

func (mw *MainWindow) removeDataWindow(dw *DataDisplayWindow) {
	mw.dataMutex.Lock()
	defer mw.dataMutex.Unlock()
	for i, w := range mw.dataWindows {
		if w == dw {
			mw.dataWindows = append(mw.dataWindows[:i], mw.dataWindows[i+1:]...)
			logger.Info("Удаление окна данных из главного окна. Теперь окон: %d", len(mw.dataWindows))
			break
		}
	}
}

// Добавляем методы для управления сводными окнами
func (mw *MainWindow) addSummaryWindow(sw fyne.Window) {
	mw.summaryMutex.Lock()
	defer mw.summaryMutex.Unlock()
	logger.Info("Добавление сводного окна в главное окно. Теперь окон: %d", len(mw.summaryWindows)+1)
	mw.summaryWindows = append(mw.summaryWindows, sw)
}

func (mw *MainWindow) removeSummaryWindow(sw fyne.Window) {
	mw.summaryMutex.Lock()
	defer mw.summaryMutex.Unlock()
	for i, w := range mw.summaryWindows {
		if w == sw {
			mw.summaryWindows = append(mw.summaryWindows[:i], mw.summaryWindows[i+1:]...)
			logger.Info("Удаление сводного окна из главного окна. Теперь окон: %d", len(mw.summaryWindows))
			break
		}
	}
}

// // NotifyAllDataWindows уведомляет все открытые окна данных об изменениях
// func (mw *MainWindow) NotifyAllDataWindows() {
// 	mw.dataMutex.Lock()
// 	defer mw.dataMutex.Unlock()

// 	logger.Info("Уведомление %d окон данных об обновлении", len(mw.dataWindows))
// 	for _, dw := range mw.dataWindows {
// 		dw.RefreshData()
// 	}
// }

// NotifyAllSummaryWindows уведомляет все открытые сводные окна об изменениях
func (mw *MainWindow) NotifyAllSummaryWindows() {
	mw.summaryMutex.Lock()
	defer mw.summaryMutex.Unlock()

	logger.Info("Закрытие %d сводных окон", len(mw.summaryWindows))
	// Закрываем сводные окна
	for _, sw := range mw.summaryWindows {
		sw.Close()
	}
	mw.summaryWindows = make([]fyne.Window, 0)
}

// RefreshAllWindows обновляет все открытые окна
func (mw *MainWindow) RefreshAllWindows() {
	logger.Info("Обновление всех окон приложения")
	mw.NotifyAllDataWindows()
	mw.NotifyAllSummaryWindows()
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

// Методы для открытия окон данных
func (mw *MainWindow) showDataDisplayWindow() {
	logger.Info("Открытие нового окна данных")
	dataWin := NewDataDisplayWindow(mw)
	dataWin.Show()
}

func (mw *MainWindow) showSummaryWindow() {
	logger.Info("Открытие нового сводного окна")
	// создание нового окна
	summaryWin := mw.app.NewWindow("Сводные данные экспериментов")
	summaryWin.Resize(fyne.NewSize(1000, 600))

	// Добавляем окно в список отслеживаемых
	mw.addSummaryWindow(summaryWin)

	// Устанавливаем обработчик закрытия окна
	summaryWin.SetOnClosed(func() {
		mw.removeSummaryWindow(summaryWin)
	})

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

// // Обновленный метод showAlterTable
// func (mw *MainWindow) showAlterTable() {
// 	logger.Info("Открытие окна ALTER TABLE")
// 	alterWin := NewAlterTableWindow(mw.rep, mw.window, func() {
// 		// Callback для обновления главного окна после изменений в таблице
// 		mw.showSuccessMessage("Структура таблицы успешно изменена")

// 		// Обновляем все открытые окна данных
// 		mw.RefreshAllWindows()
// 	})
// 	alterWin.Show()
// }

// Обновленный метод showAlterTable
func (mw *MainWindow) showAlterTable() {
	logger.Info("Открытие окна ALTER TABLE. Главное окно: %p", mw)

	alterWin := NewAlterTableWindow(mw.rep, mw.window, func() {
		logger.Info("Callback вызван! Главное окно: %p", mw)
		// Callback для обновления главного окна после изменений в таблице
		mw.showSuccessMessage("Структура таблицы успешно изменена")

		// Обновляем все открытые окна данных
		logger.Info("Вызов RefreshAllWindows из callback")
		mw.RefreshAllWindows()
	})
	alterWin.Show()
}

// Вспомогательный метод для показа сообщения об успехе
func (mw *MainWindow) showSuccessMessage(message string) {
	infoDialog := dialog.NewInformation("Успех", message, mw.window)
	infoDialog.Show()
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

// Заглушки для методов, которые должны быть реализованы
// func (mw *MainWindow) createSchemaHandler() {
// 	ctx := context.Background()
// 	err := mw.rep.CreateSchema(ctx)
// 	if err != nil {
// 		dialog.ShowError(err, mw.window)
// 	} else {
// 		dialog.ShowInformation("Успех", "Схема базы данных успешно создана", mw.window)
// 	}
// }

// func (mw *MainWindow) showDataInputDialog() {
// 	// Здесь должна быть реализация диалога ввода данных
// 	dialog.ShowInformation("В разработке", "Диалог ввода данных находится в разработке", mw.window)
// }

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
