package ui

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing-platform/db"
	"testing-platform/db/models"
	"testing-platform/pkg/logger"
	"time"

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
	customTypesBtn := widget.NewButton("Пользовательские типы", mw.showCustomTypes)
	subqueryBtn := widget.NewButton("Подзапросы", mw.showSubqueryBuilder)

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
		customTypesBtn, // НОВАЯ КНОПКА
		subqueryBtn,    // НОВАЯ КНОПКА
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

// В методе showAlterTable обновите callback
func (mw *MainWindow) showAlterTable() {
	logger.Info("Открытие окна ALTER TABLE")

	alterWin := NewAlterTableWindow(mw.rep, mw.window, func() {
		logger.Info("=== CALLBACK: Обновление после ALTER TABLE ===")

		// 1. Принудительно обновляем структуры всех таблиц
		ctx := context.Background()
		logger.Info("Принудительное обновление структур таблиц...")

		if err := mw.rep.RefreshAllTableSchemas(ctx); err != nil {
			logger.Error("Ошибка обновления структур таблиц: %v", err)
			mw.showErrorMessage(fmt.Sprintf("Ошибка обновления структур таблиц: %v", err))
		} else {
			logger.Info("Структуры таблиц успешно обновлены")
		}

		// 2. Обновляем все открытые окна данных
		logger.Info("Обновление всех окон данных...")
		mw.RefreshAllWindows()

		// 3. Дополнительная гарантия - обновление с задержкой
		go func() {
			time.Sleep(500 * time.Millisecond)
			logger.Info("Дополнительное обновление через 500ms")
			mw.RefreshAllWindows()
		}()

		logger.Info("=== CALLBACK ЗАВЕРШЕН ===")
	})
	alterWin.Show()
}

// В методе RefreshAllWindows добавьте обновление списка таблиц
func (mw *MainWindow) RefreshAllWindows() {
	logger.Info("Обновление всех окон приложения")

	// Принудительно обновляем кэш таблиц
	ctx := context.Background()
	if err := mw.rep.RefreshAllTableSchemas(ctx); err != nil {
		logger.Error("Ошибка обновления кэша таблиц: %v", err)
	}

	mw.NotifyAllDataWindows()
	mw.NotifyAllSummaryWindows()
}

// Добавьте метод для показа ошибок
func (mw *MainWindow) showErrorMessage(message string) {
	dialog.ShowError(fmt.Errorf(message), mw.window)
}

func (mw *MainWindow) NotifyAllDataWindows() {
	mw.dataMutex.Lock()
	defer mw.dataMutex.Unlock()

	logger.Info("Уведомление %d окон данных об обновлении. Главное окно: %p", len(mw.dataWindows), mw)

	for i, dw := range mw.dataWindows {
		logger.Info("  Обновление окна %d: %p", i, dw)
		if dw != nil {
			dw.RefreshData()
		} else {
			logger.Error("  Окно %d равно nil!", i)
		}
	}
}

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

// Вспомогательная функция для конвертации значений в строку
func convertValueToStringUniversal(value interface{}) string {
	if value == nil {
		return "NULL"
	}

	switch v := value.(type) {
	case string:
		return v
	case int, int8, int16, int32, int64:
		return fmt.Sprintf("%d", v)
	case uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", v)
	case float32:
		return fmt.Sprintf("%.2f", v)
	case float64:
		return fmt.Sprintf("%.2f", v)
	case bool:
		if v {
			return "true"
		}
		return "false"
	case []string:
		return strings.Join(v, ", ")
	case []byte:
		return string(v)
	case time.Time:
		return v.Format("2006-01-02 15:04:05")
	default:
		// Для неизвестных типов
		str := fmt.Sprintf("%v", v)
		return str
	}
}

