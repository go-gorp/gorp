package gorp

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
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

	AutoIncrBindValue() string

	AutoIncrInsertSuffix(col *ColumnMap) string

	// Creates the trailing foreign key reference in a column specification.
	CreateForeignKeySuffix(references *ForeignKey) string

	// Creates the separate foreign key reference for a column.
	CreateForeignKeyBlock(col *ColumnMap) string

	// string to append to "create table" statement for vendor specific
	// table attributes
	CreateTableSuffix() string

	// string to truncate tables
	TruncateClause() string

	InsertAutoIncr(exec SqlExecutor, insertSql string, params ...interface{}) (int64, error)

	// bind variable string to use when forming SQL statements
	// in many dbs it is "?", but Postgres appears to use $1
	//
	// i is a zero based index of the bind variable in this statement
	//
	BindVar(i int) string

	// Handles quoting of a field name to ensure that it doesn't raise any
	// SQL parsing exceptions by using a reserved word as a field name.
	QuoteField(field string) string

	// Handles building up of a schema.database string that is compatible with
	// the given dialect
	//
	// schema - The schema that <table> lives in
	// table - The table name
	QuotedTableForQuery(schema string, table string) string

	// Sends an initialisation instruction when connecting to the database.
	// Primarily, this exists for Sqlite3 because foreign keys are disable
	// by default, unlike Postgresql and Mysql InnoDB.
	InitString() string
}

