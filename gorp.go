package gorp

import (
    "reflect"
    "exp/sql"
    "fmt"
    "bytes"
	"errors"
)

type DbMap struct {
    Db           *sql.DB
    tables       []*TableMap
}

type TableMap struct {
    gotype      reflect.Type
    Name        string
    columns     []*ColumnMap
	keys        []*keyColumn
}

type ColumnMap struct {
    gotype      reflect.Type
    Name        string
    sqlType     string
}

type keyColumn struct {
	column        *ColumnMap
	autoIncrement bool
}

func (t *TableMap) SetKeys(autoincr bool, colnames ...string) error {
	for _, colname := range(colnames) {
		found := false
		for i := 0; i < len(t.columns) && !found; i++ {
			if t.columns[i].Name == colname {
				t.keys = append(t.keys, &keyColumn{t.columns[i], autoincr})
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
            sqlType : GoToSqlType(f.Type),
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
        buffer := bytes.NewBufferString("");
        table := m.tables[i]
        fmt.Fprintf(buffer, "create table %s (\n", table.Name)
        for x := range table.columns {
            col := table.columns[x]
            if x > 0 {
                fmt.Fprint(buffer, ", ")
            }
            fmt.Fprintf(buffer, "    %s %s\n", col.Name, col.sqlType)
        }
        fmt.Fprintf(buffer, ");")
        err = execSql(m, buffer)
    }
    return err
}

func (m *DbMap) DropTables() error {
    var err error
    for i := range m.tables {
        table := m.tables[i]
        execSqlStr(m, fmt.Sprintf("drop table %s;", table.Name))
    }
    return err
}

func (m *DbMap) Insert(i interface{}) error {
    table := m.TableFor(reflect.TypeOf(i))
    args := make([]interface{}, len(table.columns))
    buffer := bytes.NewBufferString("")
    fmt.Fprintf(buffer, "insert into %s (", table.Name)
	v := reflect.ValueOf(i)
    for x := range table.columns {
        col := table.columns[x]
        if x > 0 {
            fmt.Fprint(buffer, ", ")
        }
        fmt.Fprint(buffer, col.Name)
        args[x] = v.FieldByName(col.Name).Interface()
    }
    fmt.Fprint(buffer, ") values (")
    for x := range table.columns {
        if x > 0 {
            fmt.Fprint(buffer, ", ")
        }
        fmt.Fprint(buffer, "?")
    }
    fmt.Fprint(buffer, ");")
    _, err := m.Db.Exec(buffer.String(), args...)
    return err
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

	row := m.Db.QueryRow(s.String(), key)
	err := row.Scan(dest...); if err != nil {
		if err == sql.ErrNoRows {
			err = nil
		}
		return nil, err
	}
	return v.Elem().Interface(), nil
}

func (m *DbMap) Delete(i interface{}) (bool, error) {
	t := reflect.TypeOf(i)
    table := m.TableFor(t)

	if len(table.keys) < 1 {
		e := fmt.Sprintf("gorp: No keys defined for table: %s", table.Name)
		return false, errors.New(e)
	}

	v := reflect.ValueOf(i)
	args := make([]interface{}, 0)

	sql := bytes.Buffer{}
	sql.WriteString("delete from ")
	sql.WriteString(table.Name)
	sql.WriteString(" where ")
	for x := range table.keys {
		k := table.keys[x]
		if x > 0 {
			sql.WriteString(" and ")
		}
		sql.WriteString(k.column.Name)
		sql.WriteString("=?")

		args = append(args, v.FieldByName(k.column.Name).Interface())
	}
	sql.WriteString(";")
	res, err := m.Db.Exec(sql.String(), args...); if err != nil {
		return false, err
	}
	rows, err := res.RowsAffected(); if err != nil {
		return false, err
	}
    return rows == 1, nil
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

func execSql(m *DbMap, query *bytes.Buffer) error {
    return execSqlStr(m, query.String())
}

func execSqlStr(m *DbMap, query string) error {
    fmt.Println(query)
    _, err := m.Db.Exec(query)
    return err
}

///////////////

func GoToSqlType(val reflect.Type) string {
    switch (val.Kind()) {
    case reflect.Int, reflect.Int16, reflect.Int32:
        return "int"
    case reflect.Int64:
        return "bigint"
    }

    return "varchar(255)"
}
