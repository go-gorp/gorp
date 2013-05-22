package gorp

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	_ "github.com/bmizerany/pq"
	_ "github.com/mattn/go-sqlite3"
	_ "github.com/ziutek/mymysql/godrv"
	"log"
	"os"
	"reflect"
	"testing"
	"time"
)

type Invoice struct {
	Id       int64
	Created  int64
	Updated  int64
	Memo     string
	PersonId int64
	IsPaid   bool
}

type Person struct {
	Id      int64
	Created int64
	Updated int64
	FName   string
	LName   string
	Version int64
}

type InvoicePersonView struct {
	InvoiceId     int64
	PersonId      int64
	Memo          string
	FName         string
	LegacyVersion int64
}

type TableWithNull struct {
	Id      int64
	Str     sql.NullString
	Int64   sql.NullInt64
	Float64 sql.NullFloat64
	Bool    sql.NullBool
	Bytes   []byte
}

type WithIgnoredColumn struct {
	internal int64 `db:"-"`
	Id       int64
	Created  int64
}

type WithStringPk struct {
	Id   string
	Desc string
}

type CustomStringType string

type TypeConversionExample struct {
	Id         int64
	PersonJSON Person
	Name       CustomStringType
}

type testTypeConverter struct{}

func (me testTypeConverter) ToDb(val interface{}) (interface{}, error) {

	switch t := val.(type) {
	case Person:
		b, err := json.Marshal(t)
		if err != nil {
			return "", err
		}
		return string(b), nil
	case CustomStringType:
		return string(t), nil
	}

	return val, nil
}

func (me testTypeConverter) FromDb(target interface{}) (CustomScanner, bool) {
	switch target.(type) {
	case *Person:
		binder := func(holder, target interface{}) error {
			s, ok := holder.(*string)
			if !ok {
				return errors.New("FromDb: Unable to convert Person to *string")
			}
			b := []byte(*s)
			return json.Unmarshal(b, target)
		}
		return CustomScanner{new(string), target, binder}, true
	case *CustomStringType:
		binder := func(holder, target interface{}) error {
			s, ok := holder.(*string)
			if !ok {
				return errors.New("FromDb: Unable to convert CustomStringType to *string")
			}
			st, ok := target.(*CustomStringType)
			if !ok {
				return errors.New(fmt.Sprint("FromDb: Unable to convert target to *CustomStringType: ", reflect.TypeOf(target)))
			}
			*st = CustomStringType(*s)
			return nil
		}
		return CustomScanner{new(string), target, binder}, true
	}

	return CustomScanner{}, false
}

func (p *Person) PreInsert(s SqlExecutor) error {
	p.Created = time.Now().UnixNano()
	p.Updated = p.Created
	if p.FName == "badname" {
		return errors.New(fmt.Sprintf("Invalid name: %s", p.FName))
	}
	return nil
}

func (p *Person) PostInsert(s SqlExecutor) error {
	p.LName = "postinsert"
	return nil
}

func (p *Person) PreUpdate(s SqlExecutor) error {
	p.FName = "preupdate"
	return nil
}

func (p *Person) PostUpdate(s SqlExecutor) error {
	p.LName = "postupdate"
	return nil
}

func (p *Person) PreDelete(s SqlExecutor) error {
	p.FName = "predelete"
	return nil
}

func (p *Person) PostDelete(s SqlExecutor) error {
	p.LName = "postdelete"
	return nil
}

func (p *Person) PostGet(s SqlExecutor) error {
	p.LName = "postget"
	return nil
}

type PersistentUser struct {
	Key            int32
	Id             string
	PassedTraining bool
}

func TestCreateTablesIfNotExists(t *testing.T) {
	dbmap := initDbMap()
	defer dbmap.DropTables()

	err := dbmap.CreateTablesIfNotExists()
	if err != nil {
		t.Error(err)
	}
}

