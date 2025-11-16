package models

// CustomTypeType тип пользовательского типа
type CustomTypeType string

const (
	EnumType      CustomTypeType = "ENUM"
	CompositeType CustomTypeType = "COMPOSITE"
)

// CustomType представляет пользовательский тип данных
type CustomType struct {
	Name            string         `json:"name"`
	Type            CustomTypeType `json:"type"`
	EnumValues      []string       `json:"enum_values,omitempty"`
	CompositeFields []TypeField    `json:"composite_fields,omitempty"`
}

type CompositeField struct {
	Name     string `json:"name"`
	DataType string `json:"data_type"`
}

// TypeField представляет поле составного типа
type TypeField struct {
	Name     string `json:"name"`
	DataType string `json:"data_type"`
}

// SubqueryCondition условие с подзапросом
type SubqueryCondition struct {
	Type       string `json:"type"` // ANY, ALL, EXISTS
	MainColumn string `json:"main_column"`
	Operator   string `json:"operator"` // для ANY/ALL: =, !=, >, <, >=, <=
	Subquery   string `json:"subquery"`
	TableAlias string `json:"table_alias,omitempty"` // для коррелированных подзапросов
}
