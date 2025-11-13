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
	"github.com/jackc/pgx/v5/pgtype"
)

// DataDisplayWindow представляет окно отображения данных
type DataDisplayWindow struct {
	window         fyne.Window
	mainWindow     *MainWindow
	tableContainer *fyne.Container
	filterPanel    *fyne.Container
	currentFilter  models.ExperimentFilter // Сохраняем текущий фильтр
	tableName      string                  // храним имя таблицы

	// НОВЫЕ ПОЛЯ ДЛЯ ПОДЗАПРОСОВ
	subqueryCondition *models.SubqueryCondition
	subqueryBtn       *widget.Button
	clearSubqueryBtn  *widget.Button
	subqueryLabel     *widget.Label
}

// NewDataDisplayWindow создает новое окно отображения данных
func NewDataDisplayWindow(mw *MainWindow) *DataDisplayWindow {
	logger.Info("Создание нового окна данных. Полученное главное окно: %p", mw)

	d := &DataDisplayWindow{
		mainWindow: mw,
		window:     mw.app.NewWindow("Данные экспериментов"),
		tableName:  "experiments", // Устанавливаем таблицу по умолчанию
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

	// Выбор таблицы
	tableSelect := widget.NewSelect([]string{}, func(selected string) {
		if selected != "" {
			d.tableName = selected
			d.clearSubquery() // Очищаем подзапрос при смене таблицы
			d.updateTable(d.currentFilter)
			d.window.SetTitle("Данные таблицы: " + selected)
		}
	})
	tableSelect.PlaceHolder = "Выберите таблицу"

	// Загрузка списка таблиц
	d.loadTableList(tableSelect)

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

	// НОВЫЕ КНОПКИ ДЛЯ ПОДЗАПРОСОВ
	d.subqueryBtn = widget.NewButton("Расширенный фильтр (подзапрос)", d.showSubqueryBuilder)
	d.clearSubqueryBtn = widget.NewButton("Очистить подзапрос", d.clearSubquery)
	d.clearSubqueryBtn.Disable() // Изначально отключена, пока нет подзапроса

	d.subqueryLabel = widget.NewLabel("Подзапрос не применен")
	d.subqueryLabel.Wrapping = fyne.TextWrapWord

	// Кнопка обновления списка таблиц
	refreshTablesBtn := widget.NewButton("Обновить список таблиц", func() {
		d.loadTableList(tableSelect)
	})

	// Создаем панель фильтров с новыми элементами
	d.filterPanel = container.NewVBox(
		container.NewHBox(
			widget.NewLabel("Таблица:"),
			tableSelect,
			refreshTablesBtn,
		),
		widget.NewSeparator(),
		widget.NewLabel("Базовые фильтры:"),
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
		widget.NewSeparator(),
		// НОВАЯ СЕКЦИЯ: ПОДЗАПРОСЫ
		widget.NewLabelWithStyle("Расширенные фильтры (подзапросы)", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		d.subqueryLabel,
		container.NewHBox(
			d.subqueryBtn,
			d.clearSubqueryBtn,
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

	// ИСПРАВЛЕНИЕ: Используем VSplit для разделения панели фильтров и таблицы
	split := container.NewVSplit(
		container.NewScroll(d.filterPanel), // Верхняя часть - панель фильтров с прокруткой
		d.tableContainer,                   // Нижняя часть - таблица
	)
	split.SetOffset(0.3) // Устанавливаем начальное разделение (30% для фильтров, 70% для таблицы)

	// создание основного контейнера с исправленным макетом
	content := container.NewBorder(
		nil,                         // верхняя панель (теперь в split)
		container.NewHBox(closeBtn), // нижняя панель с кнопкой закрытия
		nil, nil,
		split, // центральная область с разделителем
	)

	d.window.SetContent(content)
}

// НОВЫЕ МЕТОДЫ ДЛЯ ПОДЗАПРОСОВ

// showSubqueryBuilder открывает окно построителя подзапросов
func (d *DataDisplayWindow) showSubqueryBuilder() {
	logger.Info("Открытие построителя подзапросов для таблицы: %s", d.tableName)

	builder := NewSubqueryBuilder(d.mainWindow.rep, d.mainWindow.app, func(condition *models.SubqueryCondition) {
		if condition != nil {
			d.subqueryCondition = condition
			d.applySubqueryFilter()
			d.updateSubqueryUI()
		}
	})

	// Устанавливаем основную таблицу, если она выбрана
	if d.tableName != "" {
		builder.SetMainTable(d.tableName)
	}

	builder.Show()
}

// applySubqueryFilter применяет подзапрос к данным
func (d *DataDisplayWindow) applySubqueryFilter() {
	if d.subqueryCondition == nil {
		logger.Warn("Попытка применить пустой подзапрос")
		return
	}

	logger.Info("Применение подзапроса типа: %s", d.subqueryCondition.Type)

	// Показываем индикатор загрузки
	d.subqueryLabel.SetText("Выполняется подзапрос...")
	d.tableContainer.Objects = []fyne.CanvasObject{
		widget.NewLabel("Выполнение подзапроса..."),
	}
	d.tableContainer.Refresh()

	ctx := context.Background()
	result, err := d.mainWindow.rep.ExecuteQuery(ctx, d.subqueryCondition.Subquery)
	if err != nil {
		errorMsg := fmt.Sprintf("Ошибка выполнения подзапроса: %v", err)
		logger.Error("%s", errorMsg)
		dialog.ShowError(fmt.Errorf("%s", errorMsg), d.window)
		d.subqueryLabel.SetText("❌ Ошибка выполнения подзапроса")
		return
	}

	// ДОБАВЛЯЕМ ПРОВЕРКУ НА NIL РЕЗУЛЬТАТ
	if result == nil {
		errorMsg := "Запрос не вернул результатов (nil result)"
		logger.Error("%s", errorMsg)
		dialog.ShowError(fmt.Errorf("%s", errorMsg), d.window)
		d.subqueryLabel.SetText("❌ Запрос не вернул результатов")
		return
	}

	if result.Error != "" {
		errorMsg := fmt.Sprintf("Ошибка БД в подзапросе: %s", result.Error)
		logger.Error("%s", errorMsg)
		dialog.ShowError(fmt.Errorf("%s", errorMsg), d.window)
		d.subqueryLabel.SetText("❌ Ошибка БД в подзапросе")
		return
	}

	// Отображаем результаты
	d.createDynamicTable(result)

	// Обновляем информацию о подзапросе
	d.updateSubqueryInfo(result)

	logger.Info("Подзапрос успешно применен. Найдено строк: %d", len(result.Rows))
}

// updateSubqueryInfo обновляет информацию о примененном подзапросе
func (d *DataDisplayWindow) updateSubqueryInfo(result *models.QueryResult) {
	if d.subqueryCondition == nil {
		return
	}

	info := fmt.Sprintf("✅ Применен подзапрос: %s\nНайдено строк: %d",
		d.subqueryCondition.Type, len(result.Rows))

	// Добавляем детали для разных типов подзапросов
	switch d.subqueryCondition.Type {
	case "EXISTS":
		info += fmt.Sprintf("\nEXISTS (SELECT ... FROM %s)",
			strings.Split(d.subqueryCondition.Subquery, "FROM ")[1])
	case "ANY", "ALL":
		info += fmt.Sprintf("\n%s %s %s (подзапрос)",
			d.subqueryCondition.MainColumn,
			d.subqueryCondition.Operator,
			d.subqueryCondition.Type)
	}

	d.subqueryLabel.SetText(info)
}

// updateSubqueryUI обновляет интерфейс в зависимости от состояния подзапроса
func (d *DataDisplayWindow) updateSubqueryUI() {
	if d.subqueryCondition != nil {
		d.clearSubqueryBtn.Enable()
		d.subqueryBtn.SetText("Изменить подзапрос")
	} else {
		d.clearSubqueryBtn.Disable()
		d.subqueryBtn.SetText("Расширенный фильтр (подзапрос)")
		d.subqueryLabel.SetText("Подзапрос не применен")
	}
}

// clearSubquery очищает примененный подзапрос
func (d *DataDisplayWindow) clearSubquery() {
	logger.Info("Очистка подзапроса")

	d.subqueryCondition = nil
	d.updateSubqueryUI()

	// Возвращаемся к обычному отображению таблицы
	d.updateTable(d.currentFilter)

	dialog.ShowInformation("Подзапрос очищен",
		"Примененный подзапрос был очищен. Отображаются все данные таблицы.",
		d.window)
}

// RefreshData принудительно обновляет данные в окне
func (d *DataDisplayWindow) RefreshData() {
	logger.Info("Обновление данных в окне отображения. Текущая таблица: %s", d.tableName)

	if d.subqueryCondition != nil {
		// Если есть активный подзапрос, переприменяем его
		d.applySubqueryFilter()
	} else {
		// Иначе обновляем обычные данные
		d.refreshDataSilent()
	}
}

// refreshDataSilent обновляет данные без показа диалога
func (d *DataDisplayWindow) refreshDataSilent() {
	if d.subqueryCondition != nil {
		d.applySubqueryFilter()
	} else {
		d.updateTable(d.currentFilter)
	}
}

// Обновляем существующий refreshData
func (d *DataDisplayWindow) refreshData() {
	d.refreshDataSilent()
	dialog.ShowInformation("Обновлено", "Данные успешно обновлены", d.window)
}

// ОСТАВШИЕСЯ СУЩЕСТВУЮЩИЕ МЕТОДЫ (без изменений)

func (d *DataDisplayWindow) loadTableList(tableSelect *widget.Select) {
	tables, err := d.mainWindow.rep.GetTableNames(context.Background())
	if err != nil {
		logger.Error("Ошибка загрузки списка таблиц: %v", err)
		dialog.ShowError(fmt.Errorf("не удалось загрузить список таблиц: %v", err), d.window)
		return
	}

	// Сохраняем текущее выделение
	currentSelection := tableSelect.Selected

	tableSelect.Options = tables
	tableSelect.Refresh()

	// Восстанавливаем выделение, если таблица существует
	if currentSelection != "" {
		for _, table := range tables {
			if table == currentSelection {
				tableSelect.SetSelected(currentSelection)
				d.tableName = currentSelection
				break
			}
		}
	} else if len(tables) > 0 && d.tableName == "" {
		// Устанавливаем первую таблицу по умолчанию
		tableSelect.SetSelected(tables[0])
		d.tableName = tables[0]
		d.window.SetTitle("Данные таблицы: " + d.tableName)
	}

	logger.Info("Список таблиц обновлен. Текущая таблица: %s", d.tableName)
}

func (d *DataDisplayWindow) updateTable(filter models.ExperimentFilter) {
	// Сохраняем текущий фильтр
	d.currentFilter = filter

	// Если есть активный подзапрос, используем его вместо обычного запроса
	if d.subqueryCondition != nil {
		d.applySubqueryFilter()
		return
	}

	ctx := context.Background()

	// Проверяем, что таблица существует
	tables, err := d.mainWindow.rep.GetTableNames(ctx)
	if err != nil {
		logger.Error("Ошибка получения списка таблиц: %v", err)
		dialog.ShowError(fmt.Errorf("не удалось получить список таблиц: %v", err), d.window)
		return
	}

	// Проверяем, существует ли текущая таблица
	tableExists := false
	for _, table := range tables {
		if table == d.tableName {
			tableExists = true
			break
		}
	}

	if !tableExists {
		logger.Warn("Таблица %s не найдена", d.tableName)
		d.tableContainer.Objects = []fyne.CanvasObject{
			widget.NewLabel(fmt.Sprintf("Таблица '%s' не найдена. Возможно, она была переименована или удалена.\n\nОбновите список таблиц.", d.tableName)),
		}
		d.tableContainer.Refresh()
		return
	}

	// Получаем динамические данные из выбранной таблицы
	query := fmt.Sprintf("SELECT * FROM %s", d.tableName)
	var args []interface{}

	// Применяем фильтры только для таблицы experiments
	if d.tableName == "experiments" && (filter.AlgorithmA != "" || filter.AlgorithmB != "" || filter.IsActive != nil || !filter.StartDateFrom.IsZero() || !filter.StartDateTo.IsZero()) {
		query += " WHERE 1=1"
		var conditions []string
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
	}

	query += " ORDER BY id DESC"

	// Выполняем запрос
	var result *models.QueryResult
	var queryErr error

	if len(args) > 0 {
		// Если есть параметры, используем метод с поддержкой параметризованных запросов
		result, queryErr = d.executeQueryWithParams(ctx, query, args...)
	} else {
		// Без параметров - простой запрос
		result, queryErr = d.mainWindow.rep.ExecuteQuery(ctx, query)
	}

	if queryErr != nil {
		logger.Error("Ошибка получения данных из таблицы %s: %v", d.tableName, queryErr)
		dialog.ShowError(fmt.Errorf("не удалось получить данные из таблицы %s: %v", d.tableName, queryErr), d.window)
		return
	}

	if result.Error != "" {
		logger.Error("Ошибка в результате запроса: %s", result.Error)
		dialog.ShowError(fmt.Errorf("ошибка выполнения запроса: %s", result.Error), d.window)
		return
	}

	d.createDynamicTable(result)
}

// convertValueToString конвертирует любое значение в строку для отображения
func convertValueToString(value interface{}) string {
	if value == nil {
		return ""
	}

	// Обработка pgtype.Numeric
	if numeric, ok := value.(pgtype.Numeric); ok {
		if !numeric.Valid {
			return ""
		}
		// Преобразуем Numeric в float64 и форматируем
		floatVal, err := numeric.Float64Value()
		if err != nil {
			return fmt.Sprintf("%v", numeric)
		}
		if !floatVal.Valid {
			return ""
		}
		return fmt.Sprintf("%.2f", floatVal.Float64)
	}

	switch v := value.(type) {
	case string:
		return v
	case int, int32, int64:
		return fmt.Sprintf("%d", v)
	case float32, float64:
		return fmt.Sprintf("%.2f", v)
	case bool:
		if v {
			return "Да"
		}
		return "Нет"
	case []string:
		return strings.Join(v, ", ")
	case []byte:
		return string(v)
	case time.Time:
		return v.Format("2006-01-02 15:04:05")
	default:
		// Для других типов пробуем преобразовать в строку
		str := fmt.Sprintf("%v", v)
		// Пытаемся распарсить как число, если похоже на числовое значение
		if strings.Contains(str, "Numeric") || strings.Contains(str, "pgtype") {
			// Это внутреннее представление pgtype, пропускаем
			return ""
		}
		return str
	}
}

// executeQueryWithParams выполняет параметризованный запрос
func (d *DataDisplayWindow) executeQueryWithParams(ctx context.Context, query string, args ...interface{}) (*models.QueryResult, error) {
	logger.Info("Выполнение параметризованного запроса: %s с параметрами: %v", query, args)

	// Начинаем транзакцию для безопасного выполнения
	tx, err := d.mainWindow.rep.Pool().Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("ошибка начала транзакции: %w", err)
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return &models.QueryResult{Error: err.Error()}, nil
	}
	defer rows.Close()

	// Получаем описание колонок
	fieldDescriptions := rows.FieldDescriptions()
	columns := make([]string, len(fieldDescriptions))
	for i, fd := range fieldDescriptions {
		columns[i] = string(fd.Name)
	}

	// Читаем данные
	var resultRows []map[string]interface{}
	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return &models.QueryResult{Error: err.Error()}, nil
		}

		row := make(map[string]interface{})
		for i, col := range columns {
			row[col] = values[i]
		}
		resultRows = append(resultRows, row)
	}

	// Коммитим транзакцию
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("ошибка коммита транзакции: %w", err)
	}

	return &models.QueryResult{
		Columns: columns,
		Rows:    resultRows,
	}, nil
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
					columnName := result.Columns[i.Col]
					// Улучшаем отображение названий столбцов
					if columnName == "user_percent" {
						columnName = "Процент пользователей (%)"
					} else if columnName == "algorithm_a" {
						columnName = "Алгоритм A"
					} else if columnName == "algorithm_b" {
						columnName = "Алгоритм B"
					} else if columnName == "is_active" {
						columnName = "Активен"
					} else if columnName == "start_date" {
						columnName = "Дата начала"
					}
					label.SetText(columnName)
					label.TextStyle = fyne.TextStyle{Bold: true}
				}
			} else {
				// Данные
				if i.Row-1 < len(data) && i.Col < len(data[i.Row-1]) {
					text := data[i.Row-1][i.Col]

					// Специальная обработка для boolean значений
					if result.Columns[i.Col] == "is_active" {
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

	// Настраиваем ширину столбцов в зависимости от содержания
	for i := 0; i < len(result.Columns); i++ {
		colName := result.Columns[i]
		switch colName {
		case "id":
			table.SetColumnWidth(i, 60)
		case "name":
			table.SetColumnWidth(i, 200)
		case "user_percent":
			table.SetColumnWidth(i, 250) // Шире для нового названия
		case "is_active":
			table.SetColumnWidth(i, 100)
		case "start_date":
			table.SetColumnWidth(i, 150)
		case "tags":
			table.SetColumnWidth(i, 200)
		default:
			table.SetColumnWidth(i, 120)
		}
	}

	// Обновление контейнера с таблицей
	d.tableContainer.Objects = []fyne.CanvasObject{table}
	d.tableContainer.Refresh()

	logger.Info("Таблица %s обновлена: %d строк, %d столбцов", d.tableName, len(data), len(result.Columns))
}

func (d *DataDisplayWindow) Show() {
	d.window.Show()
}
