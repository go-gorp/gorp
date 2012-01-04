package gorp_test

import (
    "testing"
	"log"
	"os"
	"reflect"
    "exp/sql"
	. "github.com/coopernurse/gorp"
    _ "github.com/ziutek/mymysql/godrv"
)

type Invoice struct {
    Id          int64
    Created     int64
    Updated     int64
	Memo        string
}

type LineItem struct {
    Id          int64
    Created     int64
    Updated     int64
    InvoiceId   int64
    ProductId   int64
    Quantity    int
    UnitPrice   int
}

func TestMultiple(t *testing.T) {
    dbmap := initDbMap()
	defer dbmap.DropTables()

	inv1 := &Invoice{0, 100, 200, "a"}
	inv2 := &Invoice{0, 100, 200, "b"}
	err := dbmap.Insert(inv1, inv2); if err != nil {
		panic(err)
	}

	inv1.Memo = "c"
	inv2.Memo = "d"
	err = dbmap.Update(inv1, inv2); if err != nil {
		panic(err)
	}

	count, err := dbmap.Delete(inv1, inv2); if err != nil {
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
    err := dbmap.Insert(inv); if err != nil {
		panic(err)
	}
	if inv.Id == 0 {
		t.Errorf("inv.Id was not set on INSERT")
		return
	}

	// SELECT row
    obj, err := dbmap.Get(inv.Id, Invoice{}); if err != nil {
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
	err = dbmap.Update(inv); if err != nil {
		panic(err)
	}
    obj, err = dbmap.Get(inv.Id, Invoice{}); if err != nil {
		panic(err)
	}
    inv2 = obj.(*Invoice)
    if !reflect.DeepEqual(inv, inv2) {
        t.Errorf("%v != %v", inv, inv2)
    }

	// DELETE row
	deleted, err := dbmap.Delete(inv); if err != nil {
		panic(err)
	}
	if deleted != 1 {
		t.Errorf("Did not delete row with Id: %d", inv.Id)
		return
	}

	// VERIFY deleted
	obj, err = dbmap.Get(inv.Id, Invoice{}); if err != nil {
		panic(err)
	}
	if obj != nil {
		t.Errorf("Found invoice with id: %d after Delete()", inv.Id)
	}
}

func initDbMap() *DbMap {
	dbmap := &DbMap{Db: connect(), Dialect: MySQLDialect{}}   
	dbmap.TraceOn("", log.New(os.Stdout, "gorptest: ", log.Lmicroseconds)) 
    dbmap.AddTableWithName(Invoice{}, "invoice_test").SetKeys(true, "Id")
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