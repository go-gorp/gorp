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

type Invoice struct {
	Id      int64
	Created int64
	Updated int64
	Memo    string
}

type Person struct {
	Id      int64
	Created int64
	Updated int64
	FName   string
	LName   string
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

func TestHooks(t *testing.T) {
	dbmap := initDbMap()
	defer dbmap.DropTables()

	p1 := &Person{0, 0, 0, "bob", "smith"}
	err := dbmap.Insert(p1); if err != nil {
		panic(err)
	}
	if p1.Created == 0 || p1.Updated == 0 {
		t.Errorf("p1.PreInsert() didn't run: %v", p1)
	} else if p1.LName != "postinsert" {
		t.Errorf("p1.PostInsert() didn't run: %v", p1)
	}

	obj, err := dbmap.Get(p1.Id, Person{}); if err != nil {
		panic(err)
	}
	p1 = obj.(*Person)
	if p1.LName != "postget" {
		t.Errorf("p1.PostGet() didn't run: %v", p1)
	}

	err = dbmap.Update(p1); if err != nil {
		panic(err)
	}
	if p1.FName != "preupdate" {
		t.Errorf("p1.PreUpdate() didn't run: %v", p1)
	} else if p1.LName != "postupdate" {
		t.Errorf("p1.PostUpdate() didn't run: %v", p1)
	}

	_, err = dbmap.Delete(p1); if err != nil {
		panic(err)
	}
	if p1.FName != "predelete" {
		t.Errorf("p1.PreDelete() didn't run: %v", p1)
	} else if p1.LName != "postdelete" {
		t.Errorf("p1.PostDelete() didn't run: %v", p1)
	}

	// Test error case
	p2 := &Person{0, 0, 0, "badname", ""}
	err = dbmap.Insert(p2); if err == nil {
		t.Errorf("p2.PreInsert() didn't return an error")
	}
}

func TestTransaction(t *testing.T) {
	dbmap := initDbMap()
	defer dbmap.DropTables()

	inv1 := &Invoice{0, 100, 200, "t1"}
	inv2 := &Invoice{0, 100, 200, "t2"}

	trans, err := dbmap.Begin()
	if err != nil {
		panic(err)
	}
	trans.Insert(inv1, inv2)
	err = trans.Commit()
	if err != nil {
		panic(err)
	}

	obj, err := dbmap.Get(inv1.Id, Invoice{})
	if err != nil {
		panic(err)
	}
	if !reflect.DeepEqual(inv1, obj) {
		t.Errorf("%v != %v", inv1, obj)
	}
	obj, err = dbmap.Get(inv2.Id, Invoice{})
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

	inv1 := &Invoice{0, 100, 200, "a"}
	inv2 := &Invoice{0, 100, 200, "b"}
	err := dbmap.Insert(inv1, inv2)
	if err != nil {
		panic(err)
	}

	inv1.Memo = "c"
	inv2.Memo = "d"
	err = dbmap.Update(inv1, inv2)
	if err != nil {
		panic(err)
	}

	count, err := dbmap.Delete(inv1, inv2)
	if err != nil {
		panic(err)
	}
	if count != 2 {
		t.Errorf("%d != 2", count)
	}
}

func TestCrud(t *testing.T) {
	dbmap := initDbMap()
	defer dbmap.DropTables()

	inv := &Invoice{0, 100, 200, "first order"}

	// INSERT row
	err := dbmap.Insert(inv)
	if err != nil {
		panic(err)
	}
	if inv.Id == 0 {
		t.Errorf("inv.Id was not set on INSERT")
		return
	}

	// SELECT row
	obj, err := dbmap.Get(inv.Id, Invoice{})
	if err != nil {
		panic(err)
	}
	inv2 := obj.(*Invoice)
	if !reflect.DeepEqual(inv, inv2) {
		t.Errorf("%v != %v", inv, inv2)
	}

	// UPDATE row and SELECT
	inv.Memo = "second order"
	inv.Created = 999
	inv.Updated = 11111
	err = dbmap.Update(inv)
	if err != nil {
		panic(err)
	}
	obj, err = dbmap.Get(inv.Id, Invoice{})
	if err != nil {
		panic(err)
	}
	inv2 = obj.(*Invoice)
	if !reflect.DeepEqual(inv, inv2) {
		t.Errorf("%v != %v", inv, inv2)
	}

	// DELETE row
	deleted, err := dbmap.Delete(inv)
	if err != nil {
		panic(err)
	}
	if deleted != 1 {
		t.Errorf("Did not delete row with Id: %d", inv.Id)
		return
	}

	// VERIFY deleted
	obj, err = dbmap.Get(inv.Id, Invoice{})
	if err != nil {
		panic(err)
	}
	if obj != nil {
		t.Errorf("Found invoice with id: %d after Delete()", inv.Id)
	}
}

func initDbMap() *DbMap {
	dialect := MySQLDialect{"InnoDB", "UTF8"}
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
