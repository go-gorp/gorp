package gorp

import (
	"fmt"
	"reflect"
)

// The Dialect interface encapsulates behaviors that differ across
// SQL databases.  At present the Dialect is only used by CreateTables()
// but this could change in the future
type Dialect interface {

	// ToSqlType returns the SQL column type to use when creating a 
	// table of the given Go Type.  maxsize can be used to switch based on
	// size.  For example, in MySQL []byte could map to BLOB, MEDIUMBLOB,
	// or LONGBLOB depending on the maxsize
	ToSqlType(val reflect.Type, maxsize int) string

	// string to append to primary key column definitions
	AutoIncrStr() string

	// string to append to "create table" statement for vendor specific
	// table attributes
	CreateTableSuffix() string
}

// Implementation of Dialect for MySQL databases.
type MySQLDialect struct {

	// Engine is the storage engine to use "InnoDB" vs "MyISAM" for example
	Engine string

	// Encoding is the character encoding to use for created tables
	Encoding string
}

func (m MySQLDialect) ToSqlType(val reflect.Type, maxsize int) string {
	switch val.Kind() {
	case reflect.Int, reflect.Int16, reflect.Int32:
		return "int"
	case reflect.Int64:
		return "bigint"
	case reflect.Float64, reflect.Float32:
		return "double"
	}

	switch val.Name() {
	case "NullableInt64":
		return "bigint"
	case "NullableFloat64":
		return "double"
	case "NullableBool":
		return "tinyint"
	case "NullableBytes":
		return "mediumblob"
	}

	fmt.Printf("type=%v\n", val)

	if maxsize < 1 {
		maxsize = 255
	}
	return fmt.Sprintf("varchar(%d)", maxsize)
}

// Returns auto_increment
func (m MySQLDialect) AutoIncrStr() string {
	return "auto_increment"
}

// Returns engine=%s charset=%s  based on values stored on struct
func (m MySQLDialect) CreateTableSuffix() string {
	return fmt.Sprintf(" engine=%s charset=%s", m.Engine, m.Encoding)
}
