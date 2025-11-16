package ui

import (
	"context"
	"fmt"
	"strings"
	"testing-platform/db"
	"testing-platform/pkg/logger"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

type AlterTableWindow struct {
	window         fyne.Window
	repository     *db.Repository
	mainWindow     fyne.Window
	onTableChanged func()

	// Элементы управления
	tableSelect     *widget.Select
	actionSelect    *widget.Select
	columnName      *widget.Entry
	newColumnName   *widget.Entry
	dataType        *widget.Select
	constraintType  *widget.Select
	constraintValue *widget.Entry
	defaultValue    *widget.Entry
	nullableCheck   *widget.Check
	referenceTable  *widget.Entry
	referenceColumn *widget.Entry
	resultLabel     *widget.Label

	// Контейнеры для группировки полей
	columnNameContainer      *fyne.Container
	newColumnNameContainer   *fyne.Container
	dataTypeContainer        *fyne.Container
	constraintTypeContainer  *fyne.Container
	constraintValueContainer *fyne.Container
	defaultValueContainer    *fyne.Container
	nullableCheckContainer   *fyne.Container
	referenceContainer       *fyne.Container

	currentTable      string
	standardDataTypes []string
	customTypeNames   map[string]bool // Для отслеживания пользовательских типов
}

func NewAlterTableWindow(repo *db.Repository, mainWindow fyne.Window, onTableChanged func()) *AlterTableWindow {
	a := &AlterTableWindow{
		repository:     repo,
		mainWindow:     mainWindow,
		onTableChanged: onTableChanged,
		window:         fyne.CurrentApp().NewWindow("ALTER TABLE - Изменение структуры таблиц"),
		standardDataTypes: []string{
			"INTEGER", "SERIAL", "BIGINT", "VARCHAR(255)", "TEXT", "BOOLEAN",
			"DATE", "TIMESTAMP", "NUMERIC(10,2)", "JSONB",
		},
		customTypeNames: make(map[string]bool),
	}

	a.buildUI()
	a.loadTables()
	a.loadCustomTypes()
	return a
}

func (a *AlterTableWindow) buildUI() {
	// Выбор таблицы
	a.tableSelect = widget.NewSelect([]string{}, a.onTableSelected)
	a.tableSelect.PlaceHolder = "Выберите таблицу"

	// Выбор действия
	a.actionSelect = widget.NewSelect([]string{
		"Добавить столбец",
		"Удалить столбец",
		"Переименовать столбец",
		"Изменить тип данных",
		"Добавить ограничение",
		"Удалить ограничение",
		"Переименовать таблицу",
	}, a.onActionSelected)
	a.actionSelect.PlaceHolder = "Выберите действие"

	// Поля ввода
	a.columnName = widget.NewEntry()
	a.columnName.SetPlaceHolder("Имя столбца")

	a.newColumnName = widget.NewEntry()
	a.newColumnName.SetPlaceHolder("Новое имя")

	// Инициализируем dataType с базовыми типами
	a.dataType = widget.NewSelect(a.standardDataTypes, nil)
	a.dataType.PlaceHolder = "Тип данных"

	a.constraintType = widget.NewSelect([]string{
		"PRIMARY KEY", "FOREIGN KEY", "UNIQUE", "CHECK", "NOT NULL",
	}, a.onConstraintTypeChanged)
	a.constraintType.PlaceHolder = "Тип ограничения"

	a.constraintValue = widget.NewEntry()
	a.constraintValue.SetPlaceHolder("Условие CHECK")

	a.defaultValue = widget.NewEntry()
	a.defaultValue.SetPlaceHolder("Значение по умолчанию")

	a.nullableCheck = widget.NewCheck("Разрешить NULL", nil)
	a.nullableCheck.SetChecked(true)

	a.referenceTable = widget.NewEntry()
	a.referenceTable.SetPlaceHolder("Таблица для ссылки")

	a.referenceColumn = widget.NewEntry()
	a.referenceColumn.SetPlaceHolder("Столбец для ссылки")

	a.resultLabel = widget.NewLabel("")
	a.resultLabel.Wrapping = fyne.TextWrapWord

	// Создаем контейнеры для группировки полей
	a.columnNameContainer = container.NewVBox(
		widget.NewLabel("Столбец:"),
		a.columnName,
	)

	a.newColumnNameContainer = container.NewVBox(
		widget.NewLabel("Новое имя:"),
		a.newColumnName,
	)

	a.dataTypeContainer = container.NewVBox(
		widget.NewLabel("Тип данных:"),
		a.dataType,
	)

	a.constraintTypeContainer = container.NewVBox(
		widget.NewLabel("Тип ограничения:"),
		a.constraintType,
	)

	a.constraintValueContainer = container.NewVBox(
		widget.NewLabel("Условие:"),
		a.constraintValue,
	)

	a.defaultValueContainer = container.NewVBox(
		widget.NewLabel("Значение по умолчанию:"),
		a.defaultValue,
	)

	a.nullableCheckContainer = container.NewVBox(
		a.nullableCheck,
	)

	a.referenceContainer = container.NewVBox(
		widget.NewLabel("Ссылка (FOREIGN KEY):"),
		a.referenceTable,
		a.referenceColumn,
	)

	// Кнопки
	applyBtn := widget.NewButton("Применить изменения", a.applyChanges)
	previewBtn := widget.NewButton("Показать SQL", a.showSQL)
	refreshBtn := widget.NewButton("Обновить", a.refreshData)
	closeBtn := widget.NewButton("Закрыть", func() { a.window.Close() })

	// Основная компоновка
	form := container.NewVBox(
		widget.NewLabel("Таблица:"),
		a.tableSelect,
		widget.NewLabel("Действие:"),
		a.actionSelect,
		a.columnNameContainer,
		a.newColumnNameContainer,
		a.dataTypeContainer,
		a.constraintTypeContainer,
		a.constraintValueContainer,
		a.defaultValueContainer,
		a.nullableCheckContainer,
		a.referenceContainer,
		container.NewHBox(applyBtn, previewBtn, refreshBtn, closeBtn),
		a.resultLabel,
	)

	a.window.SetContent(container.NewScroll(form))
	a.window.Resize(fyne.NewSize(600, 600))

	// Скрываем все дополнительные поля при запуске
	a.hideAllContainers()
}

func (a *AlterTableWindow) hideAllContainers() {
	a.columnNameContainer.Hide()
	a.newColumnNameContainer.Hide()
	a.dataTypeContainer.Hide()
	a.constraintTypeContainer.Hide()
	a.constraintValueContainer.Hide()
	a.defaultValueContainer.Hide()
	a.nullableCheckContainer.Hide()
	a.referenceContainer.Hide()
}

// Улучшенный метод для загрузки пользовательских типов
func (a *AlterTableWindow) loadCustomTypes() {
	ctx := context.Background()

	// Загружаем все пользовательские типы (ENUM и составные) одним запросом
	query := `
		SELECT 
			t.typname as type_name,
			t.typtype as type_category
		FROM pg_type t
		JOIN pg_catalog.pg_namespace n ON n.oid = t.typnamespace
		WHERE n.nspname = 'public'
			AND (t.typtype = 'e' OR t.typtype = 'c')  -- 'e' = ENUM, 'c' = COMPOSITE
			AND t.typname NOT LIKE '\\_%'  -- Исключаем системные типы
		ORDER BY t.typname
	`

	result, err := a.repository.ExecuteQuery(ctx, query)
	if err != nil {
		logger.Error("Ошибка загрузки пользовательских типов: %v", err)
		a.resultLabel.SetText("❌ Ошибка загрузки пользовательских типов: " + err.Error())
		return
	}

	// Очищаем карту пользовательских типов
	a.customTypeNames = make(map[string]bool)

	// Собираем все доступные типы данных
	allTypes := make([]string, 0)

	// Добавляем стандартные типы
	allTypes = append(allTypes, a.standardDataTypes...)

	// Добавляем пользовательские типы из результата запроса
	customTypesCount := 0
	for _, row := range result.Rows {
		if typeName, ok := row["type_name"].(string); ok {
			if typeCategory, ok := row["type_category"].(string); ok {
				// Добавляем тип в список
				allTypes = append(allTypes, typeName)
				a.customTypeNames[typeName] = true
				customTypesCount++

				logger.Info("Загружен пользовательский тип: %s (%s)", typeName, typeCategory)
			}
		}
	}

	// Обновляем список типов данных
	a.dataType.Options = allTypes
	a.dataType.Refresh()

	logger.Info("Загружено пользовательских типов: %d", customTypesCount)
}

// Метод для добавления пользовательского типа в селектор
func (a *AlterTableWindow) addCustomTypeToSelector(typeName string) {
	// Проверяем, нет ли уже этого типа в списке
	for _, option := range a.dataType.Options {
		if option == typeName {
			return // Уже есть
		}
	}

	// Добавляем пользовательский тип в список
	newOptions := append(a.dataType.Options, typeName)
	a.dataType.Options = newOptions
	a.dataType.Refresh()

	// Добавляем в карту пользовательских типов
	a.customTypeNames[typeName] = true
}

// SetCustomType устанавливает пользовательский тип для использования в ALTER TABLE
func (a *AlterTableWindow) SetCustomType(typeName string) {
	// Сначала проверяем, есть ли тип в текущем списке
	found := false
	for _, option := range a.dataType.Options {
		if option == typeName {
			found = true
			break
		}
	}

	// Если типа нет в списке, добавляем его
	if !found {
		a.addCustomTypeToSelector(typeName)
	}

	// Устанавливаем выбранным пользовательский тип
	a.dataType.SetSelected(typeName)

	// Показываем информационное сообщение
	a.resultLabel.SetText(fmt.Sprintf("✅ Выбран пользовательский тип: %s\nТеперь вы можете использовать его при добавлении столбца.", typeName))
}

// Проверка, является ли тип пользовательским
func (a *AlterTableWindow) isCustomType(typeName string) bool {
	// Проверяем стандартные типы
	for _, stdType := range a.standardDataTypes {
		if stdType == typeName {
			return false
		}
	}

	// Проверяем пользовательские типы
	return a.customTypeNames[typeName]
}

// Валидация использования пользовательского типа - ИСПРАВЛЕННАЯ ВЕРСИЯ
func (a *AlterTableWindow) validateCustomTypeUsage(typeName string) error {
	if a.isCustomType(typeName) {
		// Проверяем существование пользовательского типа в БД
		ctx := context.Background()

		// Форматируем запрос с параметром вместо использования плейсхолдера
		checkQuery := fmt.Sprintf(`
			SELECT 1 FROM pg_type t
			JOIN pg_namespace n ON n.oid = t.typnamespace
			WHERE n.nspname = 'public' AND t.typname = '%s'
		`, typeName)

		result, err := a.repository.ExecuteQuery(ctx, checkQuery)
		if err != nil {
			return fmt.Errorf("ошибка проверки типа %s: %v", typeName, err)
		}

		if len(result.Rows) == 0 {
			return fmt.Errorf("пользовательский тип %s не существует в базе данных", typeName)
		}
	}
	return nil
}

func (a *AlterTableWindow) onActionSelected(action string) {
	// Скрываем все поля сначала
	a.hideAllContainers()

	// Очищаем поля
	a.columnName.SetText("")
	a.newColumnName.SetText("")
	a.dataType.SetSelected("")
	a.constraintType.SetSelected("")
	a.constraintValue.SetText("")
	a.defaultValue.SetText("")
	a.referenceTable.SetText("")
	a.referenceColumn.SetText("")
	a.nullableCheck.SetChecked(true)

	switch action {
	case "Добавить столбец":
		a.columnNameContainer.Show()
		a.dataTypeContainer.Show()
		a.defaultValueContainer.Show()
		a.nullableCheckContainer.Show()
		a.resultLabel.SetText("Введите данные для нового столбца")

	case "Удалить столбец":
		a.columnNameContainer.Show()
		a.resultLabel.SetText("Выберите столбец для удаления")

	case "Переименовать столбец":
		a.columnNameContainer.Show()
		a.newColumnNameContainer.Show()
		a.resultLabel.SetText("Введите старое и новое имя столбца")

	case "Изменить тип данных":
		a.columnNameContainer.Show()
		a.dataTypeContainer.Show()
		a.resultLabel.SetText("Выберите столбец и новый тип данных. ВНИМАНИЕ: Все данные в столбце будут удалены!")

	case "Добавить ограничение":
		a.columnNameContainer.Show()
		a.constraintTypeContainer.Show()
		a.resultLabel.SetText("Выберите тип ограничения")

	case "Удалить ограничение":
		a.constraintValueContainer.Show()
		a.constraintValue.SetPlaceHolder("Имя ограничения")
		a.resultLabel.SetText("Введите имя ограничения для удаления")

	case "Переименовать таблицу":
		a.newColumnNameContainer.Show()
		a.newColumnName.SetPlaceHolder("Новое имя таблицы")
		a.resultLabel.SetText("Введите новое имя таблицы")
	}
}

func (a *AlterTableWindow) executeQuery(query string) {
	// Сохраняем информацию о текущем состоянии ДО выполнения запроса
	oldTableName := a.currentTable
	action := a.actionSelect.Selected

	logger.Info("Выполнение ALTER запроса: %s", query)
	logger.Info("Действие: %s, Текущая таблица: %s", action, oldTableName)

	// Проверяем существование таблицы перед выполнением
	tables, err := a.repository.GetTableNames(context.Background())
	if err != nil {
		a.showError(fmt.Errorf("не удалось проверить существование таблицы: %v", err))
		return
	}

	tableExists := false
	for _, table := range tables {
		if table == oldTableName {
			tableExists = true
			break
		}
	}

	if !tableExists && action != "Переименовать таблицу" {
		a.showError(fmt.Errorf("таблица %s не существует", oldTableName))
		return
	}

	err = a.repository.ExecuteAlter(context.Background(), query)
	if err != nil {
		a.showError(err)
		return
	}

	// ОСОБАЯ ОБРАБОТКА ДЛЯ ПЕРЕИМЕНОВАНИЯ ТАБЛИЦЫ
	if action == "Переименовать таблицу" {
		newTableName := strings.TrimSpace(a.newColumnName.Text)

		if newTableName == "" {
			logger.Error("Новое имя таблицы пустое")
			a.resultLabel.SetText("❌ Ошибка: новое имя таблицы не может быть пустым")
			return
		}

		logger.Info("Обработка переименования таблицы: %s -> %s", oldTableName, newTableName)

		// 1. Обновляем текущую таблицу в интерфейсе
		a.currentTable = newTableName
		logger.Info("Текущая таблица обновлена: %s", a.currentTable)

		// 2. Принудительно обновляем список таблиц
		a.loadTables()

		// 3. Устанавливаем новую таблицу как выбранную
		a.tableSelect.SetSelected(newTableName)
		logger.Info("Установлена новая таблица в селекторе: %s", newTableName)

		// 4. Обновляем результат
		a.resultLabel.SetText(fmt.Sprintf("✅ Таблица успешно переименована: %s -> %s\nSQL: %s",
			oldTableName, newTableName, query))
	} else {
		// Для других действий стандартное сообщение
		a.resultLabel.SetText("✅ Изменения успешно применены!\nSQL: " + query)
	}

	// Очищаем поля
	a.columnName.SetText("")
	a.newColumnName.SetText("")
	a.constraintValue.SetText("")
	a.defaultValue.SetText("")
	a.referenceTable.SetText("")
	a.referenceColumn.SetText("")

	// ВАЖНО: Вызываем callback для обновления всех окон данных
	logger.Info("Вызов callback onTableChanged после ALTER операции")
	if a.onTableChanged != nil {
		a.onTableChanged()
	} else {
		logger.Error("Callback onTableChanged не установлен!")
	}
}

// Метод для изменения типа данных через удаление и создание столбца
func (a *AlterTableWindow) changeDataType() {
	columnName := strings.TrimSpace(a.columnName.Text)
	newDataType := a.dataType.Selected

	if columnName == "" {
		a.showError(fmt.Errorf("не указано имя столбца"))
		return
	}
	if newDataType == "" {
		a.showError(fmt.Errorf("не выбран тип данных"))
		return
	}

	// Валидация пользовательского типа
	if err := a.validateCustomTypeUsage(newDataType); err != nil {
		a.showError(err)
		return
	}

	// Создаем транзакцию для безопасного выполнения
	ctx := context.Background()
	tx, err := a.repository.Pool().Begin(ctx)
	if err != nil {
		a.showError(fmt.Errorf("ошибка начала транзакции: %w", err))
		return
	}
	defer tx.Rollback(ctx)

	// Шаг 1: Удаляем столбец
	dropQuery := fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s", a.currentTable, columnName)
	logger.Info("Удаление столбца: %s", dropQuery)

	_, err = tx.Exec(ctx, dropQuery)
	if err != nil {
		a.showError(fmt.Errorf("ошибка удаления столбца: %w", err))
		return
	}

	// Шаг 2: Создаем столбец заново с новым типом
	addQuery := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", a.currentTable, columnName, newDataType)
	logger.Info("Создание столбца с новым типом: %s", addQuery)

	_, err = tx.Exec(ctx, addQuery)
	if err != nil {
		a.showError(fmt.Errorf("ошибка создания столбца: %w", err))
		return
	}

	// Коммитим транзакцию
	if err := tx.Commit(ctx); err != nil {
		a.showError(fmt.Errorf("ошибка коммита транзакции: %w", err))
		return
	}

	// Успешное завершение
	a.resultLabel.SetText(fmt.Sprintf("✅ Тип данных столбца '%s' успешно изменен на '%s'\n\nВыполненные операции:\n1. %s\n2. %s",
		columnName, newDataType, dropQuery, addQuery))

	// Очищаем поля
	a.columnName.SetText("")
	a.dataType.SetSelected("")

	// Вызываем callback для обновления всех окон
	if a.onTableChanged != nil {
		a.onTableChanged()
	}

	logger.Info("Тип данных столбца %s успешно изменен на %s", columnName, newDataType)
}

func (a *AlterTableWindow) loadTables() {
	logger.Info("Загрузка списка таблиц...")

	tables, err := a.repository.GetTableNames(context.Background())
	if err != nil {
		logger.Error("Ошибка загрузки таблиц: %v", err)
		a.showError(err)
		return
	}

	logger.Info("Получены таблицы из БД: %v", tables)

	// Сохраняем текущее выделение
	currentSelection := a.tableSelect.Selected
	logger.Info("Текущее выделение до обновления: %s", currentSelection)

	// Обновляем список таблиц
	a.tableSelect.Options = tables
	a.tableSelect.Refresh()

	// Восстанавливаем выделение, если таблица еще существует
	if currentSelection != "" {
		found := false
		for _, table := range tables {
			if table == currentSelection {
				a.tableSelect.SetSelected(currentSelection)
				logger.Info("Восстановлено выделение таблицы: %s", currentSelection)
				found = true
				break
			}
		}
		if !found {
			logger.Warn("Таблица %s не найдена в новом списке", currentSelection)
			// Если таблица не найдена, выбираем первую из списка
			if len(tables) > 0 {
				a.tableSelect.SetSelected(tables[0])
				a.currentTable = tables[0]
				logger.Info("Установлена первая таблица из списка: %s", tables[0])
			}
		}
	} else if len(tables) > 0 {
		// Если нет текущего выделения, выбираем первую таблицу
		a.tableSelect.SetSelected(tables[0])
		a.currentTable = tables[0]
		logger.Info("Установлена первая таблица: %s", tables[0])
	}

	logger.Info("Загрузка таблиц завершена. Текущая таблица: %s", a.currentTable)
}

func (a *AlterTableWindow) onTableSelected(table string) {
	if table == "" {
		logger.Warn("Пустое имя таблицы в onTableSelected")
		return
	}

	a.currentTable = table
	logger.Info("Таблица выбрана: %s", table)
	a.resultLabel.SetText(fmt.Sprintf("Выбрана таблица: %s", table))
}

func (a *AlterTableWindow) refreshData() {
	a.loadTables()
	a.loadCustomTypes() // Обновляем и пользовательские типы
	a.resultLabel.SetText("✅ Данные обновлены (таблицы и пользовательские типы)")

	// Вызываем callback для обновления главного окна
	if a.onTableChanged != nil {
		a.onTableChanged()
	}
}

func (a *AlterTableWindow) onConstraintTypeChanged(constraintType string) {
	// Показываем/скрываем дополнительные поля в зависимости от типа ограничения
	a.constraintValueContainer.Hide()
	a.referenceContainer.Hide()

	switch constraintType {
	case "CHECK":
		a.constraintValueContainer.Show()
		a.constraintValue.SetPlaceHolder("Условие CHECK (например: salary > 0)")
	case "FOREIGN KEY":
		a.referenceContainer.Show()
	case "NOT NULL":
		// Для NOT NULL не нужны дополнительные поля
	default:
		// Для PRIMARY KEY, UNIQUE не нужны дополнительные поля
	}
}

func (a *AlterTableWindow) validateInput() error {
	if a.currentTable == "" {
		return fmt.Errorf("не выбрана таблица")
	}

	action := a.actionSelect.Selected
	if action == "" {
		return fmt.Errorf("не выбрано действие")
	}

	switch action {
	case "Добавить столбец":
		if strings.TrimSpace(a.columnName.Text) == "" {
			return fmt.Errorf("не указано имя столбца")
		}
		if a.dataType.Selected == "" {
			return fmt.Errorf("не выбран тип данных")
		}

	case "Удалить столбец":
		if strings.TrimSpace(a.columnName.Text) == "" {
			return fmt.Errorf("не указано имя столбца")
		}

	case "Переименовать столбец":
		if strings.TrimSpace(a.columnName.Text) == "" {
			return fmt.Errorf("не указано текущее имя столбца")
		}
		if strings.TrimSpace(a.newColumnName.Text) == "" {
			return fmt.Errorf("не указано новое имя столбца")
		}

	case "Изменить тип данных":
		if strings.TrimSpace(a.columnName.Text) == "" {
			return fmt.Errorf("не указано имя столбца")
		}
		if a.dataType.Selected == "" {
			return fmt.Errorf("не выбран тип данных")
		}

	case "Добавить ограничение":
		if a.constraintType.Selected == "" {
			return fmt.Errorf("не выбран тип ограничения")
		}
		if strings.TrimSpace(a.columnName.Text) == "" && a.constraintType.Selected != "CHECK" {
			return fmt.Errorf("не указано имя столбца")
		}

		// Валидация для конкретных типов ограничений
		switch a.constraintType.Selected {
		case "FOREIGN KEY":
			if strings.TrimSpace(a.referenceTable.Text) == "" {
				return fmt.Errorf("для FOREIGN KEY не указана таблица для ссылки")
			}
			if strings.TrimSpace(a.referenceColumn.Text) == "" {
				return fmt.Errorf("для FOREIGN KEY не указан столбец для ссылки")
			}
		case "CHECK":
			if strings.TrimSpace(a.constraintValue.Text) == "" {
				return fmt.Errorf("для CHECK не указано условие")
			}
		}

	case "Удалить ограничение":
		if strings.TrimSpace(a.constraintValue.Text) == "" {
			return fmt.Errorf("не указано имя ограничения")
		}

	case "Переименовать таблицу":
		if strings.TrimSpace(a.newColumnName.Text) == "" {
			return fmt.Errorf("не указано новое имя таблицы")
		}
	}

	return nil
}

func (a *AlterTableWindow) buildAlterQuery() (string, error) {
	if err := a.validateInput(); err != nil {
		return "", err
	}

	action := a.actionSelect.Selected
	var query string

	switch action {
	case "Добавить столбец":
		// Валидация пользовательского типа
		if err := a.validateCustomTypeUsage(a.dataType.Selected); err != nil {
			return "", err
		}

		query = fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s",
			a.currentTable, a.columnName.Text, a.dataType.Selected)

		if a.defaultValue.Text != "" {
			query += " DEFAULT " + a.defaultValue.Text
		}
		if !a.nullableCheck.Checked {
			query += " NOT NULL"
		}

	case "Удалить столбец":
		query = fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s",
			a.currentTable, a.columnName.Text)

	case "Переименовать столбец":
		query = fmt.Sprintf("ALTER TABLE %s RENAME COLUMN %s TO %s",
			a.currentTable, a.columnName.Text, a.newColumnName.Text)

	case "Изменить тип данных":
		// Для изменения типа данных используем специальный метод changeDataType
		// Здесь возвращаем пустую строку, так как обработка будет в applyChanges
		return "", nil

	case "Добавить ограничение":
		constraintName := fmt.Sprintf("%s_%s_%s",
			a.currentTable, a.columnName.Text, strings.ToLower(strings.ReplaceAll(a.constraintType.Selected, " ", "_")))

		switch a.constraintType.Selected {
		case "PRIMARY KEY":
			query = fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s PRIMARY KEY (%s)",
				a.currentTable, constraintName, a.columnName.Text)
		case "FOREIGN KEY":
			query = fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s(%s)",
				a.currentTable, constraintName, a.columnName.Text, a.referenceTable.Text, a.referenceColumn.Text)
		case "UNIQUE":
			query = fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s UNIQUE (%s)",
				a.currentTable, constraintName, a.columnName.Text)
		case "CHECK":
			query = fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s CHECK (%s)",
				a.currentTable, constraintName, a.constraintValue.Text)
		case "NOT NULL":
			query = fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET NOT NULL",
				a.currentTable, a.columnName.Text)
		}

	case "Удалить ограничение":
		query = fmt.Sprintf("ALTER TABLE %s DROP CONSTRAINT %s",
			a.currentTable, a.constraintValue.Text)

	case "Переименовать таблицу":
		query = fmt.Sprintf("ALTER TABLE %s RENAME TO %s",
			a.currentTable, a.newColumnName.Text)
	}

	return query, nil
}