func TestPersistentUser(t *testing.T) {
	dbmap := newDbMap()
	dbmap.Exec("drop table if exists PersistentUser")
	dbmap.TraceOn("", log.New(os.Stdout, "gorptest: ", log.Lmicroseconds))
	table := dbmap.AddTable(PersistentUser{}).SetKeys(false, "Key")
	table.ColMap("Key").Rename("mykey")
	err := dbmap.CreateTablesIfNotExists()
	if err != nil {
		panic(err)
	}
	defer dbmap.DropTables()
	pu := &PersistentUser{43, "33r", false}
	err = dbmap.Insert(pu)
	if err != nil {
		panic(err)
	}

	// prove we can pass a pointer into Get
	pu2, err := dbmap.Get(pu, pu.Key)
	if err != nil {
		panic(err)
	}
	if !reflect.DeepEqual(pu, pu2) {
		t.Errorf("%v!=%v", pu, pu2)
	}

	arr, err := dbmap.Select(pu, "select * from PersistentUser")
	if err != nil {
		panic(err)
	}
	if !reflect.DeepEqual(pu, arr[0]) {
		t.Errorf("%v!=%v", pu, arr[0])
	}
}

func TestOverrideVersionCol(t *testing.T) {
	dbmap := initDbMap()
	dbmap.DropTables()
	t1 := dbmap.AddTable(InvoicePersonView{}).SetKeys(false, "InvoiceId", "PersonId")
	err := dbmap.CreateTables()

	if err != nil {
		panic(err)
	}
	defer dbmap.DropTables()
	c1 := t1.SetVersionCol("LegacyVersion")
	if c1.ColumnName != "LegacyVersion" {
		t.Errorf("Wrong col returned: %v", c1)
	}

	ipv := &InvoicePersonView{1, 2, "memo", "fname", 0}
	_update(dbmap, ipv)
	if ipv.LegacyVersion != 1 {
		t.Errorf("LegacyVersion not updated: %d", ipv.LegacyVersion)
	}
}

func TestOptimisticLocking(t *testing.T) {
	dbmap := initDbMap()
	defer dbmap.DropTables()

	p1 := &Person{0, 0, 0, "Bob", "Smith", 0}
	dbmap.Insert(p1) // Version is now 1
	if p1.Version != 1 {
		t.Errorf("Insert didn't incr Version: %d != %d", 1, p1.Version)
		return
	}
	if p1.Id == 0 {
		t.Errorf("Insert didn't return a generated PK")
		return
	}

	obj, err := dbmap.Get(Person{}, p1.Id)
	if err != nil {
		panic(err)
	}
	p2 := obj.(*Person)
	p2.LName = "Edwards"
	dbmap.Update(p2) // Version is now 2
	if p2.Version != 2 {
		t.Errorf("Update didn't incr Version: %d != %d", 2, p2.Version)
	}

	p1.LName = "Howard"
	count, err := dbmap.Update(p1)
	if _, ok := err.(OptimisticLockError); !ok {
		t.Errorf("update - Expected OptimisticLockError, got: %v", err)
	}
	if count != -1 {
		t.Errorf("update - Expected -1 count, got: %d", count)
	}

	count, err = dbmap.Delete(p1)
	if _, ok := err.(OptimisticLockError); !ok {
		t.Errorf("delete - Expected OptimisticLockError, got: %v", err)
	}
	if count != -1 {
		t.Errorf("delete - Expected -1 count, got: %d", count)
	}
}

// what happens if a legacy table has a null value?
func TestDoubleAddTable(t *testing.T) {
	dbmap := newDbMap()
	t1 := dbmap.AddTable(TableWithNull{}).SetKeys(false, "Id")
	t2 := dbmap.AddTable(TableWithNull{})
	if t1 != t2 {
		t.Errorf("%v != %v", t1, t2)
	}
}

