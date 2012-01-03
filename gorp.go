package gorp

import (
    "reflect"
    "exp/sql"
    "fmt"
    "bytes"
)

type DbMap struct {
    Db           *sql.DB
    tables       []*TableMap
}

type TableMap struct {
    gotype      reflect.Type
    Name        string
    Columns     []*ColumnMap
}

type ColumnMap struct {
    gotype      reflect.Type
    Name        string
    SqlType     string
}

func (m *DbMap) AddTable(i interface{}) *TableMap {
    t := reflect.TypeOf(i)
    tmap := &TableMap{gotype: t, Name: t.Name() }

    n := t.NumField()
    tmap.Columns = make([]*ColumnMap, n, n)
    for i := 0; i < n; i++ {
        f := t.Field(i)
        tmap.Columns[i] = &ColumnMap{
            gotype : f.Type, 
            Name : f.Name, 
            SqlType : ValueToSqlType(f.Type),
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
        for x := range table.Columns {
            col := table.Columns[x]
            if x > 0 {
                fmt.Fprint(buffer, ", ")
            }
            fmt.Fprintf(buffer, "    %s %s\n", col.Name, col.SqlType)
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

func (m *DbMap) Put(i interface{}) error {
    table := m.TableFor(reflect.TypeOf(i))
    args := make([]interface{}, len(table.Columns))
    buffer := bytes.NewBufferString("")
    fmt.Fprintf(buffer, "insert into %s (", table.Name)
	v := reflect.ValueOf(i)
    for x := range table.Columns {
        col := table.Columns[x]
        if x > 0 {
            fmt.Fprint(buffer, ", ")
        }
        fmt.Fprint(buffer, col.Name)
		fmt.Printf("  Put()  col.Name=%s col.gotype=%v\n", col.Name, col.gotype)
        args[x] = v.FieldByName(col.Name).Interface()
    }
    fmt.Fprint(buffer, ") values (")
    for x := range table.Columns {
        if x > 0 {
            fmt.Fprint(buffer, ", ")
        }
        fmt.Fprint(buffer, "?")
    }
    fmt.Fprint(buffer, ");")
	fmt.Printf(" sql: %s %v\n", buffer.String(), args)
    _, err := m.Db.Exec(buffer.String(), args...)
    return err
}

func (m *DbMap) Get(key interface{}, i interface{}) (interface{}, error) {
	t := reflect.TypeOf(i)
    table := m.TableFor(t)

	v := reflect.New(t)

	sql := bytes.Buffer{}
	sql.WriteString("select ")
	dest := make([]interface{}, len(table.Columns))
	for x := range table.Columns {
		col := table.Columns[x]
		if x > 0 {
			sql.WriteString(",")
		}
		sql.WriteString(col.Name)
		//f := reflect.Indirect(v).FieldByName(col.Name)
		//fmt.Printf("getting Addr of: %v\n", f)
		//dest[x] = f.Addr()

		var tmp string

		dest[x] = v.Elem().FieldByName(col.Name).Addr().Interface()
		fmt.Printf("type of dest[x]: %v to %v - %v\n", reflect.ValueOf(dest[x]).Kind(),
			reflect.Indirect(v.Elem().FieldByName(col.Name).Addr()).Kind(),
			reflect.ValueOf(&tmp).Kind())
	}
	sql.WriteString(fmt.Sprintf(" from %s where Id=?", table.Name))
	fmt.Printf("running: %s %v\n" , sql.String(), dest)
	row := m.Db.QueryRow(sql.String(), key)
	//var v1, v2, v3 int
	err := row.Scan(dest...); if err != nil {
		return nil, err
	}
	fmt.Printf("post: %s %v\n" , sql.String(), reflect.ValueOf(dest[0]).Elem().Interface())

	return v.Elem().Interface(), nil
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

func ValueToSqlType(val reflect.Type) string {
    switch (val.Kind()) {
    case reflect.Int, reflect.Int16, reflect.Int32:
        return "int"
    case reflect.Int64:
        return "bigint"
    }

    return "varchar(255)"
}
