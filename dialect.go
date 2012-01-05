package gorp

import (
	"reflect"
	"fmt"
)

type Dialect interface {
	ToSqlType(val reflect.Type) string
	AutoIncrStr() string
	CreateTableSuffix() string
}

type MySQLDialect struct { 
	Engine     string
    Encoding   string
}

func (m MySQLDialect) ToSqlType(val reflect.Type) string {
    switch (val.Kind()) {
    case reflect.Int, reflect.Int16, reflect.Int32:
        return "int"
    case reflect.Int64:
        return "bigint"
    }

    return "varchar(255)"
}

func (m MySQLDialect) AutoIncrStr() string {
	return "auto_increment"
}

func (m MySQLDialect) CreateTableSuffix() string {
	return fmt.Sprintf(" engine=%s charset=%s", m.Engine, m.Encoding)
}