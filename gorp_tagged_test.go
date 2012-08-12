package gorp_test

import (
	"database/sql"
	"errors"
	"fmt"
	. "github.com/coopernurse/gorp"
	_ "github.com/ziutek/mymysql/godrv"
	"log"
	"os"
	"reflect"
	"testing"
	"time"
)

var dialect_tagged = MySQLDialect{"InnoDB", "UTF8"}

type InvoiceTagged struct {
	Id             int64  `db:"Id_tagged"`
	Created        int64  `db:"Created_tagged"`
	Updated        int64  `db:"Updated_tagged"`
	Memo           string `db:"Memo_tagged"`
	PersonTaggedId int64  `db:"PersonTaggedId_tagged"`
}

type PersonTagged struct {
	Id      int64  `db:"Id_tagged"`
	Created int64  `db:"Created_tagged"`
	Updated int64  `db:"Updated_tagged"`
	FName   string `db:"FName_tagged"`
	LName   string `db:"LName_tagged"`
	Version int64  `db:"Version_tagged"`
}

type InvoiceTaggedPersonTaggedView struct {
	InvoiceTaggedId int64  `db:"InvoiceTaggedId_tagged"`
	PersonTaggedId  int64  `db:"PersonTaggedId_tagged"`
	Memo            string `db:"Memo_tagged"`
	FName           string `db:"FName_tagged"`
	LegacyVersion   int64  `db:"LegacyVersion_tagged"`
}

type TableWithNullTagged struct {
	Id      int64           `db:"Id_tagged"`
	Str     sql.NullString  `db:"Str_tagged"`
	Int64   sql.NullInt64   `db:"Int64_tagged"`
	Float64 sql.NullFloat64 `db:"Float64_tagged"`
	Bool    sql.NullBool    `db:"Bool_tagged"`
	Bytes   []byte          `db:"Bytes_tagged"`
}

func (p *PersonTagged) PreInsert(s SqlExecutor) error {
	p.Created = time.Now().UnixNano()
	p.Updated = p.Created
	if p.FName == "badname" {
		return errors.New(fmt.Sprintf("Invalid name: %s", p.FName))
	}
	return nil
}

func (p *PersonTagged) PostInsert(s SqlExecutor) error {
	p.LName = "postinsert"
	return nil
}

func (p *PersonTagged) PreUpdate(s SqlExecutor) error {
	p.FName = "preupdate"
	return nil
}

func (p *PersonTagged) PostUpdate(s SqlExecutor) error {
	p.LName = "postupdate"
	return nil
}

func (p *PersonTagged) PreDelete(s SqlExecutor) error {
	p.FName = "predelete"
	return nil
}

func (p *PersonTagged) PostDelete(s SqlExecutor) error {
	p.LName = "postdelete"
	return nil
}

func (p *PersonTagged) PostGet(s SqlExecutor) error {
	p.LName = "postget"
	return nil
}

type PersistentUserTagged struct {
	Key            int32
	Id             string
	PassedTraining bool
}

func TestPersistentUserTagged(t *testing.T) {
	dbmap := &DbMap{Db: connect(), Dialect: dialect}
	dbmap.Exec("drop table if exists PersistentUserTagged")
	dbmap.TraceOn("", log.New(os.Stdout, "gorptest: ", log.Lmicroseconds))
	table := dbmap.AddTable(PersistentUserTagged{}).SetKeys(false, "Key")
	table.ColMap("Key").Rename("mykey")
	err := dbmap.CreateTables()
	if err != nil {
		panic(err)
	}
	defer dbmap.DropTables()
	pu := &PersistentUserTagged{43, "33r", false}
	err = dbmap.Insert(pu)
	if err != nil {
		panic(err)
	}
}

func TestOverrideVersionColTagged(t *testing.T) {
	dbmap := initDbMapTagged()
	dbmap.DropTables()
	t1 := dbmap.AddTable(InvoiceTaggedPersonTaggedView{}).SetKeys(false, "InvoiceTaggedId_tagged", "PersonTaggedId_tagged")
	err := dbmap.CreateTables()
	if err != nil {
		panic(err)
	}
	defer dbmap.DropTables()
	c1 := t1.SetVersionCol("LegacyVersion")
	if c1.ColumnName != "LegacyVersion_tagged" {
		t.Errorf("Wrong col returned: %v", c1)
	}

	ipv := &InvoiceTaggedPersonTaggedView{1, 2, "memo", "fname", 0}
	update(dbmap, ipv)
	if ipv.LegacyVersion != 1 {
		t.Errorf("LegacyVersion not updated: %d", ipv.LegacyVersion)
	}
}

