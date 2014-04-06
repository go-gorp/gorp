// Copyright 2012 James Cooper. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// Package gorp provides a simple way to marshal Go structs to and from
// SQL databases.  It uses the database/sql package, and should work with any
// compliant database/sql driver.
//
// Source code and project home:
// https://github.com/coopernurse/gorp
//
package gorp

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// DbMap is the root gorp mapping object. Create one of these for each
// database schema you wish to map.  Each DbMap contains a list of
// mapped tables.
//
// Example:
//
//     dialect := gorp.MySQLDialect{"InnoDB", "UTF8"}
//     dbmap := &gorp.DbMap{Db: db, Dialect: dialect}
//
type DbMap struct {
	// Db handle to use with this map
	Db *sql.DB

	// Dialect implementation to use with this map
	Dialect Dialect

	TypeConverter TypeConverter

	tables    []*TableMap
	logger    GorpLogger
	logPrefix string
}

// AddTable registers the given interface type with gorp. The table name
// will be given the name of the TypeOf(i).  You must call this function,
// or AddTableWithName, for any struct type you wish to persist with
// the given DbMap.
//
// This operation is idempotent. If i's type is already mapped, the
// existing *TableMap is returned
func (m *DbMap) AddTable(i interface{}) *TableMap {
	return m.AddTableWithName(i, "")
}

// AddTableWithName has the same behavior as AddTable, but sets
// table.TableName to name.
func (m *DbMap) AddTableWithName(i interface{}, name string) *TableMap {
	return m.AddTableWithNameAndSchema(i, "", name)
}

// AddTableWithNameAndSchema has the same behavior as AddTable, but sets
// table.TableName to name.
func (m *DbMap) AddTableWithNameAndSchema(i interface{}, schema string, name string) *TableMap {
	t := reflect.TypeOf(i)
	if name == "" {
		name = t.Name()
	}

	// check if we have a table for this type already
	// if so, update the name and return the existing pointer
	for i := range m.tables {
		table := m.tables[i]
		if table.gotype == t {
			table.TableName = name
			return table
		}
	}

	tmap := &TableMap{gotype: t, TableName: name, SchemaName: schema, dbmap: m}
	tmap.columns, tmap.version = readStructColumns(t)
	m.tables = append(m.tables, tmap)

	return tmap
}

func readStructColumns(t reflect.Type) (cols []*ColumnMap, version *ColumnMap) {
	n := t.NumField()
	for i := 0; i < n; i++ {
		f := t.Field(i)
		if f.Anonymous && f.Type.Kind() == reflect.Struct {
			// Recursively add nested fields in embedded structs.
			subcols, subversion := readStructColumns(f.Type)
			// Don't append nested fields that have the same field
			// name as an already-mapped field.
			for _, subcol := range subcols {
				shouldAppend := true
				for _, col := range cols {
					if !subcol.Transient && subcol.fieldName == col.fieldName {
						shouldAppend = false
						break
					}
				}
				if shouldAppend {
					cols = append(cols, subcol)
				}
			}
			if subversion != nil {
				version = subversion
			}
		} else {
			columnName := f.Tag.Get("db")
			if columnName == "" {
				columnName = f.Name
			}
			cm := &ColumnMap{
				ColumnName: columnName,
				Transient:  columnName == "-",
				fieldName:  f.Name,
				gotype:     f.Type,
			}
			// Check for nested fields of the same field name and
			// override them.
			shouldAppend := true
			for index, col := range cols {
				if !col.Transient && col.fieldName == cm.fieldName {
					cols[index] = cm
					shouldAppend = false
					break
				}
			}
			if shouldAppend {
				cols = append(cols, cm)
			}
			if cm.fieldName == "Version" {
				version = cm
			}
		}
	}
	return
}

// CreateTables iterates through TableMaps registered to this DbMap and
// executes "create table" statements against the database for each.
//
// This is particularly useful in unit tests where you want to create
// and destroy the schema automatically.
func (m *DbMap) CreateTables() error {
	return m.createTables(false)
}