func standardInsertAutoIncr(exec SqlExecutor, insertSql string, params ...interface{}) (int64, error) {
	res, err := exec.Exec(insertSql, params...)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func standardOnChangeStr(change string, action FKOnChangeAction) string {
	prefix := "\n    "
	switch action {
	case Unspecified: return ""
	case NoAction: return prefix + "on " + change + " no action"
	case Restrict: return prefix + "on " + change + " restrict"
	case Cascade: return prefix + "on " + change + " cascade"
	case SetNull: return prefix + "on " + change + " set null"
	case Delete: return prefix + "on " + change + " delete"
	}
	return ""
}

///////////////////////////////////////////////////////
// sqlite3 //
/////////////

type SqliteDialect struct {
	suffix string
}

func (d SqliteDialect) ToSqlType(val reflect.Type, maxsize int, isAutoIncr bool) string {
	switch val.Kind() {
	case reflect.Ptr:
		return d.ToSqlType(val.Elem(), maxsize, isAutoIncr)
	case reflect.Bool:
		return "integer"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "integer"
	case reflect.Float64, reflect.Float32:
		return "real"
	case reflect.Slice:
		if val.Elem().Kind() == reflect.Uint8 {
			return "blob"
		}
	}

	switch val.Name() {
	case "NullInt64":
		return "integer"
	case "NullFloat64":
		return "real"
	case "NullBool":
		return "integer"
	case "Time":
		return "datetime"
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

func (d SqliteDialect) AutoIncrBindValue() string {
	return "null"
}

func (d SqliteDialect) AutoIncrInsertSuffix(col *ColumnMap) string {
	return ""
}

func (d SqliteDialect) CreateForeignKeySuffix(references *ForeignKey) string {
	return ""
}

func (d SqliteDialect) CreateForeignKeyBlock(col *ColumnMap) string {
	return fmt.Sprintf("foreign key (%s) references %s (%s)",
		d.QuoteField(col.ColumnName),
		d.QuoteField(col.References.ReferencedTable),
		d.QuoteField(col.References.ReferencedColumn)) +
			standardOnChangeStr("update", col.References.ActionOnUpdate) +
			standardOnChangeStr("delete", col.References.ActionOnDelete)
}

// Returns suffix
func (d SqliteDialect) CreateTableSuffix() string {
	return d.suffix
}

// With sqlite, there technically isn't a TRUNCATE statement,
// but a DELETE FROM uses a truncate optimization:
// http://www.sqlite.org/lang_delete.html
func (d SqliteDialect) TruncateClause() string {
	return "delete from"
}

// Returns "?"
func (d SqliteDialect) BindVar(i int) string {
	return "?"
}

func (d SqliteDialect) InsertAutoIncr(exec SqlExecutor, insertSql string, params ...interface{}) (int64, error) {
	return standardInsertAutoIncr(exec, insertSql, params...)
}

func (d SqliteDialect) QuoteField(f string) string {
	return `"` + f + `"`
}

// sqlite does not have schemas like PostgreSQL does, so just escape it like normal
func (d SqliteDialect) QuotedTableForQuery(schema string, table string) string {
	return d.QuoteField(table)
}

// sqlite3 has foreign keys disabled by default (will be enabled in sqlite4).
func (d SqliteDialect) InitString() string {
	return "pragma foreign_keys = ON;"
}

///////////////////////////////////////////////////////
// PostgreSQL //
////////////////

type PostgresDialect struct {
	suffix string
}

func (d PostgresDialect) ToSqlType(val reflect.Type, maxsize int, isAutoIncr bool) string {
	switch val.Kind() {
	case reflect.Ptr:
		return d.ToSqlType(val.Elem(), maxsize, isAutoIncr)
	case reflect.Bool:
		return "boolean"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Uint8, reflect.Uint16, reflect.Uint32:
		if isAutoIncr {
			return "serial"
		}
		return "integer"
	case reflect.Int64, reflect.Uint64:
		if isAutoIncr {
			return "bigserial"
		}
		return "bigint"
	case reflect.Float64:
		return "double precision"
	case reflect.Float32:
		return "real"
	case reflect.Slice:
		if val.Elem().Kind() == reflect.Uint8 {
			return "bytea"
		}
	}

	switch val.Name() {
	case "NullInt64":
		return "bigint"
	case "NullFloat64":
		return "double precision"
	case "NullBool":
		return "boolean"
	case "Time":
		return "timestamp with time zone"
	}

	if maxsize > 0 {
		return fmt.Sprintf("varchar(%d)", maxsize)
	} else {
		return "text"
	}

}

// Returns empty string
func (d PostgresDialect) AutoIncrStr() string {
	return ""
}

func (d PostgresDialect) AutoIncrBindValue() string {
	return "default"
}

func (d PostgresDialect) AutoIncrInsertSuffix(col *ColumnMap) string {
	return " returning " + col.ColumnName
}

func (d PostgresDialect) CreateForeignKeySuffix(references *ForeignKey) string {
	refTable := d.QuotedTableForQuery("", references.ReferencedTable)
	refField := d.QuoteField(references.ReferencedColumn)
	return fmt.Sprintf(" references %s (%s)%s%s", refTable, refField,
		standardOnChangeStr("delete", references.ActionOnDelete),
		standardOnChangeStr("update", references.ActionOnUpdate))
}

func (d PostgresDialect) CreateForeignKeyBlock(col *ColumnMap) string {
	return ""
}

// Returns suffix
func (d PostgresDialect) CreateTableSuffix() string {
	return d.suffix
}

func (d PostgresDialect) TruncateClause() string {
	return "truncate"
}

// Returns "$(i+1)"
func (d PostgresDialect) BindVar(i int) string {
	return fmt.Sprintf("$%d", i+1)
}

func (d PostgresDialect) InsertAutoIncr(exec SqlExecutor, insertSql string, params ...interface{}) (int64, error) {
	rows, err := exec.Query(insertSql, params...)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	if rows.Next() {
		var id int64
		err := rows.Scan(&id)
		return id, err
	}

	return 0, errors.New("No serial value returned for insert: "+insertSql+" Encountered error: "+rows.Err().Error())
}

func (d PostgresDialect) QuoteField(f string) string {
	return `"` + strings.ToLower(f) + `"`
}

func (d PostgresDialect) QuotedTableForQuery(schema string, table string) string {
	if strings.TrimSpace(schema) == "" {
		return d.QuoteField(table)
	}

	return schema + "." + d.QuoteField(table)
}

func (d PostgresDialect) InitString() string {
	return ""
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

func (d MySQLDialect) ToSqlType(val reflect.Type, maxsize int, isAutoIncr bool) string {
	switch val.Kind() {
	case reflect.Ptr:
		return d.ToSqlType(val.Elem(), maxsize, isAutoIncr)
	case reflect.Bool:
		return "boolean"
	case reflect.Int8:
		return "tinyint"
	case reflect.Uint8:
		return "tinyint unsigned"
	case reflect.Int16:
		return "smallint"
	case reflect.Uint16:
		return "smallint unsigned"
	case reflect.Int, reflect.Int32:
		return "int"
	case reflect.Uint, reflect.Uint32:
		return "int unsigned"
	case reflect.Int64:
		return "bigint"
	case reflect.Uint64:
		return "bigint unsigned"
	case reflect.Float64, reflect.Float32:
		return "double"
	case reflect.Slice:
		if val.Elem().Kind() == reflect.Uint8 {
			return "mediumblob"
		}
	}

	switch val.Name() {
	case "NullInt64":
		return "bigint"
	case "NullFloat64":
		return "double"
	case "NullBool":
		return "tinyint"
	case "Time":
		return "datetime"
	}

	if maxsize < 1 {
		maxsize = 255
	}
	return fmt.Sprintf("varchar(%d)", maxsize)
}

// Returns auto_increment
func (d MySQLDialect) AutoIncrStr() string {
	return "auto_increment"
}

func (d MySQLDialect) AutoIncrBindValue() string {
	return "null"
}

func (d MySQLDialect) AutoIncrInsertSuffix(col *ColumnMap) string {
	return ""
}

func (d MySQLDialect) CreateForeignKeySuffix(references *ForeignKey) string {
	return ""
}

func (d MySQLDialect) CreateForeignKeyBlock(col *ColumnMap) string {
	return fmt.Sprintf("foreign key (%s) references %s (%s)",
		d.QuoteField(col.ColumnName),
		d.QuoteField(col.References.ReferencedTable),
		d.QuoteField(col.References.ReferencedColumn)) +
			standardOnChangeStr("update", col.References.ActionOnUpdate) +
			standardOnChangeStr("delete", col.References.ActionOnDelete)
}

// Returns engine=%s charset=%s  based on values stored on struct
func (d MySQLDialect) CreateTableSuffix() string {
	if d.Engine == "" || d.Encoding == "" {
		msg := "gorp - undefined"

		if d.Engine == "" {
			msg += " MySQLDialect.Engine"
		}
		if d.Engine == "" && d.Encoding == "" {
			msg += ","
		}
		if d.Encoding == "" {
			msg += " MySQLDialect.Encoding"
		}
		msg += ". Check that your MySQLDialect was correctly initialized when declared."
		panic(msg)
	}

	return fmt.Sprintf(" engine=%s charset=%s", d.Engine, d.Encoding)
}

func (d MySQLDialect) TruncateClause() string {
	return "truncate"
}

// Returns "?"
func (d MySQLDialect) BindVar(i int) string {
	return "?"
}

func (d MySQLDialect) InsertAutoIncr(exec SqlExecutor, insertSql string, params ...interface{}) (int64, error) {
	return standardInsertAutoIncr(exec, insertSql, params...)
}

func (d MySQLDialect) QuoteField(f string) string {
	return "`" + f + "`"
}

// MySQL does not have schemas like PostgreSQL does, so just escape it like normal
func (d MySQLDialect) QuotedTableForQuery(schema string, table string) string {
	return d.QuoteField(table)
}

func (d MySQLDialect) InitString() string {
	return ""
}
