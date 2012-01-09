package gorp_test

import (
	"errors"
	"exp/sql"
	"fmt"
	. "github.com/coopernurse/gorp"
	_ "github.com/ziutek/mymysql/godrv"
	"log"
	"os"
	"reflect"
	"testing"
	"time"
)

var dialect = MySQLDialect{"InnoDB", "UTF8"}

type Invoice struct {
	Id       int64
	Created  int64
	Updated  int64
	Memo     string
	PersonId int64
}

type Person struct {
	Id      int64
	Created int64
	Updated int64
	FName   string
	LName   string
}

type InvoicePersonView struct {
	InvoiceId int64
	PersonId  int64
	Memo      string
	FName     string
}

type TableWithNull struct {
	Id      int64
	Str     sql.NullableString
	Int64   sql.NullableInt64
	Float64 sql.NullableFloat64
	Bool    sql.NullableBool
	Bytes   sql.NullableBytes
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

// what happens if a legacy table has a null value?
func TestDoubleAddTable(t *testing.T) {
	dbmap := &DbMap{Db: connect(), Dialect: dialect}
	t1 := dbmap.AddTable(TableWithNull{}).SetKeys(false, "Id")
	t2 := dbmap.AddTable(TableWithNull{})
	if t1 != t2 {
		t.Errorf("%v != %v", t1, t2)
	}
}

// what happens if a legacy table has a null value?
func TestNullValues(t *testing.T) {
	dbmap := &DbMap{Db: connect(), Dialect: dialect}
	dbmap.TraceOn("", log.New(os.Stdout, "gorptest: ", log.Lmicroseconds))
	dbmap.AddTable(TableWithNull{}).SetKeys(false, "Id")
	dbmap.CreateTables()
	defer dbmap.DropTables()

	// insert a row directly
	rawexec(dbmap, "insert into TableWithNull values (10, null, "+
		"null, null, null, null)")

	// try to load it
	expected := &TableWithNull{Id: 10}
	obj := get(dbmap, TableWithNull{}, 10)
	t1 := obj.(*TableWithNull)
	if !reflect.DeepEqual(expected, t1) {
		t.Errorf("%v != %v", expected, t1)
	}

	// update it
	t1.Str = sql.NullableString{"hi", true}
	expected.Str = t1.Str
	t1.Int64 = sql.NullableInt64{999, true}
	expected.Int64 = t1.Int64
	t1.Float64 = sql.NullableFloat64{53.33, true}
	expected.Float64 = t1.Float64
	t1.Bool = sql.NullableBool{true, true}
	expected.Bool = t1.Bool
	t1.Bytes = sql.NullableBytes{[]byte{1, 30, 31, 33}, true}
	expected.Bytes = t1.Bytes
	update(dbmap, t1)

	obj = get(dbmap, TableWithNull{}, 10)
	t1 = obj.(*TableWithNull)
	if t1.Str.String != "hi" {
		t.Errorf("%s != hi", t1.Str.String)
	}
	if !reflect.DeepEqual(expected, t1) {
		t.Errorf("%v != %v", expected, t1)
	}
}

func TestColumnProps(t *testing.T) {
	dbmap := &DbMap{Db: connect(), Dialect: dialect}
	dbmap.TraceOn("", log.New(os.Stdout, "gorptest: ", log.Lmicroseconds))
	t1 := dbmap.AddTable(Invoice{}).SetKeys(true, "Id")
	t1.ColMap("Created").Rename("date_created").SetNullable(false)
	t1.ColMap("Updated").SetTransient(true)
	t1.ColMap("Memo").SetMaxSize(10)
	t1.ColMap("PersonId").SetUnique(true)

	dbmap.CreateTables()
	defer dbmap.DropTables()

	// test transient
	inv := &Invoice{0, 0, 1, "my invoice", 0}
	insert(dbmap, inv)
	obj := get(dbmap, Invoice{}, inv.Id)
	inv = obj.(*Invoice)
	fmt.Printf("inv: %v\n", inv)
	if inv.Updated != 0 {
		t.Errorf("Saved transient column 'Updated'")
	}

	// test max size
	inv.Memo = "this memo is too long"
	err := dbmap.Insert(inv)
	if err == nil {
		t.Errorf("max size exceeded, but Insert did not fail.")
	}

	// test unique - same person id
	inv = &Invoice{0, 0, 1, "my invoice2", 0}
	err = dbmap.Insert(inv)
	if err == nil {
		t.Errorf("same PersonId inserted, but Insert did not fail.")
	}
}

func TestRawSelect(t *testing.T) {
	dbmap := initDbMap()
	defer dbmap.DropTables()

	p1 := &Person{0, 0, 0, "bob", "smith"}
	insert(dbmap, p1)

	inv1 := &Invoice{0, 0, 0, "xmas order", p1.Id}
	insert(dbmap, inv1)

	expected := &InvoicePersonView{inv1.Id, p1.Id, inv1.Memo, p1.FName}

	query := "select i.Id InvoiceId, p.Id PersonId, i.Memo, p.FName " +
		"from invoice_test i, person_test p " +
		"where i.PersonId = p.Id"
	list := rawselect(dbmap, InvoicePersonView{}, query)
	if len(list) != 1 {
		t.Errorf("len(list) != 1: %d", len(list))
	} else if !reflect.DeepEqual(expected, list[0]) {
		t.Errorf("%v != %v", expected, list[0])
	}
}

func TestHooks(t *testing.T) {
	dbmap := initDbMap()
	defer dbmap.DropTables()

	p1 := &Person{0, 0, 0, "bob", "smith"}
	insert(dbmap, p1)
	if p1.Created == 0 || p1.Updated == 0 {
		t.Errorf("p1.PreInsert() didn't run: %v", p1)
	} else if p1.LName != "postinsert" {
		t.Errorf("p1.PostInsert() didn't run: %v", p1)
	}

	obj := get(dbmap, Person{}, p1.Id)
	p1 = obj.(*Person)
	if p1.LName != "postget" {
		t.Errorf("p1.PostGet() didn't run: %v", p1)
	}

	update(dbmap, p1)
	if p1.FName != "preupdate" {
		t.Errorf("p1.PreUpdate() didn't run: %v", p1)
	} else if p1.LName != "postupdate" {
		t.Errorf("p1.PostUpdate() didn't run: %v", p1)
	}

	del(dbmap, p1)
	if p1.FName != "predelete" {
		t.Errorf("p1.PreDelete() didn't run: %v", p1)
	} else if p1.LName != "postdelete" {
		t.Errorf("p1.PostDelete() didn't run: %v", p1)
	}

	// Test error case
	p2 := &Person{0, 0, 0, "badname", ""}
	err := dbmap.Insert(p2)
	if err == nil {
		t.Errorf("p2.PreInsert() didn't return an error")
	}
}

func TestTransaction(t *testing.T) {
	dbmap := initDbMap()
	defer dbmap.DropTables()

	inv1 := &Invoice{0, 100, 200, "t1", 0}
	inv2 := &Invoice{0, 100, 200, "t2", 0}

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

	inv1 := &Invoice{0, 100, 200, "a", 0}
	inv2 := &Invoice{0, 100, 200, "b", 0}
	insert(dbmap, inv1, inv2)

	inv1.Memo = "c"
	inv2.Memo = "d"
	update(dbmap, inv1, inv2)

	count := del(dbmap, inv1, inv2)
	if count != 2 {
		t.Errorf("%d != 2", count)
	}
}

func TestCrud(t *testing.T) {
	dbmap := initDbMap()
	defer dbmap.DropTables()

	inv := &Invoice{0, 100, 200, "first order", 0}

	// INSERT row
	insert(dbmap, inv)
	if inv.Id == 0 {
		t.Errorf("inv.Id was not set on INSERT")
		return
	}

	// SELECT row
	obj := get(dbmap, Invoice{}, inv.Id)
	inv2 := obj.(*Invoice)
	if !reflect.DeepEqual(inv, inv2) {
		t.Errorf("%v != %v", inv, inv2)
	}

	// UPDATE row and SELECT
	inv.Memo = "second order"
	inv.Created = 999
	inv.Updated = 11111
	count := update(dbmap, inv)
	if count != 1 {
		t.Errorf("update 1 != %d", count)
	}
	obj = get(dbmap, Invoice{}, inv.Id)
	inv2 = obj.(*Invoice)
	if !reflect.DeepEqual(inv, inv2) {
		t.Errorf("%v != %v", inv, inv2)
	}

	// DELETE row
	deleted := del(dbmap, inv)
	if deleted != 1 {
		t.Errorf("Did not delete row with Id: %d", inv.Id)
		return
	}

	// VERIFY deleted
	obj = get(dbmap, Invoice{}, inv.Id)
	if obj != nil {
		t.Errorf("Found invoice with id: %d after Delete()", inv.Id)
	}
}

func initDbMap() *DbMap {
	dbmap := &DbMap{Db: connect(), Dialect: dialect}
	dbmap.TraceOn("", log.New(os.Stdout, "gorptest: ", log.Lmicroseconds))
	dbmap.AddTableWithName(Invoice{}, "invoice_test").SetKeys(true, "Id")
	dbmap.AddTableWithName(Person{}, "person_test").SetKeys(true, "Id")
	dbmap.CreateTables()

	return dbmap
}

func connect() *sql.DB {
	db, err := sql.Open("mymysql", "gomysql_test/gomysql_test/abc123")
	if err != nil {
		panic("Error connecting to db: " + err.Error())
	}
	return db
}

func insert(dbmap *DbMap, list ...interface{}) {
	err := dbmap.Insert(list...)
	if err != nil {
		panic(err)
	}
}

func update(dbmap *DbMap, list ...interface{}) int64 {
	count, err := dbmap.Update(list...)
	if err != nil {
		panic(err)
	}
	return count
}

func del(dbmap *DbMap, list ...interface{}) int64 {
	count, err := dbmap.Delete(list...)
	if err != nil {
		panic(err)
	}

	return count
}

func get(dbmap *DbMap, i interface{}, keys ...interface{}) interface{} {
	obj, err := dbmap.Get(i, keys...)
	if err != nil {
		panic(err)
	}

	return obj
}

func rawexec(dbmap *DbMap, query string, args ...interface{}) sql.Result {
	res, err := dbmap.Exec(query, args...)
	if err != nil {
		panic(err)
	}
	return res
}

func rawselect(dbmap *DbMap, i interface{}, query string, args ...interface{}) []interface{} {
	list, err := dbmap.Select(i, query, args...)
	if err != nil {
		panic(err)
	}
	return list
}