// CreateTablesIfNotExists is similar to CreateTables, but starts
// each statement with "create table if not exists" so that existing
// tables do not raise errors
func (m *DbMap) CreateTablesIfNotExists() error {
	return m.createTables(true)
}

func (m *DbMap) createTables(ifNotExists bool) error {
	var err error
	for _, t := range m.tables {
		ddl := m.createOneTableSql(ifNotExists, t)
		_, err := m.Exec(ddl)
		if err != nil {
			break
		}
	}
	return err
}

func (m *DbMap) createOneTableSql(ifNotExists bool, table *TableMap) string {
	s := bytes.Buffer{}

	if strings.TrimSpace(table.SchemaName) != "" {
		s.WriteString("create schema ")
		if ifNotExists {
			s.WriteString("if not exists ")
		}

		s.WriteString(table.SchemaName)
		s.WriteString(";")
	}

	s.WriteString("create table ")
	if ifNotExists {
		s.WriteString("if not exists ")
	}
	s.WriteString(m.Dialect.QuotedTableForQuery(table.SchemaName, table.TableName))
	s.WriteString("(")

	x := 0
	for _, col := range table.columns {
		if !col.Transient {
			if x > 0 {
				s.WriteString(", ")
			}
			stype := m.Dialect.ToSqlType(col.gotype, col.MaxSize, col.isAutoIncr)
			s.WriteString(fmt.Sprintf("%s %s", m.Dialect.QuoteField(col.ColumnName), stype))

			if col.isPK || col.isNotNull {
				s.WriteString(" not null")
			}
			if col.isPK && len(table.keys) == 1 {
				s.WriteString(" primary key")
			}
			if col.Unique {
				s.WriteString(" unique")
			}
			if col.isAutoIncr {
				s.WriteString(fmt.Sprintf(" %s", m.Dialect.AutoIncrStr()))
			}

			x++
		}
	}

	if len(table.keys) > 1 {
		s.WriteString(", primary key (")
		for x := range table.keys {
			if x > 0 {
				s.WriteString(", ")
			}
			s.WriteString(m.Dialect.QuoteField(table.keys[x].ColumnName))
		}
		s.WriteString(")")
	}

	if len(table.uniqueTogether) > 0 {
		for _, columns := range table.uniqueTogether {
			s.WriteString(", unique (")
			for i, column := range columns {
				if i > 0 {
					s.WriteString(", ")
				}
				s.WriteString(m.Dialect.QuoteField(column))
			}
			s.WriteString(")")
		}
	}
	s.WriteString(") ")
	s.WriteString(m.Dialect.CreateTableSuffix())
	s.WriteString(";")
	return s.String()
}

// DropTable drops an individual table.  Will throw an error
// if the table does not exist.
func (m *DbMap) DropTable(table interface{}) error {
	t := reflect.TypeOf(table)
	return m.dropTable(t, false)
}

// DropTable drops an individual table.  Will NOT throw an error
// if the table does not exist.
func (m *DbMap) DropTableIfExists(table interface{}) error {
	t := reflect.TypeOf(table)
	return m.dropTable(t, true)
}

// DropTables iterates through TableMaps registered to this DbMap and
// executes "drop table" statements against the database for each.
func (m *DbMap) DropTables() error {
	return m.dropTables(false)
}

// DropTablesIfExists is the same as DropTables, but uses the "if exists" clause to
// avoid errors for tables that do not exist.
func (m *DbMap) DropTablesIfExists() error {
	return m.dropTables(true)
}

// Goes through all the registered tables, dropping them one by one.
// If an error is encountered, then it is returned and the rest of
// the tables are not dropped.
func (m *DbMap) dropTables(addIfExists bool) (err error) {
	for _, table := range m.tables {
		err = m.dropTableImpl(table, addIfExists)
		if err != nil {
			return
		}
	}
	return err
}

