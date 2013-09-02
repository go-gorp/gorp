# Go Relational Persistence #

[![build status](https://secure.travis-ci.org/coopernurse/gorp.png)](http://travis-ci.org/coopernurse/gorp)

I hesitate to call gorp an ORM.  Go doesn't really have objects, at least 
not in the classic Smalltalk/Java sense.  There goes the "O".  gorp doesn't 
know anything about the relationships between your structs (at least not 
yet).  So the "R" is questionable too (but I use it in the name because, 
well, it seemed more clever).

The "M" is alive and well.  Given some Go structs and a database, gorp
should remove a fair amount of boilerplate busy-work from your code.

I hope that gorp saves you time, minimizes the drudgery of getting data 
in and out of your database, and helps your code focus on algorithms, 
not infrastructure.

## Database Drivers ##

gorp uses the Go 1 `database/sql` package.  A full list of compliant drivers is available here:

http://code.google.com/p/go-wiki/wiki/SQLDrivers

Sadly, SQL databases differ on various issues. gorp provides a Dialect interface that should be
implemented per database vendor.  Dialects are provided for:

* MySQL
* PostgreSQL
* sqlite3

Each of these three databases pass the test suite.  See `gorp_test.go` for example 
DSNs for these three databases.

## Features ##

* Bind struct fields to table columns via API or tag
* Support for embedded structs
* Support for transactions
* Forward engineer db schema from structs (great for unit tests)
* Pre/post insert/update/delete hooks
* Automatically generate insert/update/delete statements for a struct
* Automatic binding of auto increment PKs back to struct after insert
* Delete by primary key(s)
* Select by primary key(s)
* Optional trace sql logging
* Bind arbitrary SQL queries to a struct
* Bind slice pointers to SELECT query results without type assertions
* Optional optimistic locking using a version column (for update/deletes)

## Installation ##

    # install the library:
    go get github.com/coopernurse/gorp
    
    // use in your .go code:
    import (
        "github.com/coopernurse/gorp"
    )

## API Documentation ##

Full godoc output from the latest code in master is available here:

http://godoc.org/github.com/coopernurse/gorp

## Examples ##

### Mapping structs to tables ###

First define some types:

```go
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

// Example of using tags to alias fields to column names
// The 'db' value is the column name
//
// A hyphen will cause gorp to skip this field, similar to the
// Go json package.
//
// This is equivalent to using the ColMap methods:
//
//   table := dbmap.AddTableWithName(Product{}, "product")
//   table.ColMap("Id").Rename("product_id")
//   table.ColMap("Price").Rename("unit_price")
//   table.ColMap("IgnoreMe").SetTransient(true)
//
type Product struct {
    Id         int64     `db:"product_id"`
    Price      int64     `db:"unit_price"`
    IgnoreMe   string    `db:"-"`
}
```

Then create a mapper, typically you'd do this one time at app startup:

```go
// connect to db using standard Go database/sql API
// use whatever database/sql driver you wish
db, err := sql.Open("mymysql", "tcp:localhost:3306*mydb/myuser/mypassword")

// construct a gorp DbMap
dbmap := &gorp.DbMap{Db: db, Dialect: gorp.MySQLDialect{"InnoDB", "UTF8"}}

// register the structs you wish to use with gorp
// you can also use the shorter dbmap.AddTable() if you 
// don't want to override the table name
//
// SetKeys(true) means we have a auto increment primary key, which
// will get automatically bound to your struct post-insert
//
t1 := dbmap.AddTableWithName(Invoice{}, "invoice_test").SetKeys(true, "Id")
t2 := dbmap.AddTableWithName(Person{}, "person_test").SetKeys(true, "Id")
t3 := dbmap.AddTableWithName(Product{}, "product_test").SetKeys(true, "Id")
```

### Struct Embedding ###

gorp supports embedding structs.  For example:

```go
type Names struct {
    FirstName string
    LastName  string
}

type WithEmbeddedStruct struct {
    Id int64
    Names
}

es := &WithEmbeddedStruct{-1, Names{FirstName: "Alice", LastName: "Smith"}}
err := dbmap.Insert(es)
```

See the `TestWithEmbeddedStruct` function in `gorp_test.go` for a full example.

### Create/Drop Tables ###

Automatically create / drop registered tables.  This is useful for unit tests
but is entirely optional.  You can of course use gorp with tables created manually,
or with a separate migration tool (like goose: https://bitbucket.org/liamstask/goose).

```go
// create all registered tables
dbmap.CreateTables()

// same as above, but uses "if not exists" clause to skip tables that are
// already defined
dbmap.CreateTablesIfNotExists()

// drop
dbmap.DropTables()
```

### SQL Logging ###

Optionally you can pass in a log.Logger to trace all SQL statements.
I recommend enabling this initially while you're getting the feel for what
gorp is doing on your behalf.

```go
// Will log all SQL statements + args as they are run
// The first arg is a string prefix to prepend to all log messages
dbmap.TraceOn("[gorp]", log.New(os.Stdout, "myapp:", log.Lmicroseconds)) 

// Turn off tracing
dbmap.TraceOff()
```

### Insert ###

```go
// Must declare as pointers so optional callback hooks
// can operate on your data, not copies
inv1 := &Invoice{0, 100, 200, "first order", 0}
inv2 := &Invoice{0, 100, 200, "second order", 0}

// Insert your rows
err := dbmap.Insert(inv1, inv2)

// Because we called SetKeys(true) on Invoice, the Id field
// will be populated after the Insert() automatically
fmt.Printf("inv1.Id=%d  inv2.Id=%d\n", inv1.Id, inv2.Id)
```

### Update ###

Continuing the above example, use the `Update` method to modify an Invoice:

```go
// count is the # of rows updated, which should be 1 in this example
count, err := dbmap.Update(inv1)
```

### Delete ###

If you have primary key(s) defined for a struct, you can use the `Delete`
method to remove rows:

```go
count, err := dbmap.Delete(inv1)
```

### Select by Key ###

Use the `Get` method to fetch a single row by primary key.  It returns
nil if now row is found.

```go
// fetch Invoice with Id=99
obj, err := dbmap.Get(Invoice{}, 99)
inv := obj.(*Invoice)
```

### Ad Hoc SQL ###

#### SELECT ####

Want to do joins?  Just write the SQL and the struct. gorp will bind them:

```go
// Define a type for your join
// It *must* contain all the columns in your SELECT statement
//
// The names here should match the aliased column names you specify
// in your SQL - no additional binding work required.  simple.
//
type InvoicePersonView struct {
    InvoiceId   int64
    PersonId    int64
    Memo        string
    FName       string
}

// Create some rows
p1 := &Person{0, 0, 0, "bob", "smith"}
dbmap.Insert(p1)

// notice how we can wire up p1.Id to the invoice easily
inv1 := &Invoice{0, 0, 0, "xmas order", p1.Id}
dbmap.Insert(inv1)

// Run your query
query := "select i.Id InvoiceId, p.Id PersonId, i.Memo, p.FName " +
	"from invoice_test i, person_test p " +
	"where i.PersonId = p.Id"

// pass a slice of pointers to Select()
// this avoids the need to type assert after the query is run
var list []*InvoicePersonView
_, err := dbmap.Select(&list, query)

// this should test true
expected := &InvoicePersonView{inv1.Id, p1.Id, inv1.Memo, p1.FName}
if reflect.DeepEqual(list[0], expected) {
    fmt.Println("Woot! My join worked!")
}
```

#### SELECT string or int64 ####

gorp provides a few convenience methods for selecting a single string or int64.

```go
// select single int64 from db:
i64, err := dbmap.SelectInt("select count(*) from foo where blah=?", blahVal)

// select single string from db:
s, err := dbmap.SelectStr("select name from foo where blah=?", blahVal)

```

#### UPDATE / DELETE ####

You can execute raw SQL if you wish.  Particularly good for batch operations.

```go
res, err := dbmap.Exec("delete from invoice_test where PersonId=?", 10)
```

### Transactions ###

You can batch operations into a transaction:

```go
func InsertInv(dbmap *DbMap, inv *Invoice, per *Person) error {
    // Start a new transaction
    trans, err := dbmap.Begin()
    if err != nil {
        return err
    }

    trans.Insert(per)
    inv.PersonId = per.Id
    trans.Insert(inv)

    // if the commit is successful, a nil error is returned
    return trans.Commit()
}
```

### Hooks ###

Use hooks to update data before/after saving to the db. Good for timestamps:

```go
// implement the PreInsert and PreUpdate hooks
func (i *Invoice) PreInsert(s gorp.SqlExecutor) error {
    i.Created = time.Now().UnixNano()
    i.Updated = i.Created
    return nil
}

func (i *Invoice) PreUpdate(s gorp.SqlExecutor) error {
    i.Updated = time.Now().UnixNano()
    return nil
}

// You can use the SqlExecutor to cascade additional SQL
// Take care to avoid cycles. gorp won't prevent them.
//
// Here's an example of a cascading delete
//
func (p *Person) PreDelete(s gorp.SqlExecutor) error {
    query := "delete from invoice_test where PersonId=?"
    err := s.Exec(query, p.Id); if err != nil {
        return err
    }
    return nil
}
```

Full list of hooks that you can implement:

    PostGet
    PreInsert
    PostInsert
    PreUpdate
    PostUpdate
    PreDelete
    PostDelete
    
    All have the same signature.  for example:
    
    func (p *MyStruct) PostUpdate(s gorp.SqlExecutor) error
    
### Optimistic Locking ###

gorp provides a simple optimistic locking feature, similar to Java's JPA, that
will raise an error if you try to update/delete a row whose `version` column
has a value different than the one in memory.  This provides a safe way to do
"select then update" style operations without explicit read and write locks.

```go
// Version is an auto-incremented number, managed by gorp
// If this property is present on your struct, update
// operations will be constrained
//
// For example, say we defined Person as:

type Person struct {
    Id       int64
    Created  int64
    Updated  int64
    FName    string
    LName    string
    
    // automatically used as the Version col
    // use table.SetVersionCol("columnName") to map a different
    // struct field as the version field
    Version  int64
}

p1 := &Person{0, 0, 0, "Bob", "Smith", 0}
dbmap.Insert(p1)  // Version is now 1

obj, err := dbmap.Get(Person{}, p1.Id)
p2 := obj.(*Person)
p2.LName = "Edwards"
dbmap.Update(p2)  // Version is now 2

p1.LName = "Howard"

// Raises error because p1.Version == 1, which is out of date
count, err := dbmap.Update(p1)
_, ok := err.(gorp.OptimisticLockError)
if ok {
    // should reach this statement
    
    // in a real app you might reload the row and retry, or
    // you might propegate this to the user, depending on the desired
    // semantics
    fmt.Printf("Tried to update row with stale data: %v\n", err)
} else {
    // some other db error occurred - log or return up the stack
    fmt.Printf("Unknown db err: %v\n", err)
}
```

## Running the tests ##

The included tests may be run against MySQL, Postgresql, or sqlite3.
You must set two environment variables so the test code knows which driver to
use, and how to connect to your database.

```sh
# MySQL example:
export GORP_TEST_DSN=gomysql_test/gomysql_test/abc123
export GORP_TEST_DIALECT=mysql

# run the tests
go test

# run the tests and benchmarks
go test -bench="Bench" -benchtime 10
```

Valid `GORP_TEST_DIALECT` values are: "mysql", "postgres", "sqlite3"
See the `test_all.sh` script for examples of all 3 databases.  This is the script I run
locally to test the library.

## Performance ##

gorp uses reflection to construct SQL queries and bind parameters.  See the BenchmarkNativeCrud vs BenchmarkGorpCrud in gorp_test.go for a simple perf test.  On my MacBook Pro gorp is about 2-3% slower than hand written SQL. 

## Contributors

* matthias-margush - column aliasing via tags
* Rob Figueiredo - @robfig
* Quinn Slack - @sqs
