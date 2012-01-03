package gorp

import (
    "testing"
	"reflect"
    "exp/sql"
    _ "github.com/ziutek/mymysql/godrv"
)

type Invoice struct {
    Id          int64
    Created     int64
    Updated     int64
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
    inv := Invoice{Id: 99, Created: 100, Updated: 200}
    //line1 := &LineItem{ProductId: 10, Quantity: 1, UnitPrice: 30}
    //line2 := &LineItem{ProductId: 20, Quantity: 2, UnitPrice: 50}

    dbmap := &DbMap{}
    
    t1 := dbmap.AddTable(Invoice{})
    t1.Name = "invoice_test"

    dbmap.Db = connect()
    dbmap.CreateTables()

    err := dbmap.Put(inv); if err != nil {
		panic(err)
	}
    obj, err := dbmap.Get(inv.Id, Invoice{}); if err != nil {
		panic(err)
	}
    inv2 := obj.(Invoice)
    if !reflect.DeepEqual(inv, inv2) {
        t.Errorf("%v != %v", inv, inv2)
    }
    dbmap.DropTables()
}

func connect() *sql.DB {
    db, err := sql.Open("mymysql", "gomysql_test/gomysql_test/abc123")
    if err != nil {
        panic("Error connecting to db: " + err.Error())
    }
    return db
}