// Implementation of dropping a single table.
func (m *DbMap) dropTable(t reflect.Type, addIfExists bool) error {
	table := tableOrNil(m, t)
	if table == nil {
		return errors.New(fmt.Sprintf("table %s was not registered!", table.TableName))
	}

	return m.dropTableImpl(table, addIfExists)
}

func (m *DbMap) dropTableImpl(table *TableMap, addIfExists bool) (err error) {
	ifExists := ""
	if addIfExists {
		ifExists = " if exists"
	}
	_, err = m.Exec(fmt.Sprintf("drop table%s %s;", ifExists, m.Dialect.QuotedTableForQuery(table.SchemaName, table.TableName)))
	return err
}

// TableMap represents a mapping between a Go struct and a database table
// Use dbmap.AddTable() or dbmap.AddTableWithName() to create these
type TableMap struct {
	// Name of database table.
	TableName      string
	SchemaName     string
	gotype         reflect.Type
	columns        []*ColumnMap
	keys           []*ColumnMap
	uniqueTogether [][]string
	version        *ColumnMap
	insertPlan     bindPlan
	updatePlan     bindPlan
	deletePlan     bindPlan
	getPlan        bindPlan
	dbmap          *DbMap
}

// ResetSql removes cached insert/update/select/delete SQL strings
// associated with this TableMap.  Call this if you've modified
// any column names or the table name itself.
func (t *TableMap) ResetSql() {
	t.insertPlan = bindPlan{}
	t.updatePlan = bindPlan{}
	t.deletePlan = bindPlan{}
	t.getPlan = bindPlan{}
}

// SetKeys lets you specify the fields on a struct that map to primary
// key columns on the table.  If isAutoIncr is set, result.LastInsertId()
// will be used after INSERT to bind the generated id to the Go struct.
//
// Automatically calls ResetSql() to ensure SQL statements are regenerated.
//
// Panics if isAutoIncr is true, and fieldNames length != 1
//
func (t *TableMap) SetKeys(isAutoIncr bool, fieldNames ...string) *TableMap {
	if isAutoIncr && len(fieldNames) != 1 {
		panic(fmt.Sprintf(
			"gorp: SetKeys: fieldNames length must be 1 if key is auto-increment. (Saw %v fieldNames)",
			len(fieldNames)))
	}
	t.keys = make([]*ColumnMap, 0)
	for _, name := range fieldNames {
		colmap := t.ColMap(name)
		colmap.isPK = true
		colmap.isAutoIncr = isAutoIncr
		t.keys = append(t.keys, colmap)
	}
	t.ResetSql()

	return t
}

// SetUniqueTogether lets you specify uniqueness constraints across multiple
// columns on the table. Each call adds an additional constraint for the
// specified columns.
//
// Automatically calls ResetSql() to ensure SQL statements are regenerated.
//
// Panics if fieldNames length < 2.
//
func (t *TableMap) SetUniqueTogether(fieldNames ...string) *TableMap {
	if len(fieldNames) < 2 {
		panic(fmt.Sprintf(
		"gorp: SetUniqueTogether: must provide at least two fieldNames to set uniqueness constraint."))
	}

	columns := make([]string, 0)
	for _, name := range fieldNames {
		columns = append(columns, name)
	}
	t.uniqueTogether = append(t.uniqueTogether, columns)
	t.ResetSql()

	return t
}

// ColMap returns the ColumnMap pointer matching the given struct field
// name.  It panics if the struct does not contain a field matching this
// name.
func (t *TableMap) ColMap(field string) *ColumnMap {
	col := colMapOrNil(t, field)
	if col == nil {
		e := fmt.Sprintf("No ColumnMap in table %s type %s with field %s",
			t.TableName, t.gotype.Name(), field)

		panic(e)
	}
	return col
}

func colMapOrNil(t *TableMap, field string) *ColumnMap {
	for _, col := range t.columns {
		if col.fieldName == field || col.ColumnName == field {
			return col
		}
	}
	return nil
}

