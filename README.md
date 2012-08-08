# Go Relational Persistence #

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

## Note on Building ##

The database/sql package that gorp relies on is under active development.
Please use the latest Go weekly when building gorp.  Specifically, the
2012-01-20 weekly contains the change to rename exp/sql to database/sql, 
and the 2012-01-27 weekly contains the additional Null* types that the
gorp tests rely on.

Once Go 1 lands (real soon now), building gorp should become a reliable, 
boring afair, as it should be.

## Features ##

* Support for transactions
* Forward engineer db schema from structs (great for unit tests)
* Pre/post insert/update/delete hooks
* Automatically generate insert/update/delete statements for a struct
* Automatic binding of auto increment PKs back to struct after insert
* Delete by primary key(s)
* Select by primary key(s)
* Optional trace sql logging
* Bind arbitrary SQL queries to a struct
* Optional optimistic locking using a version column (for update/deletes)

## TODO ##

* Test with more database/sql drivers, such as Postgresql

## Installation ##

    # install the library:
    go get github.com/coopernurse/gorp
    
    // use in your .go code:
    import (
        "github.com/coopernurse/gorp"
    )

## Running the tests ##

The included tests are written against MySQL at the moment. I need to look into
adding additional drivers to the test suite, but for now you can clone the repo
and setup an environment variable before running "go test"

    # Set env variable with dsn using mymysql format.  From the mymysql docs, 
    # the format can be of 3 types:
    #
    #    DBNAME/USER/PASSWD
    #    unix:SOCKPATH*DBNAME/USER/PASSWD
    #    tcp:ADDR*DBNAME/USER/PASSWD
    #
    # for example, on my box I use:
    export GORP_TEST_DSN=gomysql_test/gomysql_test/abc123
    
    # run the tests
    go test
    
    # run the tests and benchmarks
    go test -bench="Bench" -benchtime 10

## Performance ##

gorp uses reflection to construct SQL queries and bind parameters.  See the BenchmarkNativeCrud vs BenchmarkGorpCrud in gorp_test.go for a simple perf test.  On my MacBook Pro gorp is about 2-3% slower than hand written SQL. 

## Examples ##

First define some types:

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

Then create a mapper, typically you'd do this one time at app startup:

    // connect to db using standard Go database/sql API
    // use whatever database/sql driver you wish
    db, err := sql.Open("mysql", "myuser:mypassword@localhost:3306/dbname")
    
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

Automatically create / drop registered tables.  Great for unit tests:

    // create all registered tables
    dbmap.CreateTables()
    
    // drop
    dbmap.DropTables()

Optionally you can pass in a log.Logger to trace all SQL statements:

    // Will log all SQL statements + args as they are run
    // The first arg is a string prefix to prepend to all log messages
    dbmap.TraceOn("[gorp]", log.New(os.Stdout, "myapp:", log.Lmicroseconds)) 
    
    // Turn off tracing
    dbmap.TraceOff()

Then save some data:

    // Must declare as pointers so optional callback hooks
    // can operate on your data, not copies
    inv1 := &Invoice{0, 100, 200, "first order", 0}
    inv2 := &Invoice{0, 100, 200, "second order", 0}
    
    // Insert your rows
    err := dbmap.Insert(inv1, inv2)
    
    // Because we called SetKeys(true) on Invoice, the Id field
    // will be populated after the Insert() automatically
    fmt.Printf("inv1.Id=%d  inv2.Id=%d\n", inv1.Id, inv2.Id)

You can execute raw SQL if you wish.  Particularly good for batch operations.

    res, err := dbmap.Exec("delete from invoice_test where PersonId=?", 10)

Want to do joins?  Just write the SQL and the struct. gorp will bind them:

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
	list, err := dbmap.Select(InvoicePersonView{}, query)
    
    // this should test true
    expected := &InvoicePersonView{inv1.Id, p1.Id, inv1.Memo, p1.FName}
    if reflect.DeepEqual(list[0], expected) {
        fmt.Println("Woot! My join worked!")
    }

You can also batch operations into a transaction:

    func InsertInv(dbmap *DbMap, inv *Invoice, per *Person) error {
        // Start a new transaction
        trans := dbmap.Begin()

        trans.Insert(per)
        inv.PersonId = per.Id
        trans.Insert(inv)

        // if the commit is successful, a nil error is returned
        return trans.Commit()
    }
    
Use hooks to update data before/after saving to the db. Good for timestamps:

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
    
Optimistic locking (similar to JPA)

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
        // use dbmap.SetVersionCol() to map a different
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
    if _, ok := err.(gorp.OptimisticLockError); !ok {
        // should reach this statement
        
        // in a real app you might reload the row and retry, or
        // you might propegate this to the user, depending on the desired
        // semantics
        fmt.Printf("Tried to update row with stale data: %v\n", err)
    }