func TestOptimisticLockingTagged(t *testing.T) {
	dbmap := initDbMapTagged()
	defer dbmap.DropTables()

	p1 := &PersonTagged{0, 0, 0, "Bob", "Smith", 0}
	dbmap.Insert(p1) // Version is now 1
	if p1.Version != 1 {
		t.Errorf("Insert didn't incr Version: %d != %d", 1, p1.Version)
		return
	}

	obj, err := dbmap.Get(PersonTagged{}, p1.Id)
	p2 := obj.(*PersonTagged)
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
func TestDoubleAddTableTagged(t *testing.T) {
	dbmap := &DbMap{Db: connect(), Dialect: dialect}
	t1 := dbmap.AddTable(TableWithNullTagged{}).SetKeys(false, "Id")
	t2 := dbmap.AddTable(TableWithNullTagged{})
	if t1 != t2 {
		t.Errorf("%v != %v", t1, t2)
	}
}

// what happens if a legacy table has a null value?
func TestNullValuesTagged(t *testing.T) {
	dbmap := &DbMap{Db: connect(), Dialect: dialect}
	dbmap.TraceOn("", log.New(os.Stdout, "gorptest: ", log.Lmicroseconds))
	dbmap.AddTable(TableWithNullTagged{}).SetKeys(false, "Id")
	dbmap.CreateTables()
	defer dbmap.DropTables()

	// insert a row directly
	rawexec(dbmap, "insert into TableWithNullTagged values (10, null, "+
		"null, null, null, null)")

	// try to load it
	expected := &TableWithNullTagged{Id: 10}
	obj := get(dbmap, TableWithNullTagged{}, 10)
	t1 := obj.(*TableWithNullTagged)
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
	update(dbmap, t1)

	obj = get(dbmap, TableWithNullTagged{}, 10)
	t1 = obj.(*TableWithNullTagged)
	if t1.Str.String != "hi" {
		t.Errorf("%s != hi", t1.Str.String)
	}
	if !reflect.DeepEqual(expected, t1) {
		t.Errorf("%v != %v", expected, t1)
	}
}

func TestColumnPropsTagged(t *testing.T) {
	dbmap := &DbMap{Db: connect(), Dialect: dialect}
	dbmap.TraceOn("", log.New(os.Stdout, "gorptest: ", log.Lmicroseconds))
	t1 := dbmap.AddTable(InvoiceTagged{}).SetKeys(true, "Id")
	t1.ColMap("Created").Rename("date_created")
	t1.ColMap("Updated").SetTransient(true)
	t1.ColMap("Memo").SetMaxSize(10)
	t1.ColMap("PersonTaggedId").SetUnique(true)

	dbmap.CreateTables()
	defer dbmap.DropTables()

	// test transient
	inv := &InvoiceTagged{0, 0, 1, "my invoice", 0}
	insert(dbmap, inv)
	obj := get(dbmap, InvoiceTagged{}, inv.Id)
	inv = obj.(*InvoiceTagged)
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
	inv = &InvoiceTagged{0, 0, 1, "my invoice2", 0}
	err = dbmap.Insert(inv)
	if err == nil {
		t.Errorf("same PersonTaggedId inserted, but Insert did not fail.")
	}
}

func TestRawSelectTagged(t *testing.T) {
	dbmap := initDbMapTagged()
	defer dbmap.DropTables()

	p1 := &PersonTagged{0, 0, 0, "bob", "smith", 0}
	insert(dbmap, p1)

	inv1 := &InvoiceTagged{0, 0, 0, "xmas order", p1.Id}
	insert(dbmap, inv1)

	expected := &InvoiceTaggedPersonTaggedView{inv1.Id, p1.Id, inv1.Memo, p1.FName, 0}

	query := "select i.Id_tagged InvoiceTaggedId_tagged, p.Id_tagged PersonTaggedId_tagged, i.Memo_tagged, p.FName_tagged " +
		"from invoice_test_tagged i, person_test_tagged p " +
		"where i.PersonTaggedId_tagged = p.Id_tagged"
	list := rawselect(dbmap, InvoiceTaggedPersonTaggedView{}, query)
	if len(list) != 1 {
		t.Errorf("len(list) != 1: %d", len(list))
	} else if !reflect.DeepEqual(expected, list[0]) {
		t.Errorf("%v != %v", expected, list[0])
	}
}

func TestHooksTagged(t *testing.T) {
	dbmap := initDbMapTagged()
	defer dbmap.DropTables()

	p1 := &PersonTagged{0, 0, 0, "bob", "smith", 0}
	insert(dbmap, p1)
	if p1.Created == 0 || p1.Updated == 0 {
		t.Errorf("p1.PreInsert() didn't run: %v", p1)
	} else if p1.LName != "postinsert" {
		t.Errorf("p1.PostInsert() didn't run: %v", p1)
	}

	obj := get(dbmap, PersonTagged{}, p1.Id)
	p1 = obj.(*PersonTagged)
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
	p2 := &PersonTagged{0, 0, 0, "badname", "", 0}
	err := dbmap.Insert(p2)
	if err == nil {
		t.Errorf("p2.PreInsert() didn't return an error")
	}
}

func TestTransactionTagged(t *testing.T) {
	dbmap := initDbMapTagged()
	defer dbmap.DropTables()

	inv1 := &InvoiceTagged{0, 100, 200, "t1", 0}
	inv2 := &InvoiceTagged{0, 100, 200, "t2", 0}

	trans, err := dbmap.Begin()
	if err != nil {
		panic(err)
	}
	trans.Insert(inv1, inv2)
	err = trans.Commit()
	if err != nil {
		panic(err)
	}

	obj, err := dbmap.Get(InvoiceTagged{}, inv1.Id)
	if err != nil {
		panic(err)
	}
	if !reflect.DeepEqual(inv1, obj) {
		t.Errorf("%v != %v", inv1, obj)
	}
	obj, err = dbmap.Get(InvoiceTagged{}, inv2.Id)
	if err != nil {
		panic(err)
	}
	if !reflect.DeepEqual(inv2, obj) {
		t.Errorf("%v != %v", inv2, obj)
	}
}

func TestMultipleTagged(t *testing.T) {
	dbmap := initDbMapTagged()
	defer dbmap.DropTables()

	inv1 := &InvoiceTagged{0, 100, 200, "a", 0}
	inv2 := &InvoiceTagged{0, 100, 200, "b", 0}
	insert(dbmap, inv1, inv2)

	inv1.Memo = "c"
	inv2.Memo = "d"
	update(dbmap, inv1, inv2)

	count := del(dbmap, inv1, inv2)
	if count != 2 {
		t.Errorf("%d != 2", count)
	}
}

func TestCrudTagged(t *testing.T) {
	dbmap := initDbMapTagged()
	defer dbmap.DropTables()

	inv := &InvoiceTagged{0, 100, 200, "first order", 0}

	// INSERT row
	insert(dbmap, inv)
	if inv.Id == 0 {
		t.Errorf("inv.Id was not set on INSERT")
		return
	}

	// SELECT row
	obj := get(dbmap, InvoiceTagged{}, inv.Id)
	inv2 := obj.(*InvoiceTagged)
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
	obj = get(dbmap, InvoiceTagged{}, inv.Id)
	inv2 = obj.(*InvoiceTagged)
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
	obj = get(dbmap, InvoiceTagged{}, inv.Id)
	if obj != nil {
		t.Errorf("Found invoice with id: %d after Delete()", inv.Id)
	}
}

func BenchmarkNativeCrudTagged(b *testing.B) {
	b.StopTimer()
	dbmap := initDbMapTaggedBench()
	defer dbmap.DropTables()
	b.StartTimer()

	insert := "insert into invoice_test (Created, Updated, Memo, PersonTaggedId) values (?, ?, ?, ?)"
	sel := "select Id, Created, Updated, Memo, PersonTaggedId from invoice_test where Id=?"
	update := "update invoice_test set Created=?, Updated=?, Memo=?, PersonTaggedId=? where Id=?"
	delete := "delete from invoice_test where Id=?"

	inv := &InvoiceTagged{0, 100, 200, "my memo", 0}

	for i := 0; i < b.N; i++ {
		res, err := dbmap.Db.Exec(insert, inv.Created, inv.Updated,
			inv.Memo, inv.PersonTaggedId)
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
			&inv.PersonTaggedId)
		if err != nil {
			panic(err)
		}

		inv.Created = 1000
		inv.Updated = 2000
		inv.Memo = "my memo 2"
		inv.PersonTaggedId = 3000

		_, err = dbmap.Db.Exec(update, inv.Created, inv.Updated, inv.Memo,
			inv.PersonTaggedId, inv.Id)
		if err != nil {
			panic(err)
		}

		_, err = dbmap.Db.Exec(delete, inv.Id)
		if err != nil {
			panic(err)
		}
	}

}

