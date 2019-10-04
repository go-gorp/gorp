// Copyright 2012 James Cooper. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// +build integration

package gorp_test

import (
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

func TestDbMap_Select_expandSliceArgs(t *testing.T) {
	tests := []struct {
		description string
		query       string
		args        []interface{}
		wantLen     int
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
			wantLen: 1,
		},
		{
			description: "it should handle slice placeholders correctly with custom types",
			query: `
SELECT 1 FROM crazy_table
WHERE field2 IN (:FieldStringList)
AND field12 IN (:FieldIntList)
`,
			args: []interface{}{
				map[string]interface{}{
					"FieldStringList": customType1{"h", "e", "y"},
					"FieldIntList":    customType2{1, 2, 3, 4},
				},
			},
			wantLen: 3,
		},
	}

	type dataFormat struct {
		Field1  int     `db:"field1"`
		Field2  string  `db:"field2"`
		Field3  uint    `db:"field3"`
		Field4  uint8   `db:"field4"`
		Field5  uint16  `db:"field5"`
		Field6  uint32  `db:"field6"`
		Field7  uint64  `db:"field7"`
		Field8  int     `db:"field8"`
		Field9  int8    `db:"field9"`
		Field10 int16   `db:"field10"`
		Field11 int32   `db:"field11"`
		Field12 int64   `db:"field12"`
		Field13 float32 `db:"field13"`
		Field14 float64 `db:"field14"`
	}

	dbmap := newDbMap()
	dbmap.ExpandSliceArgs = true
	dbmap.AddTableWithName(dataFormat{}, "crazy_table")

	err := dbmap.CreateTables()
	if err != nil {
		panic(err)
	}
	defer dropAndClose(dbmap)

	err = dbmap.Insert(
		&dataFormat{
			Field1:  123,
			Field2:  "h",
			Field3:  1,
			Field4:  1,
			Field5:  1,
			Field6:  1,
			Field7:  1,
			Field8:  1,
			Field9:  1,
			Field10: 1,
			Field11: 1,
			Field12: 1,
			Field13: 1,
			Field14: 1,
		},
		&dataFormat{
			Field1:  124,
			Field2:  "e",
			Field3:  2,
			Field4:  2,
			Field5:  2,
			Field6:  2,
			Field7:  2,
			Field8:  2,
			Field9:  2,
			Field10: 2,
			Field11: 2,
			Field12: 2,
			Field13: 2,
			Field14: 2,
		},
		&dataFormat{
			Field1:  125,
			Field2:  "y",
			Field3:  3,
			Field4:  3,
			Field5:  3,
			Field6:  3,
			Field7:  3,
			Field8:  3,
			Field9:  3,
			Field10: 3,
			Field11: 3,
			Field12: 3,
			Field13: 3,
			Field14: 3,
		},
	)

	if err != nil {
		t.Fatal(err)
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			var dummy []int
			_, err := dbmap.Select(&dummy, tt.query, tt.args...)
			if err != nil {
				t.Fatal(err)
			}

			if len(dummy) != tt.wantLen {
				t.Errorf("wrong result count\ngot:  %d\nwant: %d", len(dummy), tt.wantLen)
			}
		})
	}
}
