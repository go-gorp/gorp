package gorp

import (
	"reflect"
	"sync"
	"testing"
	"time"
)

type testUser struct {
	ID             uint64    `db:"id"`
	Username       string    `db:"user_name"`
	HashedPassword []byte    `db:"hashed_password"`
	EMail          string    `db:"email"`
	CreatedAt      time.Time `db:"created_at"`
	UpdatedAt      time.Time `db:"updated_at"`
}

type testCoolUser struct {
	testUser
	IsCool      bool     `db:"is_cool"`
	BestFriends []string `db:"best_friends"`
}

func BenchmarkCcolumnToFieldIndex(b *testing.B) {
	structType := reflect.TypeOf(testUser{})
	dbmap := &DbMap{Cache: &sync.Map{}}
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

func TestColumnToFieldIndexEmbedded(t *testing.T) {
	structType := reflect.TypeOf(testCoolUser{})
	dbmap := &DbMap{}
	cols, err := columnToFieldIndex(dbmap,
		structType,
		"some_table",
		[]string{
			"id",
			"email",
			"is_cool",
		})
	if err != nil {
		t.Fatal(err)
	}
	if len(cols) != 3 {
		t.Fatal("cols should have 3 results", cols)
	}
	if cols[0][0] != 0 && cols[0][1] != 0 {
		t.Fatal("cols[0][0] should map to id field in testCoolUser", cols)
	}
	if cols[1][0] != 0 && cols[1][1] != 3 {
		t.Fatal("cols[1][0] should map to email field in testCoolUser", cols)
	}
	if cols[2][0] != 1 {
		t.Fatal("cols[2][0] should map to is_cool field in testCoolUser", cols)
	}
}

func TestColumnToFieldIndexEmbeddedFriends(t *testing.T) {
	structType := reflect.TypeOf(testCoolUser{})
	dbmap := &DbMap{}
	cols, err := columnToFieldIndex(dbmap,
		structType,
		"some_table",
		[]string{
			"best_friends",
		})
	if err != nil {
		t.Fatal(err)
	}
	if len(cols) != 1 {
		t.Fatal("cols should have 1 results", cols)
	}
	if cols[0][0] != 2 {
		t.Fatal("cols[0][0] should map to BestFriends field in testCoolUser", cols)
	}
}