// what happens if a legacy table has a null value?
func TestNullValues(t *testing.T) {
	dbmap := initDbMapNulls()
	defer dbmap.DropTables()

	// insert a row directly
	_rawexec(dbmap, "insert into TableWithNull values (10, null, "+
		"null, null, null, null)")

	// try to load it
	expected := &TableWithNull{Id: 10}
	obj := _get(dbmap, TableWithNull{}, 10)
	t1 := obj.(*TableWithNull)
	if !reflect.DeepEqual(expected, t1) {
		t.Errorf("%v != %v", expected, t1)
	}

	// update it
	t1.Str = sql.NullString{"hi", true}
	expected.Str = t1.Str
	t1.Int64 = sql.NullInt64{999, true}
	expected.Int64 = t1.Int64
	t1.Float64 = sql.NullFloat64{53.33, true}
	expected.Float64 = t1.Float64
	t1.Bool = sql.NullBool{true, true}
	expected.Bool = t1.Bool
	t1.Bytes = []byte{1, 30, 31, 33}
	expected.Bytes = t1.Bytes
	_update(dbmap, t1)

	obj = _get(dbmap, TableWithNull{}, 10)
	t1 = obj.(*TableWithNull)
	if t1.Str.String != "hi" {
		t.Errorf("%s != hi", t1.Str.String)
	}
	if !reflect.DeepEqual(expected, t1) {
		t.Errorf("%v != %v", expected, t1)
	}
}

func TestColumnProps(t *testing.T) {
	dbmap := newDbMap()
	dbmap.TraceOn("", log.New(os.Stdout, "gorptest: ", log.Lmicroseconds))
	t1 := dbmap.AddTable(Invoice{}).SetKeys(true, "Id")
	t1.ColMap("Created").Rename("date_created")
	t1.ColMap("Updated").SetTransient(true)
	t1.ColMap("Memo").SetMaxSize(10)
	t1.ColMap("PersonId").SetUnique(true)

	err := dbmap.CreateTables()
	if err != nil {
		panic(err)
	}
	defer dbmap.DropTables()

	// test transient
	inv := &Invoice{0, 0, 1, "my invoice", 0, true}
	_insert(dbmap, inv)
	obj := _get(dbmap, Invoice{}, inv.Id)
	inv = obj.(*Invoice)
	if inv.Updated != 0 {
		t.Errorf("Saved transient column 'Updated'")
	}

	// test max size
	inv.Memo = "this memo is too long"
	err = dbmap.Insert(inv)
	if err == nil {
		t.Errorf("max size exceeded, but Insert did not fail.")
	}

	// test unique - same person id
	inv = &Invoice{0, 0, 1, "my invoice2", 0, false}
	err = dbmap.Insert(inv)
	if err == nil {
		t.Errorf("same PersonId inserted, but Insert did not fail.")
	}
}

func TestRawSelect(t *testing.T) {
	dbmap := initDbMap()
	defer dbmap.DropTables()

	p1 := &Person{0, 0, 0, "bob", "smith", 0}
	_insert(dbmap, p1)

	inv1 := &Invoice{0, 0, 0, "xmas order", p1.Id, true}
	_insert(dbmap, inv1)

	expected := &InvoicePersonView{inv1.Id, p1.Id, inv1.Memo, p1.FName, 0}

	query := "select i.Id InvoiceId, p.Id PersonId, i.Memo, p.FName " +
		"from invoice_test i, person_test p " +
		"where i.PersonId = p.Id"
	list := _rawselect(dbmap, InvoicePersonView{}, query)
	if len(list) != 1 {
		t.Errorf("len(list) != 1: %d", len(list))
	} else if !reflect.DeepEqual(expected, list[0]) {
		t.Errorf("%v != %v", expected, list[0])
	}
}

func TestHooks(t *testing.T) {
	dbmap := initDbMap()
	defer dbmap.DropTables()

	p1 := &Person{0, 0, 0, "bob", "smith", 0}
	_insert(dbmap, p1)
	if p1.Created == 0 || p1.Updated == 0 {
		t.Errorf("p1.PreInsert() didn't run: %v", p1)
	} else if p1.LName != "postinsert" {
		t.Errorf("p1.PostInsert() didn't run: %v", p1)
	}

	obj := _get(dbmap, Person{}, p1.Id)
	p1 = obj.(*Person)
	if p1.LName != "postget" {
		t.Errorf("p1.PostGet() didn't run: %v", p1)
	}

	_update(dbmap, p1)
	if p1.FName != "preupdate" {
		t.Errorf("p1.PreUpdate() didn't run: %v", p1)
	} else if p1.LName != "postupdate" {
		t.Errorf("p1.PostUpdate() didn't run: %v", p1)
	}

	_del(dbmap, p1)
	if p1.FName != "predelete" {
		t.Errorf("p1.PreDelete() didn't run: %v", p1)
	} else if p1.LName != "postdelete" {
		t.Errorf("p1.PostDelete() didn't run: %v", p1)
	}

	// Test error case
	p2 := &Person{0, 0, 0, "badname", "", 0}
	err := dbmap.Insert(p2)
	if err == nil {
		t.Errorf("p2.PreInsert() didn't return an error")
	}
}