func BenchmarkGorpCrudTagged(b *testing.B) {
	b.StopTimer()
	dbmap := initDbMapTaggedBench()
	defer dbmap.DropTables()
	b.StartTimer()

	inv := &InvoiceTagged{0, 100, 200, "my memo", 0}
	for i := 0; i < b.N; i++ {
		err := dbmap.Insert(inv)
		if err != nil {
			panic(err)
		}

		obj, err := dbmap.Get(InvoiceTagged{}, inv.Id)
		if err != nil {
			panic(err)
		}

		inv2, ok := obj.(*InvoiceTagged)
		if !ok {
			panic(fmt.Sprintf("expected *InvoiceTagged, got: %v", obj))
		}

		inv2.Created = 1000
		inv2.Updated = 2000
		inv2.Memo = "my memo 2"
		inv2.PersonTaggedId = 3000
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

func initDbMapTaggedBench() *DbMap {
	dbmap := &DbMap{Db: connect(), Dialect: dialect}
	dbmap.Db.Exec("drop table if exists invoice_test")
	dbmap.AddTableWithName(InvoiceTagged{}, "invoice_test").SetKeys(true, "Id")
	err := dbmap.CreateTables()
	if err != nil {
		panic(err)
	}
	return dbmap
}

func initDbMapTagged() *DbMap {
	dbmap := &DbMap{Db: connect(), Dialect: dialect}
	dbmap.TraceOn("", log.New(os.Stdout, "gorptest: ", log.Lmicroseconds))
	dbmap.AddTableWithName(InvoiceTagged{}, "invoice_test_tagged").SetKeys(true, "Id")
	dbmap.AddTableWithName(PersonTagged{}, "person_test_tagged").SetKeys(true, "Id")
	dbmap.CreateTables()

	return dbmap
}
