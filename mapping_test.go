package gorp

import (
	"reflect"
	"testing"
	"time"
)

type testUser struct {
	Id             uint64    `db:"id"`
	Username       string    `db:"user_name"`
	HashedPassword []byte    `db:"hashed_password"`
	EMail          string    `db:"email"`
	CreatedAt      time.Time `db:"created_at"`
	UpdatedAt      time.Time `db:"updated_at"`
}

func BenchmarkCcolumnToFieldIndex(b *testing.B) {
	structType := reflect.TypeOf(testUser{})
	dbmap := &DbMap{}
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_, err := columnToFieldIndex(dbmap,
			structType,
			"some_table",
			[]string{
				"user_name",
				"email",
				"created_at",
				"updated_at",
				"id",
			})
		if err != nil {
			panic(err)
		}
	}
}

func TestColumnToFieldIndexBasic(t *testing.T) {
	structType := reflect.TypeOf(testUser{})
	dbmap := &DbMap{}
	cols, err := columnToFieldIndex(dbmap,
		structType,
		"some_table",
		[]string{
			"email",
		})
	if err != nil {
		t.Fatal(err)
	}
	if len(cols) != 1 {
		t.Fatal("cols should have 1 result", cols)
	}
	if cols[0][0] != 3 {
		t.Fatal("cols[0][0] should map to email field in testUser", cols)
	}
}

func TestColumnToFieldIndexSome(t *testing.T) {
	structType := reflect.TypeOf(testUser{})
	dbmap := &DbMap{}
	cols, err := columnToFieldIndex(dbmap,
		structType,
		"some_table",
		[]string{
			"id",
			"email",
			"created_at",
		})
	if err != nil {
		t.Fatal(err)
	}
	if len(cols) != 3 {
		t.Fatal("cols should have 3 results", cols)
	}
	if cols[0][0] != 0 {
		t.Fatal("cols[0][0] should map to id field in testUser", cols)
	}
	if cols[1][0] != 3 {
		t.Fatal("cols[1][0] should map to email field in testUser", cols)
	}
	if cols[2][0] != 4 {
		t.Fatal("cols[2][0] should map to created_at field in testUser", cols)
	}
}
