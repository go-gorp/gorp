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

type DbMap struct {
	Db        *sql.DB
	Dialect   Dialect
	tables    []*TableMap
	logger    *log.Logger
	logPrefix string
}

type TableMap struct {
	gotype  reflect.Type
	TableName string
	columns []*ColumnMap
	keys    []*ColumnMap
}

func (t *TableMap) SetKeys(isAutoIncr bool, propnames ...string) *TableMap {
	t.keys = make([]*ColumnMap, 0)
	for _, name := range propnames {
		colmap := t.ColMap(name)
		colmap.isPK = true
		colmap.isAutoIncr = isAutoIncr
		t.keys = append(t.keys, colmap)
	}

	return t
}

func (t *TableMap) ColMap(prop string) *ColumnMap {
	for _, col := range t.columns {
		if col.propName == prop {
			return col
		} 
	}

	e := fmt.Sprintf("No ColumnMap in table %s type %s with property %s",
		t.TableName, t.gotype.Name(), prop)
	panic(e)
}

type ColumnMap struct {
	ColumnName       string
	Transient bool
	Nullable bool
	Unique bool
	MaxSize int
	propName string
	gotype     reflect.Type
	isPK       bool
	isAutoIncr bool
}

func (c *ColumnMap) Rename(colname string) *ColumnMap { 
	c.ColumnName = colname
	return c
}

func (c *ColumnMap) SetTransient(b bool) *ColumnMap { 
	c.Transient = b
	return c
}

func (c *ColumnMap) SetNullable(b bool) *ColumnMap { 
	c.Nullable = b
	return c
}

func (c *ColumnMap) SetUnique(b bool) *ColumnMap { 
	c.Unique = b
	return c
}

func (c *ColumnMap) SetMaxSize(size int) *ColumnMap { 
	c.MaxSize = size
	return c
}

type Transaction struct {
	dbmap *DbMap
	tx    *sql.Tx
}

type SqlExecutor interface {
	Get(i interface{}, keys ...interface{}) (interface{}, error)
	Insert(list ...interface{}) error
	Update(list ...interface{}) error
	Delete(list ...interface{}) (int64, error)
	Exec(query string, args ...interface{}) (sql.Result, error)
	Select(i interface{}, query string, 
		args ...interface{}) ([]interface{}, error)
	query(query string, args ...interface{}) (*sql.Rows, error)
	queryRow(query string, args ...interface{}) *sql.Row
}

func (m *DbMap) TraceOn(prefix string, logger *log.Logger) {
	m.logger = logger
	if prefix == "" {
		m.logPrefix = prefix
	} else {
		m.logPrefix = fmt.Sprintf("%s ", prefix)
	}
}

func (m *DbMap) TraceOff() {
	m.logger = nil
	m.logPrefix = ""
}

func (m *DbMap) AddTable(i interface{}) *TableMap {
	return m.AddTableWithName(i, "")
}

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
			ColumnName:    f.Name,
		Nullable: true,
		propName : f.Name,
			gotype:  f.Type,
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

func (m *DbMap) Insert(list ...interface{}) error {
	return insert(m, m, list...)
}

func (m *DbMap) Update(list ...interface{}) error {
	return update(m, m, list...)
}

func (m *DbMap) Delete(list ...interface{}) (int64, error) {
	return delete(m, m, list...)
}

func (m *DbMap) Get(i interface{}, keys ...interface{}) (interface{}, error) {
	return get(m, m, i, keys...)
}

func (m *DbMap) Select(i interface{}, query string, args ...interface{}) ([]interface{}, error) {
	return rawselect(m, m, i, query, args...)
}

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

func (m *DbMap) Exec(query string, args ...interface{}) (sql.Result, error) {
	m.trace(query, args)
	stmt, err := m.Db.Prepare(query); if err != nil {
		return nil, err
	}
	return stmt.Exec(args...)
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

func (t *Transaction) Insert(list ...interface{}) error {
	return insert(t.dbmap, t, list...)
}

func (t *Transaction) Update(list ...interface{}) error {
	return update(t.dbmap, t, list...)
}

func (t *Transaction) Delete(list ...interface{}) (int64, error) {
	return delete(t.dbmap, t, list...)
}

func (t *Transaction) Get(i interface{}, keys ...interface{}) (interface{}, error) {
	return get(t.dbmap, t, i, keys...)
}

func (t *Transaction) Select(i interface{}, query string, args ...interface{}) ([]interface{}, error) {
	return rawselect(t.dbmap, t, i, query, args...)
}

func (t *Transaction) Commit() error {
	return t.tx.Commit()
}

func (t *Transaction) Rollback() error {
	return t.tx.Rollback()
}

func (t *Transaction) Exec(query string, args ...interface{}) (sql.Result, error) {
	t.dbmap.trace(query, args)
	stmt, err := t.tx.Prepare(query); if err != nil {
		return nil, err
	}
	return stmt.Exec(args...)
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
		// a property in the i struct
		for x := range cols {
			propName := cols[x]
			f := v.Elem().FieldByName(propName)
			if f == zeroVal {
				e := fmt.Sprintf("gorp: No prop %s in type %s (query: %s)",
					propName, t.Name(), query)
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
			
			f := v.Elem().FieldByName(col.propName)
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

			args = append(args, elem.FieldByName(k.propName).Interface())
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

func update(m *DbMap, exec SqlExecutor, list ...interface{}) error {
	hookarg := hookArg(exec)
	for _, ptr := range list {
		table, elem, err := m.tableForPointer(ptr, true)
		if err != nil {
			return err
		}

		eptr := elem.Addr()
		err = runHook("PreUpdate", eptr, hookarg)
		if err != nil {
			return err
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

				args = append(args, elem.FieldByName(col.propName).Interface())
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
			args = append(args, elem.FieldByName(col.propName).Interface())
			x++
		}
		s.WriteString(";")

		_, err = exec.Exec(s.String(), args...)
		if err != nil {
			return err
		}

		err = runHook("PostUpdate", eptr, hookarg)
		if err != nil {
			return err
		}
	}
	return nil
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

				args = append(args, elem.FieldByName(col.propName).Interface())
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