func TestTransaction(t *testing.T) {
	dbmap := initDbMap()
	defer dbmap.DropTables()

	inv1 := &Invoice{0, 100, 200, "t1", 0, true}
	inv2 := &Invoice{0, 100, 200, "t2", 0, false}

	trans, err := dbmap.Begin()
	if err != nil {
		panic(err)
	}
	trans.Insert(inv1, inv2)
	err = trans.Commit()
	if err != nil {
		panic(err)
	}

	obj, err := dbmap.Get(Invoice{}, inv1.Id)
	if err != nil {
		panic(err)
	}
	if !reflect.DeepEqual(inv1, obj) {
		t.Errorf("%v != %v", inv1, obj)
	}
	obj, err = dbmap.Get(Invoice{}, inv2.Id)
	if err != nil {
		panic(err)
	}
	if !reflect.DeepEqual(inv2, obj) {
		t.Errorf("%v != %v", inv2, obj)
	}
}

func TestMultiple(t *testing.T) {
	dbmap := initDbMap()
	defer dbmap.DropTables()

	inv1 := &Invoice{0, 100, 200, "a", 0, false}
	inv2 := &Invoice{0, 100, 200, "b", 0, true}
	_insert(dbmap, inv1, inv2)

	inv1.Memo = "c"
	inv2.Memo = "d"
	_update(dbmap, inv1, inv2)

	count := _del(dbmap, inv1, inv2)
	if count != 2 {
		t.Errorf("%d != 2", count)
	}
}

func TestCrud(t *testing.T) {
	dbmap := initDbMap()
	defer dbmap.DropTables()

	inv := &Invoice{0, 100, 200, "first order", 0, true}

	// INSERT row
	_insert(dbmap, inv)
	if inv.Id == 0 {
		t.Errorf("inv.Id was not set on INSERT")
		return
	}

	// SELECT row
	obj := _get(dbmap, Invoice{}, inv.Id)
	inv2 := obj.(*Invoice)
	if !reflect.DeepEqual(inv, inv2) {
		t.Errorf("%v != %v", inv, inv2)
	}

	// UPDATE row and SELECT
	inv.Memo = "second order"
	inv.Created = 999
	inv.Updated = 11111
	count := _update(dbmap, inv)
	if count != 1 {
		t.Errorf("update 1 != %d", count)
	}
	obj = _get(dbmap, Invoice{}, inv.Id)
	inv2 = obj.(*Invoice)
	if !reflect.DeepEqual(inv, inv2) {
		t.Errorf("%v != %v", inv, inv2)
	}

	// DELETE row
	deleted := _del(dbmap, inv)
	if deleted != 1 {
		t.Errorf("Did not delete row with Id: %d", inv.Id)
		return
	}

	// VERIFY deleted
	obj = _get(dbmap, Invoice{}, inv.Id)
	if obj != nil {
		t.Errorf("Found invoice with id: %d after Delete()", inv.Id)
	}
}

func TestWithIgnoredColumn(t *testing.T) {
	dbmap := initDbMap()
	defer dbmap.DropTables()

	ic := &WithIgnoredColumn{-1, 0, 1}
	_insert(dbmap, ic)
	expected := &WithIgnoredColumn{0, 1, 1}
	ic2 := _get(dbmap, WithIgnoredColumn{}, ic.Id).(*WithIgnoredColumn)

	if !reflect.DeepEqual(expected, ic2) {
		t.Errorf("%v != %v", expected, ic2)
	}
	if _del(dbmap, ic) != 1 {
		t.Errorf("Did not delete row with Id: %d", ic.Id)
		return
	}
	if _get(dbmap, WithIgnoredColumn{}, ic.Id) != nil {
		t.Errorf("Found id: %d after Delete()", ic.Id)
	}
}

