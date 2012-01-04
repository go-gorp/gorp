package gorp_test

import (
    "testing"
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

func TestCrud(t *testing.T) {
    dbmap := &DbMap{}    
    dbmap.AddTableWithName(Invoice{}, "invoice_test").SetKeys(true, "Id")

    dbmap.Db = connect()
    dbmap.CreateTables()
	defer dbmap.DropTables()

    inv := Invoice{99, 100, 200, "first order"}

	// INSERT row
    err := dbmap.Insert(inv); if err != nil {
		panic(err)
	}

	// SELECT row
    obj, err := dbmap.Get(inv.Id, Invoice{}); if err != nil {
		panic(err)
	}
    inv2 := obj.(Invoice)
    if !reflect.DeepEqual(inv, inv2) {
        t.Errorf("%v != %v", inv, inv2)
    }

	// DELETE row
	deleted, err := dbmap.Delete(inv); if err != nil {
		panic(err)
	}
	if !deleted {
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

func connect() *sql.DB {
    db, err := sql.Open("mymysql", "gomysql_test/gomysql_test/abc123")
    if err != nil {
        panic("Error connecting to db: " + err.Error())
    }
    return db
}