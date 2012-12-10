package gorp

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"time"
)

// The Dialect interface encapsulates behaviors that differ across
// SQL databases.  At present the Dialect is only used by CreateTables()
// but this could change in the future
type Dialect interface {

	// ToSqlType returns the SQL column type to use when creating a
	// table of the given Go Type.  maxsize can be used to switch based on
	// size.  For example, in MySQL []byte could map to BLOB, MEDIUMBLOB,
	// or LONGBLOB depending on the maxsize
	ToSqlType(val reflect.Type, maxsize int, isAutoIncr bool) string

	// string to append to primary key column definitions
	AutoIncrStr() string

	// string to append to "create table" statement for vendor specific
	// table attributes
	CreateTableSuffix() string

	LastInsertId(res *sql.Result, table *TableMap, exec SqlExecutor) (int64, error)

	// bind variable string to use when forming SQL statements
	// in many dbs it is "?", but Postgres appears to use $1
	//
	// i is a zero based index of the bind variable in this statement
	//
	BindVar(i int) string
}

///////////////////////////////////////////////////////
// sqlite3 //
/////////////

type SqliteDialect struct {
	suffix   string
}

func (d SqliteDialect) ToSqlType(val reflect.Type, maxsize int, isAutoIncr bool) string {
	switch val.Kind() {
	case reflect.Int, reflect.Int16, reflect.Int32, reflect.Int64:
		return "integer"
	case reflect.Float64, reflect.Float32:
		return "real"
	case reflect.Slice:
		if val.Elem().Kind() == reflect.Uint8 {
			return "blob"
		}
	}

	switch val.Name() {
	case "NullableInt64":
		return "integer"
	case "NullableFloat64":
		return "real"
	case "NullableBool":
		return "integer"
	case "NullableBytes":
		return "blob"
	}

	if maxsize < 1 {
		maxsize = 255
	}
	return fmt.Sprintf("varchar(%d)", maxsize)
}

// Returns autoincrement
func (d SqliteDialect) AutoIncrStr() string {
	return "autoincrement"
}

// Returns suffix
func (d SqliteDialect) CreateTableSuffix() string {
	return d.suffix
}

// Returns "?"
func (d SqliteDialect) BindVar(i int) string {
	return "?"
}

func (d SqliteDialect) LastInsertId(res *sql.Result, table *TableMap, exec SqlExecutor) (int64, error) {
	return (*res).LastInsertId()
}

///////////////////////////////////////////////////////
// PostgreSQL //
////////////////

type PostgresDialect struct {
	suffix string
}

func (d PostgresDialect) ToSqlType(val reflect.Type, maxsize int, isAutoIncr bool) string {
	switch val.Kind() {
	case reflect.Int, reflect.Int16, reflect.Int32:
		if isAutoIncr {
			return "serial"
		}
		return "integer"
	case reflect.Int64:
		if isAutoIncr {
			return "serial"
		}
		return "bigint"
	case reflect.Float64, reflect.Float32:
		return "real"
	case reflect.Slice:
		if val.Elem().Kind() == reflect.Uint8 {
			return "bytea"
		}
	}

	switch val.Name() {
	case "NullableInt64":
		return "bigint"
	case "NullableFloat64":
		return "double"
	case "NullableBool":
		return "smallint"
	case "NullableBytes":
		return "bytea"
	}

	if maxsize < 1 {
		maxsize = 255
	}
	return fmt.Sprintf("varchar(%d)", maxsize)
}

// Returns empty string
func (d PostgresDialect) AutoIncrStr() string {
	return ""
}

// Returns suffix
func (d PostgresDialect) CreateTableSuffix() string {
	return d.suffix
}

// Returns "$(i+1)"
func (d PostgresDialect) BindVar(i int) string {
	return fmt.Sprintf("$%d", i+1)
}

func (d PostgresDialect) LastInsertId(res *sql.Result, table *TableMap, exec SqlExecutor) (int64, error) {
	sql := fmt.Sprintf("select currval('%s_%s_seq')", table.TableName, table.keys[0].ColumnName)
	rows, err := exec.query(sql)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	if rows.Next() {
		var dest int64
		err = rows.Scan(&dest)
		return dest, nil
	}
	return 0, errors.New(fmt.Sprintf("PostgresDialect: %s did not return a row", sql))
}

///////////////////////////////////////////////////////
// MySQL //
///////////

// Implementation of Dialect for MySQL databases.
type MySQLDialect struct {

	// Engine is the storage engine to use "InnoDB" vs "MyISAM" for example
	Engine string

	// Encoding is the character encoding to use for created tables
	Encoding string
}

func (m MySQLDialect) ToSqlType(val reflect.Type, maxsize int, isAutoIncr bool) string {
	switch val.Kind() {
	case reflect.Int, reflect.Int16, reflect.Int32:
		return "int"
	case reflect.Int64:
		return "bigint"
	case reflect.Float64, reflect.Float32:
		return "double"
	case reflect.Slice:
		if val.Elem().Kind() == reflect.Uint8 {
			return "mediumblob"
		}
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

// Returns "?"
func (m MySQLDialect) BindVar(i int) string {
	return "?"
}

func (m MySQLDialect) LastInsertId(res *sql.Result, table *TableMap, exec SqlExecutor) (int64, error) {
	return (*res).LastInsertId()
}
