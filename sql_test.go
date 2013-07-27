package gorp_test

import (
	"database/sql"
	_ "github.com/ziutek/mymysql/godrv"
	"testing"
)

func connectDb() *sql.DB {
	db, err := sql.Open("mymysql", "gomysql_test/gomysql_test/abc123")
	if err != nil {
		panic("Error connecting to db: " + err.Error())
	}
	return db
}

// This fails on my machine with:
//
// panic: Received #1461 error from MySQL server:
// "Can't create more than max_prepared_stmt_count statements
// (current value: 16382)"
//
// Cause: stmt.Exec() is opening a new db connection for each call
// because each connection is still considered in use
//
func _TestPrepareExec(t *testing.T) {
	db := connectDb()
	defer db.Close()

	db.Exec("drop table if exists test")
	db.Exec("create table test (id int primary key, str varchar(20))")

	query := "insert into test values (?, ?)"
	stmt, err := db.Prepare(query)
	if err != nil {
		panic(err)
	}
	defer stmt.Close()

	for i := 0; i < 20000; i++ {
		_, err := stmt.Exec(i, "some str")
		if err != nil {
			panic(err)
		}
	}

}

// This works
func _TestQuery(t *testing.T) {
	db := connectDb()
	defer db.Close()

	db.Exec("drop table if exists test")
	db.Exec("create table test (id int primary key, str varchar(20))")

	query := "insert into test values (?, ?)"

	for i := 0; i < 20000; i++ {
		_, err := db.Exec(query, i, "some str")
		if err != nil {
			panic(err)
		}
	}

}
