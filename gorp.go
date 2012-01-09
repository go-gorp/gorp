// Copyright 2012 James Cooper. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// Package gorp provides a simple way to marshall Go structs to and from
// SQL databases.  It uses the exp/sql package, and should work with any 
// compliant exp/sql driver.
//
// Source code and project home:
// https://github.com/coopernurse/gorp
//
package gorp

import (
	"bytes"
	"errors"
	"exp/sql"
	"fmt"
	"log"
	"reflect"
)

var zeroVal reflect.Value

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

	tables    []*TableMap
	logger    *log.Logger
	logPrefix string
}

// TableMap represents a mapping between a Go struct and a database table
// Use dbmap.AddTable() or dbmap.AddTableWithName() to create these
type TableMap struct {
	// Name of database table.
	TableName string
	gotype    reflect.Type
	columns   []*ColumnMap
	keys      []*ColumnMap
}

// SetKeys lets you specify the fields on a struct that map to primary 
// key columns on the table.  If isAutoIncr is set, result.LastInsertId()
// will be used after INSERT to bind the generated id to the Go struct 
func (t *TableMap) SetKeys(isAutoIncr bool, fieldNames ...string) *TableMap {
	t.keys = make([]*ColumnMap, 0)
	for _, name := range fieldNames {
		colmap := t.ColMap(name)
		colmap.isPK = true
		colmap.isAutoIncr = isAutoIncr
		t.keys = append(t.keys, colmap)
	}

	return t
}

// ColMap returns the ColumnMap pointer matching the given struct field
// name.  It panics if the struct does not contain a field matching this
// name.
func (t *TableMap) ColMap(field string) *ColumnMap {
	for _, col := range t.columns {
		if col.fieldName == field {
			return col
		}
	}

	e := fmt.Sprintf("No ColumnMap in table %s type %s with field %s",
		t.TableName, t.gotype.Name(), field)
	panic(e)
}

