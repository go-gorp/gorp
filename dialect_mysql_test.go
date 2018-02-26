// Copyright 2012 James Cooper. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// Package gorp provides a simple way to marshal Go structs to and from
// SQL databases.  It uses the database/sql package, and should work with any
// compliant database/sql driver.
//
// Source code and project home:
// https://github.com/go-gorp/gorp

package gorp_test

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/apoydence/onpar"
	"github.com/apoydence/onpar/expect"
	"github.com/apoydence/onpar/matchers"
	"github.com/go-gorp/gorp"
)

func TestMySQLDialect(t *testing.T) {
	o := onpar.New()
	defer o.Run(t)

	o.BeforeEach(func(t *testing.T) (expect.Expectation, gorp.MySQLDialect) {
		return expect.New(t), gorp.MySQLDialect{
			Engine:   "foo",
			Encoding: "bar",
		}
	})

	o.Group("ToSqlType", func() {
		tests := []struct {
			name     string
			value    interface{}
			maxSize  int
			autoIncr bool
			expected string
		}{
			{"bool", true, 0, false, "boolean"},
			{"int8", int8(1), 0, false, "tinyint"},
			{"uint8", uint8(1), 0, false, "tinyint unsigned"},
			{"int16", int16(1), 0, false, "smallint"},
			{"uint16", uint16(1), 0, false, "smallint unsigned"},
			{"int32", int32(1), 0, false, "int"},
			{"int (treated as int32)", int(1), 0, false, "int"},
			{"uint32", uint32(1), 0, false, "int unsigned"},
			{"uint (treated as uint32)", uint(1), 0, false, "int unsigned"},
			{"int64", int64(1), 0, false, "bigint"},
			{"uint64", uint64(1), 0, false, "bigint unsigned"},
			{"float32", float32(1), 0, false, "double"},
			{"float64", float64(1), 0, false, "double"},
			{"[]uint8", []uint8{1}, 0, false, "mediumblob"},
			{"NullInt64", sql.NullInt64{}, 0, false, "bigint"},
			{"NullFloat64", sql.NullFloat64{}, 0, false, "double"},
			{"NullBool", sql.NullBool{}, 0, false, "tinyint"},
			{"Time", time.Time{}, 0, false, "datetime"},
			{"default-size string", "", 0, false, "varchar(255)"},
			{"sized string", "", 50, false, "varchar(50)"},
			{"large string", "", 1024, false, "text"},
		}
		for _, t := range tests {
			o.Spec(t.name, func(expect expect.Expectation, dialect gorp.MySQLDialect) {
				typ := reflect.TypeOf(t.value)
				sqlType := dialect.ToSqlType(typ, t.maxSize, t.autoIncr)
				expect(sqlType).To(matchers.Equal(t.expected))
			})
		}
	})

	o.Spec("AutoIncrStr", func(expect expect.Expectation, dialect gorp.MySQLDialect) {
		expect(dialect.AutoIncrStr()).To(matchers.Equal("auto_increment"))
	})

	o.Spec("AutoIncrBindValue", func(expect expect.Expectation, dialect gorp.MySQLDialect) {
		expect(dialect.AutoIncrBindValue()).To(matchers.Equal("null"))
	})

	o.Spec("AutoIncrInsertSuffix", func(expect expect.Expectation, dialect gorp.MySQLDialect) {
		expect(dialect.AutoIncrInsertSuffix(nil)).To(matchers.Equal(""))
	})

	o.Group("CreateTableSuffix", func() {
		o.Group("with an empty engine", func() {
			o.BeforeEach(func(expect expect.Expectation, dialect gorp.MySQLDialect) (expect.Expectation, gorp.MySQLDialect) {
				dialect.Engine = ""
				return expect, dialect
			})
			o.Spec("panics", func(expect expect.Expectation, dialect gorp.MySQLDialect) {
				expect(func() { dialect.CreateTableSuffix() }).To(Panic())
			})
		})

		o.Group("with an empty encoding", func() {
			o.BeforeEach(func(expect expect.Expectation, dialect gorp.MySQLDialect) (expect.Expectation, gorp.MySQLDialect) {
				dialect.Encoding = ""
				return expect, dialect
			})
			o.Spec("panics", func(expect expect.Expectation, dialect gorp.MySQLDialect) {
				expect(func() { dialect.CreateTableSuffix() }).To(Panic())
			})
		})

		o.Spec("with an engine and an encoding", func(expect expect.Expectation, dialect gorp.MySQLDialect) {
			expect(dialect.CreateTableSuffix()).To(matchers.Equal(" engine=foo charset=bar"))
		})
	})

	o.Spec("CreateIndexSuffix", func(expect expect.Expectation, dialect gorp.MySQLDialect) {
		expect(dialect.CreateIndexSuffix()).To(matchers.Equal("using"))
	})

	o.Spec("DropIndexSuffix", func(expect expect.Expectation, dialect gorp.MySQLDialect) {
		expect(dialect.DropIndexSuffix()).To(matchers.Equal("on"))
	})

	o.Spec("TruncateClause", func(expect expect.Expectation, dialect gorp.MySQLDialect) {
		expect(dialect.TruncateClause()).To(matchers.Equal("truncate"))
	})

	o.Spec("SleepClause", func(expect expect.Expectation, dialect gorp.MySQLDialect) {
		expect(dialect.SleepClause(1 * time.Second)).To(matchers.Equal("sleep(1.000000)"))
		expect(dialect.SleepClause(100 * time.Millisecond)).To(matchers.Equal("sleep(0.100000)"))
	})

	o.Spec("BindVar", func(expect expect.Expectation, dialect gorp.MySQLDialect) {
		expect(dialect.BindVar(0)).To(matchers.Equal("?"))
	})

	o.Spec("QuoteField", func(expect expect.Expectation, dialect gorp.MySQLDialect) {
		expect(dialect.QuoteField("foo")).To(matchers.Equal("`foo`"))
	})

	o.Group("QuotedTableForQuery", func() {
		o.Spec("using the default schema", func(expect expect.Expectation, dialect gorp.MySQLDialect) {
			expect(dialect.QuotedTableForQuery("", "foo")).To(matchers.Equal("`foo`"))
		})

		o.Spec("with a supplied schema", func(expect expect.Expectation, dialect gorp.MySQLDialect) {
			expect(dialect.QuotedTableForQuery("foo", "bar")).To(matchers.Equal("foo.`bar`"))
		})
	})

	o.Spec("IfSchemaNotExists", func(expect expect.Expectation, dialect gorp.MySQLDialect) {
		expect(dialect.IfSchemaNotExists("foo", "bar")).To(matchers.Equal("foo if not exists"))
	})

	o.Spec("IfTableExists", func(expect expect.Expectation, dialect gorp.MySQLDialect) {
		expect(dialect.IfTableExists("foo", "bar", "baz")).To(matchers.Equal("foo if exists"))
	})

	o.Spec("IfTableNotExists", func(expect expect.Expectation, dialect gorp.MySQLDialect) {
		expect(dialect.IfTableNotExists("foo", "bar", "baz")).To(matchers.Equal("foo if not exists"))
	})
}

type panicMatcher struct {
}

func Panic() panicMatcher {
	return panicMatcher{}
}

func (m panicMatcher) Match(actual interface{}) (resultValue interface{}, err error) {
	switch f := actual.(type) {
	case func():
		panicked := false
		func() {
			defer func() {
				if r := recover(); r != nil {
					panicked = true
				}
			}()
			f()
		}()
		if panicked {
			return f, nil
		}
		return f, errors.New("function did not panic")
	default:
		return f, fmt.Errorf("%T is not func()", f)
	}
}
