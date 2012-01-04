package gorp

import (
    "reflect"
    "exp/sql"
    "fmt"
    "bytes"
	"log"
	"errors"
)

type DbMap struct {
    Db           *sql.DB
	Dialect      Dialect
    tables       []*TableMap
	logger       *log.Logger
    logPrefix    string
}

type TableMap struct {
    gotype      reflect.Type
    Name        string
    columns     []*ColumnMap
    keys        []*ColumnMap
}

type ColumnMap struct {
    gotype      reflect.Type
    Name        string
    sqlType     string
	isPK        bool
	isAutoIncr  bool
}

func (t *TableMap) SetKeys(isAutoIncr bool, colnames ...string) error {
	t.keys = make([]*ColumnMap, 0)
	for _, colname := range(colnames) {
		found := false
		for i := 0; i < len(t.columns) && !found; i++ {
			if t.columns[i].Name == colname {
				t.columns[i].isPK = true
				t.columns[i].isAutoIncr = isAutoIncr
				t.keys = append(t.keys, t.columns[i])
				found = true
			}
		}
		if !found {
			return errors.New(fmt.Sprintf("gorp: No column with name: %s", 
				colname))		
		}
	}

	return nil
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
    tmap := &TableMap{gotype: t, Name: name }

    n := t.NumField()
    tmap.columns = make([]*ColumnMap, n, n)
    for i := 0; i < n; i++ {
        f := t.Field(i)
        tmap.columns[i] = &ColumnMap{
            gotype : f.Type, 
            Name : f.Name, 
            sqlType : m.Dialect.ToSqlType(f.Type),
        }
    }

    // append to slice
    // expand slice as necessary
    n = len(m.tables)
    if (n+1) > cap(m.tables) { 
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
        s.WriteString(fmt.Sprintf("create table %s (\n", table.Name))
        for x := range table.columns {
            col := table.columns[x]
            if x > 0 {
				s.WriteString(",\n")
            }
            s.WriteString(fmt.Sprintf("    %s %s", col.Name, col.sqlType))

			if col.isPK {
				s.WriteString(" primary key")
			}
			if col.isAutoIncr {
				s.WriteString(fmt.Sprintf(" %s", m.Dialect.AutoIncrStr()))
			}
        }
        s.WriteString(");")
        _, err = m.execSql(s.String())
    }
    return err
}

func (m *DbMap) DropTables() error {
    var err error
    for i := range m.tables {
        table := m.tables[i]
		_, e := m.execSql(fmt.Sprintf("drop table %s;", table.Name))
		if e != nil {
			err = e
		}
    }
    return err
}

func (m *DbMap) Insert(list ...interface{}) error {
	for _, ptr := range(list) {
		table, elem, err := m.tableForPointer(ptr, false); if err != nil {
			return err
		}

		args := make([]interface{}, 0)
		s := bytes.Buffer{}
		s2 := bytes.Buffer{}
		s.WriteString(fmt.Sprintf("insert into %s (", table.Name))
		autoIncrIdx := -1
		x := 0
		for y := range table.columns {
			col := table.columns[y]
			if col.isAutoIncr {
				autoIncrIdx = y
			} else {
				if x > 0 {
					s.WriteString(",")
					s2.WriteString(",")
				}
				s.WriteString(col.Name)
				s2.WriteString("?")

				args = append(args, elem.FieldByName(col.Name).Interface())
				x++
			}
		}
		s.WriteString(") values (")
		s.WriteString(s2.String())
		s.WriteString(");")
		res, err := m.execSql(s.String(), args...); if err != nil {
			return err
		}

		if autoIncrIdx > -1 {
			id, err := res.LastInsertId(); if err != nil {
				return err
			}
			elem.Field(autoIncrIdx).SetInt(id)
		}
	}
	return nil
}

