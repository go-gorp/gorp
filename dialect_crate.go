package gorp

import (
	"fmt"
	_ "net"
	"reflect"
	"strings"
	_ "time"
)

type CrateDialect struct {
	suffix string
}

func (d CrateDialect) QuerySuffix() string { return ";" }

func (d CrateDialect) ToSqlType(val reflect.Type, maxsize int, isAutoIncr bool) string {
	//Maxsize and isAutoIncrement are never used in this dialect
	//	switch val.Elem() {
	//	//https://crate.io/docs/reference/sql/data_types.html#ip
	//	case reflect.TypeOf(net.IP{}):
	//		return "ip"
	//	//https://crate.io/docs/reference/sql/data_types.html#timestamp
	//	case reflect.TypeOf(time.Time{}):
	//		return "timestamp"
	//	}

	switch val.Kind() {
	case reflect.Ptr:
		return d.ToSqlType(val.Elem(), maxsize, isAutoIncr)
	case reflect.Bool:
		return "boolean"
	//https://crate.io/docs/reference/sql/data_types.html#numeric-types
	case reflect.Int, reflect.Int32, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "integer"
	case reflect.Int8:
		return "byte"
	case reflect.Int16:
		return "short"
	case reflect.Int64:
		return "long"
	case reflect.Float32:
		return "float"
	case reflect.Float64:
		return "double"
	//https://crate.io/docs/reference/sql/data_types.html#string
	case reflect.String:
		return "string"
	//https://crate.io/docs/reference/sql/data_types.html#array
	case reflect.Slice:
		return "array(" + d.ToSqlType(val.Elem(), -1, false) + ")"
	default:
		return "object"
	}
}

func (d CrateDialect) AutoIncrStr() string {
	return "PRIMARY KEY"
}

func (d CrateDialect) AutoIncrBindValue() string {
	return ""
}

func (d CrateDialect) AutoIncrInsertSuffix(col *ColumnMap) string {
	return ""
}

func (d CrateDialect) CreateTableSuffix() string {
	return ""
}

func (m CrateDialect) CreateIndexSuffix() string {
	return ""
}

func (m CrateDialect) DropIndexSuffix() string {
	return ""
}

func (m CrateDialect) TruncateClause() string {
	return "DELETE FROM"
}

// Returns "?"
func (d CrateDialect) BindVar(i int) string {
	return "?"
}

func (d CrateDialect) QuoteField(f string) string {
	return `"` + f + `"`
}

func (d CrateDialect) QuotedTableForQuery(schema string, table string) string {
	if strings.TrimSpace(schema) == "" {
		return d.QuoteField(table)
	}

	return schema + "." + d.QuoteField(table)
}

func (d CrateDialect) IfSchemaNotExists(command, schema string) string {
	return fmt.Sprintf("%s IF NOT EXISTS", command)
}

func (d CrateDialect) IfTableExists(command, schema, table string) string {
	return fmt.Sprintf("%s IF EXISTS", command)
}

func (d CrateDialect) IfTableNotExists(command, schema, table string) string {
	return fmt.Sprintf("%s IF NOT EXISTS", command)
}

func (d CrateDialect) InsertQueryToTarget(exec SqlExecutor, insertSql, idSql string, target interface{}, params ...interface{}) error {
	_, err := exec.Exec(insertSql, params...)
	if err != nil {
		return err
	}
	_, err = exec.SelectInt(idSql)
	if err != nil {
		return err
	}
	return nil
}
