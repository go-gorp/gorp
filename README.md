# Go Relational Persistence #

## Goals ##

* Support transactions
* Forward engineer db schema from structs (especially good for unit tests)
* Optional optimistic locking using a version column
* Pre/post insert/update/delete hooks
* Automatically generate insert/update statements for a struct
* Delete by primary key
* Select by primary key
* Optional trace sql logging

## Non-Goals ##

* Eliminate need to know SQL
* Graph loads or saves

## Examples ##

First define some types:

    type DbStruct {
        Id        int64,
        Created   int64,
        Updated   int64,
        Version   int32,
    }
    
    type Product struct {
        DbStruct
        Description  string,
        UnitPrice    int32,   // in pennies
        IsTaxable    bool
    }
    
    type Order struct {
        DbStruct
        PaymentType  string,
        IsPaid       bool,
        SalesTax     int32
    }
    
    type LineItem struct {
        DbStruct
        OrderId      int64,
        ProductId    int64,
        Quantity     int32,
        UnitPrice    int32
    }

Then create a mapper, typically you'd do this one time at app startup:

    // connect to db using standard Go exp/sql API
    // use whatever exp/sql driver you wish
    db, err := sql.Open("mysql", "myuser:mypassword@localhost:3306/dbname")
    
    // construct a gorp DbMap
    dbmap := &gorp.DbMap{Db: db, Dialect: gorp.MySQLDialect{}}
    
    // register the structs you wish to use with gorp
    t1 : = dbmap.AddTable(Product{})
    t1.SetKeys(true, "Id")
    
    // or use the builder syntax
    dbmap.AddTable(Order{}).SetKeys("Id")
    
    // optionally override the table name
    dbmap.AddTableWithName(LineItem{}, "line_item").SetKeys("Id")

Optionally you can pass in a log.Logger to trace all SQL statements:

    // Will log all SQL statements + args as they are run
    // The first arg is a string prefix to prepend to all log messages
    dbmap.TraceOn("[gorp]", log.New(os.Stdout, "myapp:", log.Lmicroseconds)) 
    
    // Turn off tracing
    dbmap.TraceOff()

Then save some data:

    p1 := &Product{Description: "Wool socks", UnitPrice: 499, IsTaxable: true}
    p2 := &Product{Description: "Tofu", UnitPrice: 249, IsTaxable: false}
    
    // Pass as a pointer so that optional callback hooks
    // can operate on your data, not copies
    dbmap.Insert(&p1, &p2)
    
    // Because we called SetAutoIncrPK() on Product, the Id field
    // will be populated after the Insert() automatically
    fmt.Printf("p1.Id=%d\n", p1.Id)

You can also batch operations into a transaction:

    func InsertOrder(dbmap *DbMap, order *Order, items []LineItem) error {
        // Start a new transaction
        trans := dbmap.Begin()

        trans.Insert(order)
        for _, v := range(items) {
            trans.Insert(v)
        }

        // if the commit is successful, a nil error is returned
        return trans.Commit()
    }
    
How would I set the date updated/created?

    // implement the PreInsert and PreUpdate hooks
    func (d *DbStruct) PreInsert(mapper *gorp.Mapper) error {
        d.Created = time.Nanoseconds()
        d.Updated = d.Created
        return nil
    }
    
    func (d *DbStruct) PreUpdate(mapper *gorp.Mapper) error {
        d.Updated = time.Nanoseconds()
        return nil
    }
    
What is Version used for?

    // Version is an auto-incremented number, managed by gorp
    // If this property is present on your struct, update
    // operations will be constrained
    //
    // For example:
    
    p1 := &Product{Description: "Wool socks", UnitPrice: 499, IsTaxable: true}
    dbmap.Save(p1)  // Version is now 1
    
    p2 := dbmap.Get(Product, p1.Id)
    p2.UnitPrice = 599
    dbmap.Save(p2)  // Version is now 2
    
    p1.UnitPrice = 399
    err := dbmap.Save(p1)  // Raises error - p1.Version == 1, which is stale
    if err != nil {
        // should reach this statement
        fmt.Printf("Got err: %v\n", err)
    }
    