func TestTypeConversionExample(t *testing.T) {
	dbmap := initDbMap()
	defer dbmap.DropTables()

	p := Person{FName: "Bob", LName: "Smith"}
	tc := &TypeConversionExample{-1, p, CustomStringType("hi")}
	_insert(dbmap, tc)

	expected := &TypeConversionExample{1, p, CustomStringType("hi")}
	tc2 := _get(dbmap, TypeConversionExample{}, tc.Id).(*TypeConversionExample)
	if !reflect.DeepEqual(expected, tc2) {
		t.Errorf("tc2 %v != %v", expected, tc2)
	}

	tc2.Name = CustomStringType("hi2")
	tc2.PersonJSON = Person{FName: "Jane", LName: "Doe"}
	_update(dbmap, tc2)

	expected = &TypeConversionExample{1, tc2.PersonJSON, CustomStringType("hi2")}
	tc3 := _get(dbmap, TypeConversionExample{}, tc.Id).(*TypeConversionExample)
	if !reflect.DeepEqual(expected, tc3) {
		t.Errorf("tc3 %v != %v", expected, tc3)
	}

	if _del(dbmap, tc) != 1 {
		t.Errorf("Did not delete row with Id: %d", tc.Id)
	}

}

func TestSelectVal(t *testing.T) {
	dbmap := initDbMapNulls()
	defer dbmap.DropTables()

	bindVar := dbmap.Dialect.BindVar(0)

	t1 := TableWithNull{Str: sql.NullString{"abc", true},
		Int64:   sql.NullInt64{78, true},
		Float64: sql.NullFloat64{32.2, true},
		Bool:    sql.NullBool{true, true},
		Bytes:   []byte("hi")}
	_insert(dbmap, &t1)

	// SelectInt
	i64 := selectInt(dbmap, "select Int64 from TableWithNull where Str='abc'")
	if i64 != 78 {
		t.Errorf("int64 %d != 78", i64)
	}
	i64 = selectInt(dbmap, "select count(*) from TableWithNull")
	if i64 != 1 {
		t.Errorf("int64 count %d != 1", i64)
	}
	i64 = selectInt(dbmap, "select count(*) from TableWithNull where Str="+bindVar, "asdfasdf")
	if i64 != 0 {
		t.Errorf("int64 no rows %d != 0", i64)
	}

	// SelectNullInt
	n := selectNullInt(dbmap, "select Int64 from TableWithNull where Str='notfound'")
	if !reflect.DeepEqual(n, sql.NullInt64{0, false}) {
		t.Errorf("nullint %v != 0,false", n)
	}

	n = selectNullInt(dbmap, "select Int64 from TableWithNull where Str='abc'")
	if !reflect.DeepEqual(n, sql.NullInt64{78, true}) {
		t.Errorf("nullint %v != 78, true", n)
	}

	// SelectStr
	s := selectStr(dbmap, "select Str from TableWithNull where Int64="+bindVar, 78)
	if s != "abc" {
		t.Errorf("s %s != abc", s)
	}
	s = selectStr(dbmap, "select Str from TableWithNull where Str='asdfasdf'")
	if s != "" {
		t.Errorf("s no rows %s != ''", s)
	}

	// SelectNullStr
	ns := selectNullStr(dbmap, "select Str from TableWithNull where Int64="+bindVar, 78)
	if !reflect.DeepEqual(ns, sql.NullString{"abc", true}) {
		t.Errorf("nullstr %v != abc,true", ns)
	}
	ns = selectNullStr(dbmap, "select Str from TableWithNull where Str='asdfasdf'")
	if !reflect.DeepEqual(ns, sql.NullString{"", false}) {
		t.Errorf("nullstr no rows %v != '',false", ns)
	}
}

