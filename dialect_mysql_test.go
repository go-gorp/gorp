// Copyright 2012 James Cooper. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

//go:build !integration
// +build !integration

package gorp_test

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/go-gorp/gorp/v3"
	"github.com/poy/onpar"
	"github.com/poy/onpar/expect"
	"github.com/poy/onpar/matchers"
)

func TestMySQLDialect(t *testing.T) {
	type testContext struct {
		expect  expect.Expectation
		dialect gorp.MySQLDialect
	}

	o := onpar.BeforeEach(onpar.New(t), func(t *testing.T) testContext {
		return testContext{
			expect: expect.New(t),
			dialect: gorp.MySQLDialect{
				Engine:   "foo",
				Encoding: "bar",
			},
		}
	})
	defer o.Run()

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
			o.Spec(t.name, func(tt testContext) {
				typ := reflect.TypeOf(t.value)
				sqlType := tt.dialect.ToSqlType(typ, t.maxSize, t.autoIncr)
				tt.expect(sqlType).To(matchers.Equal(t.expected))
			})
		}
	})

	o.Spec("AutoIncrStr", func(tt testContext) {
		tt.expect(tt.dialect.AutoIncrStr()).To(matchers.Equal("auto_increment"))
	})

	o.Spec("AutoIncrBindValue", func(tt testContext) {
		tt.expect(tt.dialect.AutoIncrBindValue()).To(matchers.Equal("null"))
	})

	o.Spec("AutoIncrInsertSuffix", func(tt testContext) {
		tt.expect(tt.dialect.AutoIncrInsertSuffix(nil)).To(matchers.Equal(""))
	})

	o.Group("CreateTableSuffix", func() {
		o.Group("with an empty engine", func() {
			o := onpar.BeforeEach(o, func(tt testContext) testContext {
				tt.dialect.Engine = ""
				return tt
			})
			o.Spec("panics", func(tt testContext) {
				tt.expect(func() { tt.dialect.CreateTableSuffix() }).To(Panic())
			})
		})

		o.Group("with an empty encoding", func() {
			o := onpar.BeforeEach(o, func(tt testContext) testContext {
				tt.dialect.Encoding = ""
				return tt
			})
			o.Spec("panics", func(tt testContext) {
				tt.expect(func() { tt.dialect.CreateTableSuffix() }).To(Panic())
			})
		})

		o.Spec("with an engine and an encoding", func(tt testContext) {
			tt.expect(tt.dialect.CreateTableSuffix()).To(matchers.Equal(" engine=foo charset=bar"))
		})
	})

	o.Spec("CreateIndexSuffix", func(tt testContext) {
		tt.expect(tt.dialect.CreateIndexSuffix()).To(matchers.Equal("using"))
	})

	o.Spec("DropIndexSuffix", func(tt testContext) {
		tt.expect(tt.dialect.DropIndexSuffix()).To(matchers.Equal("on"))
	})

	o.Spec("TruncateClause", func(tt testContext) {
		tt.expect(tt.dialect.TruncateClause()).To(matchers.Equal("truncate"))
	})

	o.Spec("SleepClause", func(tt testContext) {
		tt.expect(tt.dialect.SleepClause(1 * time.Second)).To(matchers.Equal("sleep(1.000000)"))
		tt.expect(tt.dialect.SleepClause(100 * time.Millisecond)).To(matchers.Equal("sleep(0.100000)"))
	})

	o.Spec("BindVar", func(tt testContext) {
		tt.expect(tt.dialect.BindVar(0)).To(matchers.Equal("?"))
	})

	o.Spec("QuoteField", func(tt testContext) {
		tt.expect(tt.dialect.QuoteField("foo")).To(matchers.Equal("`foo`"))
	})

	o.Group("QuotedTableForQuery", func() {
		o.Spec("using the default schema", func(tt testContext) {
			tt.expect(tt.dialect.QuotedTableForQuery("", "foo")).To(matchers.Equal("`foo`"))
		})

		o.Spec("with a supplied schema", func(tt testContext) {
			tt.expect(tt.dialect.QuotedTableForQuery("foo", "bar")).To(matchers.Equal("foo.`bar`"))
		})
	})

	o.Spec("IfSchemaNotExists", func(tt testContext) {
		tt.expect(tt.dialect.IfSchemaNotExists("foo", "bar")).To(matchers.Equal("foo if not exists"))
	})

	o.Spec("IfTableExists", func(tt testContext) {
		tt.expect(tt.dialect.IfTableExists("foo", "bar", "baz")).To(matchers.Equal("foo if exists"))
	})

	o.Spec("IfTableNotExists", func(tt testContext) {
		tt.expect(tt.dialect.IfTableNotExists("foo", "bar", "baz")).To(matchers.Equal("foo if not exists"))
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
