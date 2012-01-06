package gorp

import (
	"fmt"
	"reflect"
)

type Dialect interface {
	ToSqlType(val reflect.Type, maxsize int) string
	AutoIncrStr() string
	CreateTableSuffix() string
}

type MySQLDialect struct {
	Engine   string
	Encoding string
}

func (m MySQLDialect) ToSqlType(val reflect.Type, maxsize int) string {
	switch val.Kind() {
	case reflect.Int, reflect.Int16, reflect.Int32:
		return "int"
	case reflect.Int64:
		return "bigint"
	}

	if maxsize < 1 {
		maxsize = 255
	}
	return fmt.Sprintf("varchar(%d)", maxsize)
}

func (m MySQLDialect) AutoIncrStr() string {
	return "auto_increment"
}

func (m MySQLDialect) CreateTableSuffix() string {
	return fmt.Sprintf(" engine=%s charset=%s", m.Engine, m.Encoding)
}