func (a *AlterTableWindow) applyChanges() {
	action := a.actionSelect.Selected

	// ОСОБАЯ ОБРАБОТКА ДЛЯ ИЗМЕНЕНИЯ ТИПА ДАННЫХ
	if action == "Изменить тип данных" {
		// Показываем предупреждение о потере данных
		dialog.ShowConfirm("Внимание: Потеря данных",
			fmt.Sprintf("Изменение типа данных столбца '%s' приведет к ПОЛНОЙ ПОТЕРЕ всех данных в этом столбце!\n\nПродолжить?",
				a.columnName.Text),
			func(confirmed bool) {
				if confirmed {
					a.changeDataType()
				}
			}, a.window)
		return
	}

	// Для остальных действий используем стандартную логику
	query, err := a.buildAlterQuery()
	if err != nil {
		a.showError(err)
		return
	}

	// Если для изменения типа данных вернулась пустая строка, значит что-то пошло не так
	if action == "Изменить тип данных" && query == "" {
		a.showError(fmt.Errorf("не удалось построить запрос для изменения типа данных"))
		return
	}

	// Подтверждение для деструктивных операций
	if action == "Удалить столбец" || action == "Удалить ограничение" {
		dialog.ShowConfirm("Подтверждение",
			fmt.Sprintf("Вы уверены, что хотите выполнить действие: %s?\n\nSQL: %s", action, query),
			func(confirmed bool) {
				if confirmed {
					a.executeQuery(query)
				}
			}, a.window)
	} else {
		a.executeQuery(query)
	}
}

func (a *AlterTableWindow) showSQL() {
	action := a.actionSelect.Selected

	// ОСОБАЯ ОБРАБОТКА ДЛЯ ИЗМЕНЕНИЯ ТИПА ДАННЫХ
	if action == "Изменить тип данных" {
		columnName := strings.TrimSpace(a.columnName.Text)
		newDataType := a.dataType.Selected

		if columnName == "" || newDataType == "" {
			a.resultLabel.SetText("Заполните имя столбца и выберите тип данных")
			return
		}

		dropQuery := fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s", a.currentTable, columnName)
		addQuery := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", a.currentTable, columnName, newDataType)

		a.resultLabel.SetText(fmt.Sprintf("Будут выполнены два запроса:\n\n1. %s\n\n2. %s", dropQuery, addQuery))
		return
	}

	// Для остальных действий стандартная логика
	query, err := a.buildAlterQuery()
	if err != nil {
		a.showError(err)
		return
	}

	a.resultLabel.SetText("SQL запрос:\n" + query)
}

func (a *AlterTableWindow) showError(err error) {
	dialog.ShowError(err, a.window)
	a.resultLabel.SetText("❌ Ошибка: " + err.Error())
}

func (a *AlterTableWindow) Show() {
	a.window.Show()
}