func TestVersionMultipleRows(t *testing.T) {
	dbmap := initDbMap()
	defer dbmap.DropTables()

	persons := []*Person{
		&Person{0, 0, 0, "Bob", "Smith", 0},
		&Person{0, 0, 0, "Jane", "Smith", 0},
		&Person{0, 0, 0, "Mike", "Smith", 0},
	}

	_insert(dbmap, persons[0], persons[1], persons[2])

	for x, p := range persons {
		if p.Version != 1 {
			t.Errorf("person[%d].Version != 1: %d", x, p.Version)
		}
	}
}

/*
func TestWithStringPk(t *testing.T) {
	dbmap := initDbMap()
	defer dbmap.DropTables()

	row := &WithStringPk{"myid", "foo"}
	err := dbmap.Insert(row)
	if err == nil {
		t.Errorf("Expected error when inserting into table w/non Int PK and autoincr set true")
	}
}*/

func BenchmarkNativeCrud(b *testing.B) {
	b.StopTimer()
	dbmap := initDbMapBench()
	defer dbmap.DropTables()
	b.StartTimer()

	insert := "insert into invoice_test (Created, Updated, Memo, PersonId) values (?, ?, ?, ?)"
	sel := "select Id, Created, Updated, Memo, PersonId from invoice_test where Id=?"
	update := "update invoice_test set Created=?, Updated=?, Memo=?, PersonId=? where Id=?"
	delete := "delete from invoice_test where Id=?"

	inv := &Invoice{0, 100, 200, "my memo", 0, false}

	for i := 0; i < b.N; i++ {
		res, err := dbmap.Db.Exec(insert, inv.Created, inv.Updated,
			inv.Memo, inv.PersonId)
		if err != nil {
			panic(err)
		}

		newid, err := res.LastInsertId()
		if err != nil {
			panic(err)
		}
		inv.Id = newid

		row := dbmap.Db.QueryRow(sel, inv.Id)
		err = row.Scan(&inv.Id, &inv.Created, &inv.Updated, &inv.Memo,
			&inv.PersonId)
		if err != nil {
			panic(err)
		}

		inv.Created = 1000
		inv.Updated = 2000
		inv.Memo = "my memo 2"
		inv.PersonId = 3000

		_, err = dbmap.Db.Exec(update, inv.Created, inv.Updated, inv.Memo,
			inv.PersonId, inv.Id)
		if err != nil {
			panic(err)
		}

		_, err = dbmap.Db.Exec(delete, inv.Id)
		if err != nil {
			panic(err)
		}
	}

}

func BenchmarkGorpCrud(b *testing.B) {
	b.StopTimer()
	dbmap := initDbMapBench()
	defer dbmap.DropTables()
	b.StartTimer()

	inv := &Invoice{0, 100, 200, "my memo", 0, true}
	for i := 0; i < b.N; i++ {
		err := dbmap.Insert(inv)
		if err != nil {
			panic(err)
		}

		obj, err := dbmap.Get(Invoice{}, inv.Id)
		if err != nil {
			panic(err)
		}

		inv2, ok := obj.(*Invoice)
		if !ok {
			panic(fmt.Sprintf("expected *Invoice, got: %v", obj))
		}

		inv2.Created = 1000
		inv2.Updated = 2000
		inv2.Memo = "my memo 2"
		inv2.PersonId = 3000
		_, err = dbmap.Update(inv2)
		if err != nil {
			panic(err)
		}

		_, err = dbmap.Delete(inv2)
		if err != nil {
			panic(err)
		}

	}
}

func initDbMapBench() *DbMap {
	dbmap := newDbMap()
	dbmap.Db.Exec("drop table if exists invoice_test")
	dbmap.AddTableWithName(Invoice{}, "invoice_test").SetKeys(true, "Id")
	err := dbmap.CreateTables()
	if err != nil {
		panic(err)
	}
	return dbmap
}