// Функция для отображения табличных данных
func displayTableData(container *fyne.Container, result *models.QueryResult, tableName string) {
	if len(result.Rows) == 0 {
		// Создаем пустую таблицу с сообщением
		table := widget.NewTable(
			func() (int, int) { return 1, 1 },
			func() fyne.CanvasObject {
				label := widget.NewLabel("")
				label.Wrapping = fyne.TextWrapWord
				return label
			},
			func(i widget.TableCellID, o fyne.CanvasObject) {
				label := o.(*widget.Label)
				if i.Row == 0 && i.Col == 0 {
					label.SetText("Таблица results пуста")
				}
			})
		container.Objects = []fyne.CanvasObject{table}
		container.Refresh()
		return
	}

	// Создаем таблицу с динамическими столбцами
	table := widget.NewTable(
		func() (int, int) {
			return len(result.Rows) + 1, len(result.Columns) // +1 для заголовков
		},
		func() fyne.CanvasObject {
			label := widget.NewLabel("")
			label.Wrapping = fyne.TextWrapWord
			return label
		},
		func(i widget.TableCellID, o fyne.CanvasObject) {
			label := o.(*widget.Label)
			label.Wrapping = fyne.TextWrapWord

			// Первая строка - заголовки
			if i.Row == 0 {
				if i.Col < len(result.Columns) {
					columnName := result.Columns[i.Col]
					// Улучшаем отображение названий столбцов
					switch columnName {
					case "user_id":
						columnName = "ID пользователя"
					case "recommendation_id":
						columnName = "ID рекомендации"
					case "clicked":
						columnName = "Кликнут"
					case "clicked_at":
						columnName = "Время клика"
					case "rating":
						columnName = "Рейтинг"
					}
					label.SetText(columnName)
					label.TextStyle = fyne.TextStyle{Bold: true}
				}
			} else {
				// Данные
				rowIndex := i.Row - 1
				if rowIndex < len(result.Rows) && i.Col < len(result.Columns) {
					value := result.Rows[rowIndex][result.Columns[i.Col]]
					text := convertValueToStringUniversal(value)

					// Специальная обработка для boolean значений
					if result.Columns[i.Col] == "clicked" {
						if text == "true" {
							text = "Да"
						} else if text == "false" {
							text = "Нет"
						}
					}

					label.SetText(text)
					label.TextStyle = fyne.TextStyle{}
				}
			}
		})

	// Автоматическая настройка ширины столбцов
	for col := 0; col < len(result.Columns); col++ {
		maxWidth := float32(150) // минимальная ширина

		// Учитываем заголовок
		headerWidth := float32(len(result.Columns[col])) * 8
		if headerWidth > maxWidth {
			maxWidth = headerWidth
		}

		// Учитываем данные (первые 20 строк)
		for row := 0; row < len(result.Rows) && row < 20; row++ {
			if value := result.Rows[row][result.Columns[col]]; value != nil {
				text := convertValueToStringUniversal(value)
				textWidth := float32(len(text)) * 7
				if textWidth > maxWidth {
					maxWidth = textWidth
				}
			}
		}

		// Ограничение максимальной ширины
		if maxWidth > 500 {
			maxWidth = 500
		}

		table.SetColumnWidth(col, maxWidth)
	}

	container.Objects = []fyne.CanvasObject{table}
	container.Refresh()
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
		fyne.NewMenu("Расширенные функции",
			fyne.NewMenuItem("Пользовательские типы", mw.showCustomTypes),
			fyne.NewMenuItem("Построитель подзапросов", mw.showSubqueryBuilder),
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
	logger.Info("Открытие окна результатов экспериментов")

	// Создаем новое окно для отображения таблицы results
	resultsWin := mw.app.NewWindow("Результаты экспериментов")
	resultsWin.Resize(fyne.NewSize(1200, 700))

	// Добавляем окно в список отслеживаемых
	mw.addSummaryWindow(resultsWin)

	// Устанавливаем обработчик закрытия окна
	resultsWin.SetOnClosed(func() {
		mw.removeSummaryWindow(resultsWin)
	})

	// Создаем контейнер для таблицы
	tableContainer := container.NewStack()
	resultLabel := widget.NewLabel("Загрузка данных...")
	resultLabel.Wrapping = fyne.TextWrapWord

	// Функция для загрузки и отображения данных
	loadResultsData := func() {
		resultLabel.SetText("Загрузка данных из таблицы results...")

		ctx := context.Background()
		result, err := mw.rep.GetTableData(ctx, "results")

		if err != nil {
			resultLabel.SetText("Ошибка загрузки: " + err.Error())
			return
		}

		if result.Error != "" {
			resultLabel.SetText("Ошибка БД: " + result.Error)
			return
		}

		// Отображаем таблицу
		displayTableData(tableContainer, result, "results")
		resultLabel.SetText(fmt.Sprintf("Таблица 'results': %d строк, %d столбцов", len(result.Rows), len(result.Columns)))
	}

	// Кнопки управления
	refreshBtn := widget.NewButton("Обновить", func() {
		loadResultsData()
	})

	closeBtn := widget.NewButton("Закрыть", func() {
		resultsWin.Close()
	})

	// Панель управления
	controlPanel := container.NewHBox(
		refreshBtn,
		closeBtn,
	)

	// Основной контент
	content := container.NewBorder(
		container.NewVBox(resultLabel, controlPanel),
		nil, nil, nil,
		container.NewScroll(tableContainer),
	)

	resultsWin.SetContent(content)

	// Первоначальная загрузка данных
	loadResultsData()

	resultsWin.Show()
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
    - Рейтинг: целое число от 0 до 5 (обязательно при клике)`

	text := widget.NewLabel(instructionText)
	text.Wrapping = fyne.TextWrapWord
	scroll := container.NewScroll(text)
	scroll.SetMinSize(fyne.NewSize(500, 300))

	dialog.ShowCustom("Инструкция по внесению данных", "Понятно", scroll, mw.window)
}
func (mw *MainWindow) showCustomTypes() {
	customTypesWin := NewCustomTypesWindow(mw.rep, mw.app)
	customTypesWin.Show()
}

func (mw *MainWindow) showSubqueryBuilder() {
	// Можно открыть общий построитель или показать сообщение
	dialog.ShowInformation("Подзапросы",
		"Функция подзапросов доступна в окне просмотра данных через кнопку 'Расширенный фильтр'",
		mw.window)
}
