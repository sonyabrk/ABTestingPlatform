package models

// ColumnInfo представляет информацию о столбце таблицы
type ColumnInfo struct {
	Name         string `json:"name"`
	DataType     string `json:"data_type"`
	IsNullable   bool   `json:"is_nullable"`
	DefaultValue string `json:"default_value"`
	MaxLength    int    `json:"max_length"`
	IsPrimaryKey bool   `json:"is_primary_key"`
}

// TableInfo информация о таблице
type TableInfo struct {
	Name    string       `json:"name"`
	Columns []ColumnInfo `json:"columns"`
}

// QueryResult результат выполнения произвольного запроса
type QueryResult struct {
	Columns []string                 `json:"columns"`
	Rows    []map[string]interface{} `json:"rows"`
	Error   string                   `json:"error,omitempty"`
}

// WhereCondition условие WHERE
type WhereCondition struct {
	Column    string `json:"column"`
	Operator  string `json:"operator"`
	Value     string `json:"value"`
	LogicalOp string `json:"logical_op"` // AND/OR
}

// OrderBy сортировка
type OrderBy struct {
	Column    string `json:"column"`
	Direction string `json:"direction"` // ASC/DESC
}

// JoinDefinition определение JOIN
type JoinDefinition struct {
	Type        string `json:"type"` // INNER, LEFT, RIGHT, FULL
	Table       string `json:"table"`
	LeftColumn  string `json:"left_column"`
	RightColumn string `json:"right_column"`
}

// StringFunction строковая функция
type StringFunction struct {
	Function   string            `json:"function"`
	Column     string            `json:"column"`
	Parameters map[string]string `json:"parameters"`
}

// TextSearchConfig конфигурация текстового поиска
type TextSearchConfig struct {
	Type    string `json:"type"` // LIKE, ~, ~*, !~, !~*
	Column  string `json:"column"`
	Pattern string `json:"pattern"`
}