func initDbMap() *DbMap {
	dbmap := newDbMap()
	dbmap.TraceOn("", log.New(os.Stdout, "gorptest: ", log.Lmicroseconds))
	dbmap.AddTableWithName(Invoice{}, "invoice_test").SetKeys(true, "Id")
	dbmap.AddTableWithName(Person{}, "person_test").SetKeys(true, "Id")
	dbmap.AddTableWithName(WithIgnoredColumn{}, "ignored_column_test").SetKeys(true, "Id")
	dbmap.AddTableWithName(WithStringPk{}, "string_pk_test").SetKeys(false, "Id")
	dbmap.AddTableWithName(TypeConversionExample{}, "type_conv_test").SetKeys(true, "Id")
	dbmap.TypeConverter = testTypeConverter{}
	err := dbmap.CreateTables()
	if err != nil {
		panic(err)
	}

	return dbmap
}

func initDbMapNulls() *DbMap {
	dbmap := newDbMap()
	dbmap.TraceOn("", log.New(os.Stdout, "gorptest: ", log.Lmicroseconds))
	dbmap.AddTable(TableWithNull{}).SetKeys(false, "Id")
	err := dbmap.CreateTables()
	if err != nil {
		panic(err)
	}
	return dbmap
}

func newDbMap() *DbMap {
	dialect, driver := dialectAndDriver()
	return &DbMap{Db: connect(driver), Dialect: dialect}
}

func connect(driver string) *sql.DB {
	dsn := os.Getenv("GORP_TEST_DSN")
	if dsn == "" {
		panic("GORP_TEST_DSN env variable is not set. Please see README.md")
	}

	db, err := sql.Open(driver, dsn)
	if err != nil {
		panic("Error connecting to db: " + err.Error())
	}
	return db
}

func dialectAndDriver() (Dialect, string) {
	switch os.Getenv("GORP_TEST_DIALECT") {
	case "mysql":
		return MySQLDialect{"InnoDB", "UTF8"}, "mymysql"
	case "postgres":
		return PostgresDialect{}, "postgres"
	case "sqlite":
		return SqliteDialect{}, "sqlite3"
	}
	panic("GORP_TEST_DIALECT env variable is not set or is invalid. Please see README.md")
}

func _insert(dbmap *DbMap, list ...interface{}) {
	err := dbmap.Insert(list...)
	if err != nil {
		panic(err)
	}
}

func _update(dbmap *DbMap, list ...interface{}) int64 {
	count, err := dbmap.Update(list...)
	if err != nil {
		panic(err)
	}
	return count
}

func _del(dbmap *DbMap, list ...interface{}) int64 {
	count, err := dbmap.Delete(list...)
	if err != nil {
		panic(err)
	}

	return count
}

func _get(dbmap *DbMap, i interface{}, keys ...interface{}) interface{} {
	obj, err := dbmap.Get(i, keys...)
	if err != nil {
		panic(err)
	}

	return obj
}

func selectInt(dbmap *DbMap, query string, args ...interface{}) int64 {
	i64, err := SelectInt(dbmap, query, args...)
	if err != nil {
		panic(err)
	}

	return i64
}

func selectNullInt(dbmap *DbMap, query string, args ...interface{}) sql.NullInt64 {
	i64, err := SelectNullInt(dbmap, query, args...)
	if err != nil {
		panic(err)
	}

	return i64
}

func selectStr(dbmap *DbMap, query string, args ...interface{}) string {
	s, err := SelectStr(dbmap, query, args...)
	if err != nil {
		panic(err)
	}

	return s
}

func selectNullStr(dbmap *DbMap, query string, args ...interface{}) sql.NullString {
	s, err := SelectNullStr(dbmap, query, args...)
	if err != nil {
		panic(err)
	}

	return s
}

func _rawexec(dbmap *DbMap, query string, args ...interface{}) sql.Result {
	res, err := dbmap.Exec(query, args...)
	if err != nil {
		panic(err)
	}
	return res
}

func _rawselect(dbmap *DbMap, i interface{}, query string, args ...interface{}) []interface{} {
	list, err := dbmap.Select(i, query, args...)
	if err != nil {
		panic(err)
	}
	return list
}
