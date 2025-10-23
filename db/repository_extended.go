package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing-platform/db/models"
	"testing-platform/pkg/logger"
)

// GetTableSchema возвращает информацию о структуре таблицы
func (r *Repository) GetTableSchema(ctx context.Context, tableName string) ([]models.ColumnInfo, error) {
	return r.GetTableColumns(ctx, tableName)
}

// GetTableData возвращает все данные из таблицы
func (r *Repository) GetTableData(ctx context.Context, tableName string) (*models.QueryResult, error) {
	query := fmt.Sprintf("SELECT * FROM %s ORDER BY id DESC", tableName)
	return r.ExecuteQuery(ctx, query)
}

// GetTables возвращает список всех таблиц в базе данных
func (r *Repository) GetTables(ctx context.Context) ([]string, error) {
	query := `
        SELECT table_name 
        FROM information_schema.tables 
        WHERE table_schema = 'public' 
        ORDER BY table_name
    `

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		logger.Error("Ошибка при получении списка таблиц: %v", err)
		return nil, fmt.Errorf("не удалось получить список таблиц: %w", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			return nil, err
		}
		tables = append(tables, table)
	}

	return tables, nil
}

// GetTableColumns возвращает информацию о столбцах таблицы
func (r *Repository) GetTableColumns(ctx context.Context, tableName string) ([]models.ColumnInfo, error) {
	query := `
        SELECT 
            column_name,
            data_type,
            is_nullable = 'YES' as is_nullable,
            column_default,
            character_maximum_length,
            EXISTS (
                SELECT 1 
                FROM information_schema.key_column_usage kcu
                WHERE kcu.table_name = $1 
                AND kcu.column_name = c.column_name
                AND kcu.table_schema = 'public'
            ) as is_primary_key
        FROM information_schema.columns c
        WHERE table_name = $1 AND table_schema = 'public'
        ORDER BY ordinal_position
    `

	rows, err := r.pool.Query(ctx, query, tableName)
	if err != nil {
		logger.Error("Ошибка при получении столбцов таблицы %s: %v", tableName, err)
		return nil, fmt.Errorf("не удалось получить столбцы таблицы: %w", err)
	}
	defer rows.Close()

	var columns []models.ColumnInfo
	for rows.Next() {
		var col models.ColumnInfo
		var isNullable bool
		var defaultValue sql.NullString
		var maxLength sql.NullInt64

		err := rows.Scan(&col.Name, &col.DataType, &isNullable, &defaultValue, &maxLength, &col.IsPrimaryKey)
		if err != nil {
			return nil, err
		}

		col.IsNullable = isNullable
		if defaultValue.Valid {
			col.DefaultValue = defaultValue.String
		}
		if maxLength.Valid {
			col.MaxLength = int(maxLength.Int64)
		}

		columns = append(columns, col)
	}

	return columns, nil
}

// ExecuteQuery выполняет произвольный SQL запрос и возвращает результат
func (r *Repository) ExecuteQuery(ctx context.Context, query string) (*models.QueryResult, error) {
	logger.Info("Выполнение запроса: %s", query)

	// Начинаем транзакцию для безопасного выполнения
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("ошибка начала транзакции: %w", err)
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, query)
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

	// Если это SELECT, коммитим
	if strings.HasPrefix(strings.ToUpper(strings.TrimSpace(query)), "SELECT") {
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("ошибка коммита транзакции: %w", err)
		}
	}

	return &models.QueryResult{
		Columns: columns,
		Rows:    resultRows,
	}, nil
}

// ExecuteAlter выполняет ALTER TABLE операции в транзакции
func (r *Repository) ExecuteAlter(ctx context.Context, query string) error {
	logger.Info("Выполнение ALTER: %s", query)

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("ошибка начала транзакции: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, query)
	if err != nil {
		logger.Error("Ошибка выполнения ALTER: %v", err)
		return fmt.Errorf("ошибка выполнения ALTER TABLE: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("ошибка коммита транзакции: %w", err)
	}

	logger.Info("ALTER TABLE выполнен успешно")
	return nil
}

// GetForeignKeyInfo возвращает информацию о внешних ключах
func (r *Repository) GetForeignKeyInfo(ctx context.Context) ([]map[string]interface{}, error) {
	query := `
        SELECT
            tc.table_name,
            kcu.column_name,
            ccu.table_name AS foreign_table_name,
            ccu.column_name AS foreign_column_name
        FROM information_schema.table_constraints AS tc
        JOIN information_schema.key_column_usage AS kcu
            ON tc.constraint_name = kcu.constraint_name
        JOIN information_schema.constraint_column_usage AS ccu
            ON ccu.constraint_name = tc.constraint_name
        WHERE tc.constraint_type = 'FOREIGN KEY'
    `

	result, err := r.ExecuteQuery(ctx, query)
	if err != nil {
		return nil, err
	}

	return result.Rows, nil
}
