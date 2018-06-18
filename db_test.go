// Copyright 2012 James Cooper. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// Package gorp provides a simple way to marshal Go structs to and from
// SQL databases.  It uses the database/sql package, and should work with any
// compliant database/sql driver.
//
// Source code and project home:
// https://github.com/go-gorp/gorp

package gorp

import (
	"reflect"
	"testing"
)

type customType1 []string

func (c customType1) ToStringSlice() []string {
	return []string(c)
}

type customType2 []int64

func (c customType2) ToInt64Slice() []int64 {
	return []int64(c)
}

func TestExpandSliceArgs(t *testing.T) {
	tests := []struct {
		description string
		query       string
		args        []interface{}
		wantQuery   string
		wantArgs    []interface{}
	}{
		{
			description: "it should handle slice placeholders correctly",
			query: `
SELECT 1 FROM crazy_table
WHERE field1 = :Field1
AND field2 IN (:FieldStringList)
AND field3 IN (:FieldUIntList)
AND field4 IN (:FieldUInt8List)
AND field5 IN (:FieldUInt16List)
AND field6 IN (:FieldUInt32List)
AND field7 IN (:FieldUInt64List)
AND field8 IN (:FieldIntList)
AND field9 IN (:FieldInt8List)
AND field10 IN (:FieldInt16List)
AND field11 IN (:FieldInt32List)
AND field12 IN (:FieldInt64List)
AND field13 IN (:FieldFloat32List)
AND field14 IN (:FieldFloat64List)
`,
			args: []interface{}{
				map[string]interface{}{
					"Field1":           123,
					"FieldStringList":  []string{"h", "e", "y"},
					"FieldUIntList":    []uint{1, 2, 3, 4},
					"FieldUInt8List":   []uint8{1, 2, 3, 4},
					"FieldUInt16List":  []uint16{1, 2, 3, 4},
					"FieldUInt32List":  []uint32{1, 2, 3, 4},
					"FieldUInt64List":  []uint64{1, 2, 3, 4},
					"FieldIntList":     []int{1, 2, 3, 4},
					"FieldInt8List":    []int8{1, 2, 3, 4},
					"FieldInt16List":   []int16{1, 2, 3, 4},
					"FieldInt32List":   []int32{1, 2, 3, 4},
					"FieldInt64List":   []int64{1, 2, 3, 4},
					"FieldFloat32List": []float32{1, 2, 3, 4},
					"FieldFloat64List": []float64{1, 2, 3, 4},
				},
			},
			wantQuery: `
SELECT 1 FROM crazy_table
WHERE field1 = :Field1
AND field2 IN (:FieldStringList0,:FieldStringList1,:FieldStringList2)
AND field3 IN (:FieldUIntList0,:FieldUIntList1,:FieldUIntList2,:FieldUIntList3)
AND field4 IN (:FieldUInt8List0,:FieldUInt8List1,:FieldUInt8List2,:FieldUInt8List3)
AND field5 IN (:FieldUInt16List0,:FieldUInt16List1,:FieldUInt16List2,:FieldUInt16List3)
AND field6 IN (:FieldUInt32List0,:FieldUInt32List1,:FieldUInt32List2,:FieldUInt32List3)
AND field7 IN (:FieldUInt64List0,:FieldUInt64List1,:FieldUInt64List2,:FieldUInt64List3)
AND field8 IN (:FieldIntList0,:FieldIntList1,:FieldIntList2,:FieldIntList3)
AND field9 IN (:FieldInt8List0,:FieldInt8List1,:FieldInt8List2,:FieldInt8List3)
AND field10 IN (:FieldInt16List0,:FieldInt16List1,:FieldInt16List2,:FieldInt16List3)
AND field11 IN (:FieldInt32List0,:FieldInt32List1,:FieldInt32List2,:FieldInt32List3)
AND field12 IN (:FieldInt64List0,:FieldInt64List1,:FieldInt64List2,:FieldInt64List3)
AND field13 IN (:FieldFloat32List0,:FieldFloat32List1,:FieldFloat32List2,:FieldFloat32List3)
AND field14 IN (:FieldFloat64List0,:FieldFloat64List1,:FieldFloat64List2,:FieldFloat64List3)
`,
			wantArgs: []interface{}{
				map[string]interface{}{
					"Field1":            123,
					"FieldStringList":   []string{"h", "e", "y"},
					"FieldStringList0":  "h",
					"FieldStringList1":  "e",
					"FieldStringList2":  "y",
					"FieldUIntList":     []uint{1, 2, 3, 4},
					"FieldUIntList0":    uint(1),
					"FieldUIntList1":    uint(2),
					"FieldUIntList2":    uint(3),
					"FieldUIntList3":    uint(4),
					"FieldUInt8List":    []uint8{1, 2, 3, 4},
					"FieldUInt8List0":   uint8(1),
					"FieldUInt8List1":   uint8(2),
					"FieldUInt8List2":   uint8(3),
					"FieldUInt8List3":   uint8(4),
					"FieldUInt16List":   []uint16{1, 2, 3, 4},
					"FieldUInt16List0":  uint16(1),
					"FieldUInt16List1":  uint16(2),
					"FieldUInt16List2":  uint16(3),
					"FieldUInt16List3":  uint16(4),
					"FieldUInt32List":   []uint32{1, 2, 3, 4},
					"FieldUInt32List0":  uint32(1),
					"FieldUInt32List1":  uint32(2),
					"FieldUInt32List2":  uint32(3),
					"FieldUInt32List3":  uint32(4),
					"FieldUInt64List":   []uint64{1, 2, 3, 4},
					"FieldUInt64List0":  uint64(1),
					"FieldUInt64List1":  uint64(2),
					"FieldUInt64List2":  uint64(3),
					"FieldUInt64List3":  uint64(4),
					"FieldIntList":      []int{1, 2, 3, 4},
					"FieldIntList0":     int(1),
					"FieldIntList1":     int(2),
					"FieldIntList2":     int(3),
					"FieldIntList3":     int(4),
					"FieldInt8List":     []int8{1, 2, 3, 4},
					"FieldInt8List0":    int8(1),
					"FieldInt8List1":    int8(2),
					"FieldInt8List2":    int8(3),
					"FieldInt8List3":    int8(4),
					"FieldInt16List":    []int16{1, 2, 3, 4},
					"FieldInt16List0":   int16(1),
					"FieldInt16List1":   int16(2),
					"FieldInt16List2":   int16(3),
					"FieldInt16List3":   int16(4),
					"FieldInt32List":    []int32{1, 2, 3, 4},
					"FieldInt32List0":   int32(1),
					"FieldInt32List1":   int32(2),
					"FieldInt32List2":   int32(3),
					"FieldInt32List3":   int32(4),
					"FieldInt64List":    []int64{1, 2, 3, 4},
					"FieldInt64List0":   int64(1),
					"FieldInt64List1":   int64(2),
					"FieldInt64List2":   int64(3),
					"FieldInt64List3":   int64(4),
					"FieldFloat32List":  []float32{1, 2, 3, 4},
					"FieldFloat32List0": float32(1),
					"FieldFloat32List1": float32(2),
					"FieldFloat32List2": float32(3),
					"FieldFloat32List3": float32(4),
					"FieldFloat64List":  []float64{1, 2, 3, 4},
					"FieldFloat64List0": float64(1),
					"FieldFloat64List1": float64(2),
					"FieldFloat64List2": float64(3),
					"FieldFloat64List3": float64(4),
				},
			},
		},
		{
			description: "it should handle slice placeholders correctly with custom types",
			query: `
SELECT 1 FROM crazy_table
WHERE field1 = :Field1
AND field2 IN (:FieldIntList)
AND field3 IN (:FieldStringList)
`,
			args: []interface{}{
				map[string]interface{}{
					"Field1":          123,
					"FieldIntList":    customType2{1, 2, 3, 4},
					"FieldStringList": customType1{"h", "e", "y"},
				},
			},
			wantQuery: `
SELECT 1 FROM crazy_table
WHERE field1 = :Field1
AND field2 IN (:FieldIntList0,:FieldIntList1,:FieldIntList2,:FieldIntList3)
AND field3 IN (:FieldStringList0,:FieldStringList1,:FieldStringList2)
`,
			wantArgs: []interface{}{
				map[string]interface{}{
					"Field1":           123,
					"FieldIntList":     customType2{1, 2, 3, 4},
					"FieldIntList0":    int64(1),
					"FieldIntList1":    int64(2),
					"FieldIntList2":    int64(3),
					"FieldIntList3":    int64(4),
					"FieldStringList":  customType1{"h", "e", "y"},
					"FieldStringList0": "h",
					"FieldStringList1": "e",
					"FieldStringList2": "y",
				},
			},
		},
		{
			description: "it should ignore empty slices",
			query: `
SELECT 1 FROM crazy_table
WHERE field1 IN (:FieldIntList)
`,
			args: []interface{}{
				map[string]interface{}{
					"FieldIntList": []int64{},
				},
			},
			wantQuery: `
SELECT 1 FROM crazy_table
WHERE field1 IN (:FieldIntList)
`,
			wantArgs: []interface{}{
				map[string]interface{}{
					"FieldIntList": []int64{},
				},
			},
		},
		{
			description: "it should ignore non-mappers",
			query: `
SELECT 1 FROM crazy_table
WHERE field1 = :Field1
AND field2 IN (:FieldIntList)
AND field3 IN (:FieldStringList)
`,
			args: []interface{}{
				123,
			},
			wantQuery: `
SELECT 1 FROM crazy_table
WHERE field1 = :Field1
AND field2 IN (:FieldIntList)
AND field3 IN (:FieldStringList)
`,
			wantArgs: []interface{}{
				123,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			expandSliceArgs(&tt.query, tt.args...)

			if tt.query != tt.wantQuery {
				t.Errorf("wrong query\ngot:  %s\nwant: %s", tt.query, tt.wantQuery)
			}

			if !reflect.DeepEqual(tt.wantArgs, tt.args) {
				t.Errorf("wrong args\ngot: %v\nwant: %v", tt.args, tt.wantArgs)
			}
		})
	}
}
