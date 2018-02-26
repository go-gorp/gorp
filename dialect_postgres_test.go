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
	"reflect"
	"testing"
	"time"

	"github.com/apoydence/onpar"
	"github.com/apoydence/onpar/expect"
	"github.com/apoydence/onpar/matchers"
	"github.com/go-gorp/gorp"
)

func TestPostgresDialect(t *testing.T) {
	o := onpar.New()
	defer o.Run(t)

	o.BeforeEach(func(t *testing.T) (expect.Expectation, gorp.PostgresDialect) {
		return expect.New(t), gorp.PostgresDialect{
			LowercaseFields: false,
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
			{"int8", int8(1), 0, false, "integer"},
			{"uint8", uint8(1), 0, false, "integer"},
			{"int16", int16(1), 0, false, "integer"},
			{"uint16", uint16(1), 0, false, "integer"},
			{"int32", int32(1), 0, false, "integer"},
			{"int (treated as int32)", int(1), 0, false, "integer"},
			{"uint32", uint32(1), 0, false, "integer"},
			{"uint (treated as uint32)", uint(1), 0, false, "integer"},
			{"int64", int64(1), 0, false, "bigint"},
			{"uint64", uint64(1), 0, false, "bigint"},
			{"float32", float32(1), 0, false, "real"},
			{"float64", float64(1), 0, false, "double precision"},
			{"[]uint8", []uint8{1}, 0, false, "bytea"},
			{"NullInt64", sql.NullInt64{}, 0, false, "bigint"},
			{"NullFloat64", sql.NullFloat64{}, 0, false, "double precision"},
			{"NullBool", sql.NullBool{}, 0, false, "boolean"},
			{"Time", time.Time{}, 0, false, "timestamp with time zone"},
			{"default-size string", "", 0, false, "text"},
			{"sized string", "", 50, false, "varchar(50)"},
			{"large string", "", 1024, false, "varchar(1024)"},
		}
		for _, t := range tests {
			o.Spec(t.name, func(expect expect.Expectation, dialect gorp.PostgresDialect) {
				typ := reflect.TypeOf(t.value)
				sqlType := dialect.ToSqlType(typ, t.maxSize, t.autoIncr)
				expect(sqlType).To(matchers.Equal(t.expected))
			})
		}
	})

	o.Spec("AutoIncrStr", func(expect expect.Expectation, dialect gorp.PostgresDialect) {
		expect(dialect.AutoIncrStr()).To(matchers.Equal(""))
	})

	o.Spec("AutoIncrBindValue", func(expect expect.Expectation, dialect gorp.PostgresDialect) {
		expect(dialect.AutoIncrBindValue()).To(matchers.Equal("default"))
	})

	o.Spec("AutoIncrInsertSuffix", func(expect expect.Expectation, dialect gorp.PostgresDialect) {
		cm := gorp.ColumnMap{
			ColumnName: "foo",
		}
		expect(dialect.AutoIncrInsertSuffix(&cm)).To(matchers.Equal(` returning "foo"`))
	})

	o.Spec("CreateTableSuffix", func(expect expect.Expectation, dialect gorp.PostgresDialect) {
		expect(dialect.CreateTableSuffix()).To(matchers.Equal(""))
	})

	o.Spec("CreateIndexSuffix", func(expect expect.Expectation, dialect gorp.PostgresDialect) {
		expect(dialect.CreateIndexSuffix()).To(matchers.Equal("using"))
	})

	o.Spec("DropIndexSuffix", func(expect expect.Expectation, dialect gorp.PostgresDialect) {
		expect(dialect.DropIndexSuffix()).To(matchers.Equal(""))
	})

	o.Spec("TruncateClause", func(expect expect.Expectation, dialect gorp.PostgresDialect) {
		expect(dialect.TruncateClause()).To(matchers.Equal("truncate"))
	})

	o.Spec("SleepClause", func(expect expect.Expectation, dialect gorp.PostgresDialect) {
		expect(dialect.SleepClause(1 * time.Second)).To(matchers.Equal("pg_sleep(1.000000)"))
		expect(dialect.SleepClause(100 * time.Millisecond)).To(matchers.Equal("pg_sleep(0.100000)"))
	})

	o.Spec("BindVar", func(expect expect.Expectation, dialect gorp.PostgresDialect) {
		expect(dialect.BindVar(0)).To(matchers.Equal("$1"))
		expect(dialect.BindVar(4)).To(matchers.Equal("$5"))
	})

	o.Group("QuoteField", func() {
		o.Spec("By default, case is preserved", func(expect expect.Expectation, dialect gorp.PostgresDialect) {
			expect(dialect.QuoteField("Foo")).To(matchers.Equal(`"Foo"`))
			expect(dialect.QuoteField("bar")).To(matchers.Equal(`"bar"`))
		})

		o.Group("With LowercaseFields set to true", func() {
			o.BeforeEach(func(expect expect.Expectation, dialect gorp.PostgresDialect) (expect.Expectation, gorp.PostgresDialect) {
				dialect.LowercaseFields = true
				return expect, dialect
			})

			o.Spec("fields are lowercased", func(expect expect.Expectation, dialect gorp.PostgresDialect) {
				expect(dialect.QuoteField("Foo")).To(matchers.Equal(`"foo"`))
			})
		})
	})

	o.Group("QuotedTableForQuery", func() {
		o.Spec("using the default schema", func(expect expect.Expectation, dialect gorp.PostgresDialect) {
			expect(dialect.QuotedTableForQuery("", "foo")).To(matchers.Equal(`"foo"`))
		})

		o.Spec("with a supplied schema", func(expect expect.Expectation, dialect gorp.PostgresDialect) {
			expect(dialect.QuotedTableForQuery("foo", "bar")).To(matchers.Equal(`foo."bar"`))
		})
	})

	o.Spec("IfSchemaNotExists", func(expect expect.Expectation, dialect gorp.PostgresDialect) {
		expect(dialect.IfSchemaNotExists("foo", "bar")).To(matchers.Equal("foo if not exists"))
	})

	o.Spec("IfTableExists", func(expect expect.Expectation, dialect gorp.PostgresDialect) {
		expect(dialect.IfTableExists("foo", "bar", "baz")).To(matchers.Equal("foo if exists"))
	})

	o.Spec("IfTableNotExists", func(expect expect.Expectation, dialect gorp.PostgresDialect) {
		expect(dialect.IfTableNotExists("foo", "bar", "baz")).To(matchers.Equal("foo if not exists"))
	})
}