// SetVersionCol sets the column to use as the Version field.  By default
// the "Version" field is used.  Returns the column found, or panics
// if the struct does not contain a field matching this name.
//
// Automatically calls ResetSql() to ensure SQL statements are regenerated.
func (t *TableMap) SetVersionCol(field string) *ColumnMap {
	c := t.ColMap(field)
	t.version = c
	t.ResetSql()
	return c
}

// ColumnMap represents a mapping between a Go struct field and a single
// column in a table.
// Unique and MaxSize only inform the
// CreateTables() function and are not used by Insert/Update/Delete/Get.
type ColumnMap struct {
	// Column name in db table
	ColumnName string

	// If true, this column is skipped in generated SQL statements
	Transient bool

	// If true, " unique" is added to create table statements.
	// Not used elsewhere
	Unique bool

	// Passed to Dialect.ToSqlType() to assist in informing the
	// correct column type to map to in CreateTables()
	// Not used elsewhere
	MaxSize int

	// If present, specifies that this column is a foreign key that
	// references another column of another table.
	References *ForeignKey

	fieldName  string
	gotype     reflect.Type
	isPK       bool
	isAutoIncr bool
	isNotNull  bool
}

// Rename allows you to specify the column name in the table
//
// Example:  table.ColMap("Updated").Rename("date_updated")
//
func (c *ColumnMap) Rename(colname string) *ColumnMap {
	c.ColumnName = colname
	return c
}

// SetTransient allows you to mark the column as transient. If true
// this column will be skipped when SQL statements are generated
func (c *ColumnMap) SetTransient(b bool) *ColumnMap {
	c.Transient = b
	return c
}

// SetUnique adds "unique" to the create table statements for this
// column, if b is true.
func (c *ColumnMap) SetUnique(b bool) *ColumnMap {
	c.Unique = b
	return c
}

// SetNotNull adds "not null" to the create table statements for this
// column, if nn is true.
func (c *ColumnMap) SetNotNull(nn bool) *ColumnMap {
	c.isNotNull = nn
	return c
}

// SetMaxSize specifies the max length of values of this column. This is
// passed to the dialect.ToSqlType() function, which can use the value
// to alter the generated type for "create table" statements
func (c *ColumnMap) SetMaxSize(size int) *ColumnMap {
	c.MaxSize = size
	return c
}

// SetForeignKey specifies the foreign-key relationship between this column
// and a column in another table.
func (c *ColumnMap) SetForeignKey(fk *ForeignKey) *ColumnMap {
	c.References = fk
	return c
}

// Specifies what foreign-key constraints will be enforced by the database.
type FKOnChangeAction int

const (
	UNSPECIFIED FKOnChangeAction = iota
	NO_ACTION
	RESTRICT
	CASCADE
	SET_NULL
	//SET_DEFAULT // may not be supported by MySql
	DELETE
)

// ForeignKey specifies the relationship formed when one column refers to the
// primary key of another table.
type ForeignKey struct {
	ReferencedTable  string
	ReferencedColumn string
	ActionOnDelete   FKOnChangeAction
	ActionOnUpdate   FKOnChangeAction
}

// NewForeignKey creates a new ForeignKey for a specified table/column reference.
func NewForeignKey(referencedTable, referencedColumn string) *ForeignKey {
	return &ForeignKey{referencedTable, referencedColumn, UNSPECIFIED, UNSPECIFIED}
}

// Sets the action that the database is to perform when the parent record
// is updated. The default is usually RESTRICT.
func (fk *ForeignKey) OnUpdate(action FKOnChangeAction) *ForeignKey {
	fk.ActionOnUpdate = action
	return fk
}

// Sets the action that the database is to perform when the parent record
// is deleted. The default is usually RESTRICT.
func (fk *ForeignKey) OnDelete(action FKOnChangeAction) *ForeignKey {
	fk.ActionOnDelete = action
	return fk
}