// ColumnMap represents a mapping between a Go struct field and a single
// column in a table.
// Nullable, Unique, and MaxSize only inform the 
// CreateTables() function and are not used by Insert/Update/Delete/Get.
type ColumnMap struct {
	// Column name in db table
	ColumnName string

	// If true, this column is skipped in generated SQL statements
	Transient bool

	// If false, " not null" is added to create table statements.
	// Not used elsewhere
	Nullable bool

	// If true, " unique" is added to create table statements.
	// Not used elsewhere
	Unique bool

	// Passed to Dialect.ToSqlType() to assist in informing the
	// correct column type to map to in CreateTables()
	// Not used elsewhere
	MaxSize int

	fieldName  string
	gotype     reflect.Type
	isPK       bool
	isAutoIncr bool
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

// If true " not null" will be added to create table statements for this
// column
func (c *ColumnMap) SetNullable(b bool) *ColumnMap {
	c.Nullable = b
	return c
}

// If true " unique" will be added to create table statements for this
// column
func (c *ColumnMap) SetUnique(b bool) *ColumnMap {
	c.Unique = b
	return c
}

// SetMaxSize specifies the max length of values of this column. This is
// passed to the dialect.ToSqlType() function, which can use the value
// to alter the generated type for "create table" statements
func (c *ColumnMap) SetMaxSize(size int) *ColumnMap {
	c.MaxSize = size
	return c
}

// Transaction represents a database transaction.  
// Insert/Update/Delete/Get/Exec operations will be run in the context
// of that transaction.  Transactions should be terminated with
// a call to Commit() or Rollback()
type Transaction struct {
	dbmap *DbMap
	tx    *sql.Tx
}

// SqlExecutor exposes gorp operations that can be run from Pre/Post
// hooks.  This hides whether the current operation that triggered the
// hook is in a transaction.
//
// See the DbMap function docs for each of the functions below for more
// information.
type SqlExecutor interface {
	Get(i interface{}, keys ...interface{}) (interface{}, error)
	Insert(list ...interface{}) error
	Update(list ...interface{}) (int64, error)
	Delete(list ...interface{}) (int64, error)
	Exec(query string, args ...interface{}) (sql.Result, error)
	Select(i interface{}, query string,
		args ...interface{}) ([]interface{}, error)
	query(query string, args ...interface{}) (*sql.Rows, error)
	queryRow(query string, args ...interface{}) *sql.Row
}

// TraceOn turns on SQL statement logging for this DbMap.  After this is
// called, all SQL statements will be sent to the logger.  If prefix is
// a non-empty string, it will be written to the front of all logged 
// strings, which can aid in filtering log lines.
//
// Use TraceOn if you want to spy on the SQL statements that gorp
// generates.
func (m *DbMap) TraceOn(prefix string, logger *log.Logger) {
	m.logger = logger
	if prefix == "" {
		m.logPrefix = prefix
	} else {
		m.logPrefix = fmt.Sprintf("%s ", prefix)
	}
}

// TraceOff turns off tracing. It is idempotent.
func (m *DbMap) TraceOff() {
	m.logger = nil
	m.logPrefix = ""
}

// AddTable registers the given interface type with gorp. The table name
// will be given the name of the TypeOf(i).  You must call this function,
// or AddTableWithName, for any struct type you wish to persist with 
// the given DbMap.
func (m *DbMap) AddTable(i interface{}) *TableMap {
	return m.AddTableWithName(i, "")
}

// AddTableWithName has the same behavior as AddTable, but sets 
// table.TableName to name.
func (m *DbMap) AddTableWithName(i interface{}, name string) *TableMap {
	t := reflect.TypeOf(i)
	if name == "" {
		name = t.Name()
	}
	tmap := &TableMap{gotype: t, TableName: name}

	n := t.NumField()
	tmap.columns = make([]*ColumnMap, n, n)
	for i := 0; i < n; i++ {
		f := t.Field(i)
		tmap.columns[i] = &ColumnMap{
			ColumnName: f.Name,
			Nullable:   true,
			fieldName:  f.Name,
			gotype:     f.Type,
		}
	}

	// append to slice
	// expand slice as necessary
	n = len(m.tables)
	if (n + 1) > cap(m.tables) {
		newArr := make([]*TableMap, n, 2*(n+1))
		copy(newArr, m.tables)
		m.tables = newArr

	}
	m.tables = m.tables[0 : n+1]
	m.tables[n] = tmap

	return tmap
}

// CreateTables iterates through TableMaps registered to this DbMap and
// executes "create table" statements against the database for each.
// 
// This is particularly useful in unit tests where you want to create
// and destroy the schema automatically.
func (m *DbMap) CreateTables() error {
	var err error
	for i := range m.tables {
		table := m.tables[i]

		s := bytes.Buffer{}
		s.WriteString(fmt.Sprintf("create table %s (", table.TableName))
		x := 0
		for _, col := range table.columns {
			if !col.Transient {
				if x > 0 {
					s.WriteString(", ")
				}
				stype := m.Dialect.ToSqlType(col.gotype, col.MaxSize)
				s.WriteString(fmt.Sprintf("%s %s", col.ColumnName, stype))

				if !col.Nullable {
					s.WriteString(" not null")
				}
				if col.Unique {
					s.WriteString(" unique")
				}
				if col.isPK {
					s.WriteString(" primary key")
				}
				if col.isAutoIncr {
					s.WriteString(fmt.Sprintf(" %s", m.Dialect.AutoIncrStr()))
				}

				x++
			}
		}
		s.WriteString(") ")
		s.WriteString(m.Dialect.CreateTableSuffix())
		s.WriteString(";")
		_, err = m.Exec(s.String())
	}
	return err
}

// DropTables iterates through TableMaps registered to this DbMap and
// executes "drop table" statements against the database for each.
func (m *DbMap) DropTables() error {
	var err error
	for i := range m.tables {
		table := m.tables[i]
		_, e := m.Exec(fmt.Sprintf("drop table %s;", table.TableName))
		if e != nil {
			err = e
		}
	}
	return err
}

// Insert runs a SQL INSERT statement for each element in list.  List 
// items must be pointers.
//
// Any interface whose TableMap has an auto-increment primary key will
// have its last insert id bound to the PK field on the struct.
//
// Hook functions PreInsert() and/or PostInsert() will be executed 
// before/after the INSERT statement if the interface defines them.
//
// Panics if any interface in the list has not been registered with AddTable
func (m *DbMap) Insert(list ...interface{}) error {
	return insert(m, m, list...)
}

// Update runs a SQL UPDATE statement for each element in list.  List 
// items must be pointers.  
//
// Hook functions PreUpdate() and/or PostUpdate() will be executed 
// before/after the UPDATE statement if the interface defines them.
//
// Returns number of rows updated
//
// Returns an error if SetKeys has not been called on the TableMap
// Panics if any interface in the list has not been registered with AddTable
func (m *DbMap) Update(list ...interface{}) (int64, error) {
	return update(m, m, list...)
}

// Delete runs a SQL DELETE statement for each element in list.  List 
// items must be pointers.
//
// Hook functions PreDelete() and/or PostDelete() will be executed 
// before/after the DELETE statement if the interface defines them.  
//
// Returns number of rows deleted
//
// Returns an error if SetKeys has not been called on the TableMap
// Panics if any interface in the list has not been registered with AddTable
func (m *DbMap) Delete(list ...interface{}) (int64, error) {
	return delete(m, m, list...)
}

// Get runs a SQL SELECT to fetch a single row from the table based on the
// primary key(s)
//
//  i should be an empty value for the struct to load
//  keys should be the primary key value(s) for the row to load
//
// Hook function PostGet() will be executed 
// after the SELECT statement if the interface defines them.
//
// Returns a pointer to a struct that matches or nil if no row is found
//
// Returns an error if SetKeys has not been called on the TableMap
// Panics if any interface in the list has not been registered with AddTable
func (m *DbMap) Get(i interface{}, keys ...interface{}) (interface{}, error) {
	return get(m, m, i, keys...)
}

// Select runs an arbitrary SQL query, binding the columns in the result
// to fields on the struct specified by i.  args represent the bind 
// parameters for the SQL statement.
//
// Column names on the SELECT statement should be aliased to the field names
// on the struct i. Returns an error if one or more columns in the result
// do not match.  It is OK if fields on i are not part of the SQL 
// statement.
//
// Hook function PostGet() will be executed 
// after the SELECT statement if the interface defines them.
//
// Returns a slice of pointers to matching rows of type i.
//
// i does NOT need to be registered with AddTable()
func (m *DbMap) Select(i interface{}, query string, args ...interface{}) ([]interface{}, error) {
	return rawselect(m, m, i, query, args...)
}

// Exec runs an arbitrary SQL statement.  args represent the bind parameters.
// This is equivalent to running:  Prepare(), Exec() using exp/sql
func (m *DbMap) Exec(query string, args ...interface{}) (sql.Result, error) {
	m.trace(query, args)
	stmt, err := m.Db.Prepare(query)
	if err != nil {
		return nil, err
	}
	return stmt.Exec(args...)
}

// Begin starts a gorp Transaction
func (m *DbMap) Begin() (*Transaction, error) {
	tx, err := m.Db.Begin()
	if err != nil {
		return nil, err
	}
	return &Transaction{m, tx}, nil
}

func (m *DbMap) tableFor(t reflect.Type, checkPK bool) (*TableMap, error) {
	for i := range m.tables {
		table := m.tables[i]
		if table.gotype == t {
			if checkPK && len(table.keys) < 1 {
				e := fmt.Sprintf("gorp: No keys defined for table: %s",
					table.TableName)
				return nil, errors.New(e)
			}
			return table, nil
		}
	}
	panic(fmt.Sprintf("No table found for type: %v", t.Name()))
}

func (m *DbMap) tableForPointer(ptr interface{}, checkPK bool) (*TableMap, reflect.Value, error) {
	ptrv := reflect.ValueOf(ptr)
	if ptrv.Kind() != reflect.Ptr {
		e := fmt.Sprintf("gorp: passed non-pointer: %v (kind=%v)", ptr,
			ptrv.Kind())
		return nil, reflect.Value{}, errors.New(e)
	}
	elem := ptrv.Elem()
	etype := reflect.TypeOf(elem.Interface())
	t, err := m.tableFor(etype, checkPK)
	if err != nil {
		return nil, reflect.Value{}, err
	}

	return t, elem, nil
}

func (m *DbMap) queryRow(query string, args ...interface{}) *sql.Row {
	m.trace(query, args)
	return m.Db.QueryRow(query, args...)
}

func (m *DbMap) query(query string, args ...interface{}) (*sql.Rows, error) {
	m.trace(query, args)
	return m.Db.Query(query, args...)
}

func (m *DbMap) trace(query string, args ...interface{}) {
	if m.logger != nil {
		m.logger.Printf("%s%s %v", m.logPrefix, query, args)
	}
}

///////////////

// Same behavior as DbMap.Insert(), but runs in a transaction
func (t *Transaction) Insert(list ...interface{}) error {
	return insert(t.dbmap, t, list...)
}

// Same behavior as DbMap.Update(), but runs in a transaction
func (t *Transaction) Update(list ...interface{}) (int64, error) {
	return update(t.dbmap, t, list...)
}

// Same behavior as DbMap.Delete(), but runs in a transaction
func (t *Transaction) Delete(list ...interface{}) (int64, error) {
	return delete(t.dbmap, t, list...)
}

// Same behavior as DbMap.Get(), but runs in a transaction
func (t *Transaction) Get(i interface{}, keys ...interface{}) (interface{}, error) {
	return get(t.dbmap, t, i, keys...)
}

// Same behavior as DbMap.Select(), but runs in a transaction
func (t *Transaction) Select(i interface{}, query string, args ...interface{}) ([]interface{}, error) {
	return rawselect(t.dbmap, t, i, query, args...)
}

// Same behavior as DbMap.Exec(), but runs in a transaction
func (t *Transaction) Exec(query string, args ...interface{}) (sql.Result, error) {
	t.dbmap.trace(query, args)
	stmt, err := t.tx.Prepare(query)
	if err != nil {
		return nil, err
	}
	return stmt.Exec(args...)
}

// Commits the underlying database transaction
func (t *Transaction) Commit() error {
	return t.tx.Commit()
}

// Rolls back the underlying database transaction
func (t *Transaction) Rollback() error {
	return t.tx.Rollback()
}

func (t *Transaction) queryRow(query string, args ...interface{}) *sql.Row {
	t.dbmap.trace(query, args)
	return t.tx.QueryRow(query, args...)
}

func (t *Transaction) query(query string, args ...interface{}) (*sql.Rows, error) {
	t.dbmap.trace(query, args)
	return t.tx.Query(query, args...)
}

///////////////

func rawselect(m *DbMap, exec SqlExecutor, i interface{}, query string,
	args ...interface{}) ([]interface{}, error) {

	// Run the query
	rows, err := exec.query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Fetch the column names as returned from db
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	t := reflect.TypeOf(i)

	list := make([]interface{}, 0)

	for rows.Next() {
		v := reflect.New(t)
		dest := make([]interface{}, len(cols))

		// Loop over column names and find field in i to bind to
		// based on column name. all returned columns must match
		// a field in the i struct
		for x := range cols {
			fieldName := cols[x]
			f := v.Elem().FieldByName(fieldName)
			if f == zeroVal {
				e := fmt.Sprintf("gorp: No field %s in type %s (query: %s)",
					fieldName, t.Name(), query)
				return nil, errors.New(e)
			} else {
				dest[x] = f.Addr().Interface()
			}
		}

		err = rows.Scan(dest...)
		if err != nil {
			return nil, err
		}

		err = runHook("PostGet", v, hookArg(exec))
		if err != nil {
			return nil, err
		}

		list = append(list, v.Interface())
	}

	return list, nil
}

func get(m *DbMap, exec SqlExecutor, i interface{},
	keys ...interface{}) (interface{}, error) {

	t := reflect.TypeOf(i)
	table, err := m.tableFor(t, true)
	if err != nil {
		return nil, err
	}

	v := reflect.New(t)
	dest := make([]interface{}, 0)

	s := bytes.Buffer{}
	s.WriteString("select ")

	x := 0
	for _, col := range table.columns {
		if !col.Transient {
			if x > 0 {
				s.WriteString(",")
			}
			s.WriteString(col.ColumnName)

			f := v.Elem().FieldByName(col.fieldName)
			dest = append(dest, f.Addr().Interface())
			x++
		}
	}
	s.WriteString(" from ")
	s.WriteString(table.TableName)
	s.WriteString(" where ")
	for x := range table.keys {
		col := table.keys[x]
		if x > 0 {
			s.WriteString(" and ")
		}
		s.WriteString(col.ColumnName)
		s.WriteString("=?")
	}
	s.WriteString(";")

	sqlstr := s.String()
	m.trace(sqlstr, keys)
	row := exec.queryRow(sqlstr, keys...)
	err = row.Scan(dest...)
	if err != nil {
		if err == sql.ErrNoRows {
			err = nil
		}
		return nil, err
	}

	err = runHook("PostGet", v, hookArg(exec))
	if err != nil {
		return nil, err
	}

	return v.Interface(), nil
}

func delete(m *DbMap, exec SqlExecutor, list ...interface{}) (int64, error) {
	hookarg := hookArg(exec)
	count := int64(0)
	for _, ptr := range list {
		table, elem, err := m.tableForPointer(ptr, true)
		if err != nil {
			return -1, err
		}

		eptr := elem.Addr()
		err = runHook("PreDelete", eptr, hookarg)
		if err != nil {
			return -1, err
		}

		args := make([]interface{}, 0)
		s := bytes.Buffer{}
		s.WriteString("delete from ")
		s.WriteString(table.TableName)
		s.WriteString(" where ")
		for x := range table.keys {
			k := table.keys[x]
			if x > 0 {
				s.WriteString(" and ")
			}
			s.WriteString(k.ColumnName)
			s.WriteString("=?")

			args = append(args, elem.FieldByName(k.fieldName).Interface())
		}
		s.WriteString(";")
		res, err := exec.Exec(s.String(), args...)
		if err != nil {
			return -1, err
		}
		rows, err := res.RowsAffected()
		if err != nil {
			return -1, err
		}
		count += rows

		err = runHook("PostDelete", eptr, hookarg)
		if err != nil {
			return -1, err
		}
	}

	return count, nil
}

func update(m *DbMap, exec SqlExecutor, list ...interface{}) (int64, error) {
	hookarg := hookArg(exec)
	count := int64(0)
	for _, ptr := range list {
		table, elem, err := m.tableForPointer(ptr, true)
		if err != nil {
			return -1, err
		}

		eptr := elem.Addr()
		err = runHook("PreUpdate", eptr, hookarg)
		if err != nil {
			return -1, err
		}

		args := make([]interface{}, 0)
		s := bytes.Buffer{}
		s.WriteString("update ")
		s.WriteString(table.TableName)
		s.WriteString(" set ")
		x := 0
		for y := range table.columns {
			col := table.columns[y]
			if !col.isPK && !col.Transient {
				if x > 0 {
					s.WriteString(", ")
				}
				s.WriteString(col.ColumnName)
				s.WriteString("=?")

				args = append(args, elem.FieldByName(col.fieldName).Interface())
				x++
			}
		}
		s.WriteString(" where ")
		for y := range table.keys {
			col := table.keys[y]
			if y > 0 {
				s.WriteString(" and ")
			}
			s.WriteString(col.ColumnName)
			s.WriteString("=?")
			args = append(args, elem.FieldByName(col.fieldName).Interface())
			x++
		}
		s.WriteString(";")

		res, err := exec.Exec(s.String(), args...)
		if err != nil {
			return -1, err
		}

		rows, err := res.RowsAffected()
		if err != nil {
			return -1, err
		}
		count += rows

		err = runHook("PostUpdate", eptr, hookarg)
		if err != nil {
			return -1, err
		}
	}
	return count, nil
}

func insert(m *DbMap, exec SqlExecutor, list ...interface{}) error {
	hookarg := hookArg(exec)
	for _, ptr := range list {
		table, elem, err := m.tableForPointer(ptr, false)
		if err != nil {
			return err
		}

		eptr := elem.Addr()
		err = runHook("PreInsert", eptr, hookarg)
		if err != nil {
			return err
		}

		args := make([]interface{}, 0)
		s := bytes.Buffer{}
		s2 := bytes.Buffer{}
		s.WriteString(fmt.Sprintf("insert into %s (", table.TableName))
		autoIncrIdx := -1
		x := 0
		for y := range table.columns {
			col := table.columns[y]
			if col.isAutoIncr {
				autoIncrIdx = y
			} else if !col.Transient {
				if x > 0 {
					s.WriteString(",")
					s2.WriteString(",")
				}
				s.WriteString(col.ColumnName)
				s2.WriteString("?")

				args = append(args, elem.FieldByName(col.fieldName).Interface())
				x++
			}
		}
		s.WriteString(") values (")
		s.WriteString(s2.String())
		s.WriteString(");")
		res, err := exec.Exec(s.String(), args...)
		if err != nil {
			return err
		}

		if autoIncrIdx > -1 {
			id, err := res.LastInsertId()
			if err != nil {
				return err
			}
			elem.Field(autoIncrIdx).SetInt(id)
		}

		err = runHook("PostInsert", eptr, hookarg)
		if err != nil {
			return err
		}
	}
	return nil
}

func hookArg(exec SqlExecutor) []reflect.Value {
	execval := reflect.ValueOf(exec)
	return []reflect.Value{execval}
}

func runHook(name string, eptr reflect.Value, arg []reflect.Value) error {
	hook := eptr.MethodByName(name)
	if hook != zeroVal {
		ret := hook.Call(arg)
		if len(ret) > 0 && !ret[0].IsNil() {
			return ret[0].Interface().(error)
		}
	}
	return nil
}
