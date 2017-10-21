package gorp_test

import (
	"reflect"

	// ginkgo/gomega functions read better as dot-imports.
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	"github.com/go-gorp/gorp"
)

var _ = Describe("CrateDialect", func() {

	var dialect gorp.CrateDialect

	JustBeforeEach(func() {
		dialect = gorp.CrateDialect{}
	})

	DescribeTable("ToSqlType",
		func(value interface{}, maxsize int, autoIncr bool, expected string) {
			typ := reflect.TypeOf(value)
			sqlType := dialect.ToSqlType(typ, maxsize, autoIncr)
			Expect(sqlType).To(Equal(expected))
		},
		Entry("bool", true, 0, false, "boolean"),
		Entry("int8", int8(1), 0, false, "byte"),
		Entry("uint8", uint8(1), 0, false, "integer"),
		Entry("int16", int16(1), 0, false, "short"),
		Entry("uint16", uint16(1), 0, false, "integer"),
		Entry("int32", int32(1), 0, false, "integer"),
		Entry("uint32", uint32(1), 0, false, "integer"),
		Entry("int64", int64(1), 0, false, "long"),
		Entry("uint64", uint64(1), 0, false, "integer"),
		Entry("float32", float32(1), 0, false, "float"),
		Entry("float64", float64(1), 0, false, "double"),
		Entry("string", "", 0, false, "string"),
		//TODO(inge4pres) add net.IPAddr,
	)

	Describe("AutoIncrStr", func() {
		It("returns the auto increment string", func() {
			Expect(dialect.AutoIncrStr()).To(Equal("PRIMARY KEY"))
		})
	})

	Describe("TruncateClause", func() {
		It("returns the clause for truncating a table", func() {
			Expect(dialect.TruncateClause()).To(Equal("DELETE FROM"))
		})
	})

	Describe("BindVar", func() {
		It("returns the variable binding sequence", func() {
			Expect(dialect.BindVar(0)).To(Equal("?"))
		})
	})

	Describe("QuoteField", func() {
		It("returns the argument quoted as a field", func() {
			Expect(dialect.QuoteField("foo")).To(Equal("'foo'"))
		})
	})

	Describe("IfSchemaNotExists", func() {
		It("appends 'IF NOT EXISTS' to the command", func() {
			Expect(dialect.IfSchemaNotExists("foo", "bar")).To(Equal("foo IF NOT EXISTS"))
		})
	})

	Describe("IfTableExists", func() {
		It("appends 'IF EXISTS' to the command", func() {
			Expect(dialect.IfTableExists("foo", "bar", "baz")).To(Equal("foo IF EXISTS"))
		})
	})

	Describe("IfTableNotExists", func() {
		It("appends 'IF NOT EXISTS' to the command", func() {
			Expect(dialect.IfTableNotExists("foo", "bar", "baz")).To(Equal("foo IF NOT EXISTS"))
		})
	})
})