func (m *DbMap) Update(list ...interface{}) error {
	for _, ptr := range(list) {
		table, elem, err := m.tableForPointer(ptr, true); if err != nil {
			return err
		}

		args := make([]interface{}, len(table.columns))
		s := bytes.Buffer{}
		s.WriteString("update ")
		s.WriteString(table.Name)
		s.WriteString(" set ")
		x := 0
		for y := range table.columns {
			col := table.columns[y]
			if !col.isPK {
				if x > 0 {
					s.WriteString(", ")
				}
				s.WriteString(col.Name)
				s.WriteString("=?")

				args[x] = elem.FieldByName(col.Name).Interface()
				x++
			}
		}
		s.WriteString(" where ")
		for y := range table.keys {
			col := table.keys[y]
			if y > 0 {
				s.WriteString(" and ")
			}
			s.WriteString(col.Name)
			s.WriteString("=?")
			args[x] = elem.FieldByName(col.Name).Interface()
			x++
		}
		s.WriteString(";")

		_, err = m.execSql(s.String(), args...); if err != nil {
			return err
		}
	}
	return nil
}

func (m *DbMap) Delete(list ...interface{}) (int64, error) {
	count := int64(0)
	for _, ptr := range(list) {
		table, elem, err := m.tableForPointer(ptr, true); if err != nil {
			return -1, err
		}

		args := make([]interface{}, 0)
		s := bytes.Buffer{}
		s.WriteString("delete from ")
		s.WriteString(table.Name)
		s.WriteString(" where ")
		for x := range table.keys {
			k := table.keys[x]
			if x > 0 {
				s.WriteString(" and ")
			}
			s.WriteString(k.Name)
			s.WriteString("=?")
			
			args = append(args, elem.FieldByName(k.Name).Interface())
		}
		s.WriteString(";")
		res, err := m.execSql(s.String(), args...); if err != nil {
			return -1, err
		}
		rows, err := res.RowsAffected(); if err != nil {
			return -1, err
		}
		count += rows
	}

    return count, nil
}

func (m *DbMap) Get(key interface{}, i interface{}) (interface{}, error) {
	t := reflect.TypeOf(i)
    table := m.TableFor(t)

	v := reflect.New(t)
	dest := make([]interface{}, len(table.columns))

	s := bytes.Buffer{}
	s.WriteString("select ")

	for x := range table.columns {
		col := table.columns[x]
		if x > 0 {
			s.WriteString(",")
		}
		s.WriteString(col.Name)

		dest[x] = v.Elem().FieldByName(col.Name).Addr().Interface()
	}
	s.WriteString(fmt.Sprintf(" from %s where Id=?;", table.Name))

	sqlstr := s.String()
	m.trace(sqlstr, key)
	row := m.Db.QueryRow(sqlstr, key)
	err := row.Scan(dest...); if err != nil {
		if err == sql.ErrNoRows {
			err = nil
		}
		return nil, err
	}
	return v.Interface(), nil
}

func (m *DbMap) TableFor(t reflect.Type) *TableMap {
    for i := range m.tables {
        table := m.tables[i]
        if (table.gotype == t) {
            return table
        }
    }
    panic(fmt.Sprintf("No table found for type: %v", t.Name()))
}

func (m *DbMap) tableForPointer(ptr interface{}, checkPK bool) (*TableMap, reflect.Value, error) {
	ptrv := reflect.ValueOf(ptr)
	if ptrv.Kind() != reflect.Ptr {
		e := fmt.Sprintf("gorp: Update() passed non-pointer: %v", ptr)
		return nil, reflect.Value{}, errors.New(e)
	}
	elem := ptrv.Elem()
	t := m.TableFor(reflect.TypeOf(elem.Interface()))

	if checkPK && len(t.keys) < 1 {
		e := fmt.Sprintf("gorp: No keys defined for table: %s", t.Name)
		return nil, reflect.Value{}, errors.New(e)
	}

	return t, elem, nil
}

func (m *DbMap) execSql(query string, args ...interface{}) (sql.Result, error) {
	m.trace(query, args)
    return m.Db.Exec(query, args...)
}

func (m *DbMap) trace(query string, args ...interface{}) {
	if m.logger != nil {
		m.logger.Printf("%s%s %v", m.logPrefix, query, args)
	}
}

///////////////

type Dialect interface {
	ToSqlType(val reflect.Type) string
	AutoIncrStr() string
}

type MySQLDialect struct { }

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