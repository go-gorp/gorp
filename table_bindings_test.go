// Copyright 2012 James Cooper. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package gorp

import (
	"reflect"
	"testing"
)

func Test_getZeroValueStringForSQL(t *testing.T) {
	type args struct {
		t reflect.Type
	}
	tests := []struct {
		name  string
		args  args
		wantS string
	}{
		{"bool", args{reflect.TypeOf(true)}, "false"},
		{"int", args{reflect.TypeOf(-5)}, "0"},
		{"uint", args{reflect.TypeOf(100)}, "0"},
		{"float", args{reflect.TypeOf(12.3)}, "0.0"},
		{"string", args{reflect.TypeOf("gorp")}, "''"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotS := getZeroValueStringForSQL(tt.args.t); gotS != tt.wantS {
				t.Errorf("getZerovalueStringForSQL() = %v, want %v", gotS, tt.wantS)
			}
		})
	}
}

func Test_getValueAsType(t *testing.T) {
	type args struct {
		t     reflect.Type
		value string
	}
	tests := []struct {
		name    string
		args    args
		wantS   string
		wantErr bool
	}{
		{"int", args{reflect.TypeOf(1), "774"}, "774", false},
		{"int with single quotation", args{reflect.TypeOf(1), "'774'"}, "774", false},
		{"int of empty string", args{reflect.TypeOf(1), ""}, "", true},
		{"float", args{reflect.TypeOf(1.00), "1.23"}, "1.23", false},
		{"float with single quotation", args{reflect.TypeOf(1.00), "'1.23'"}, "1.23", false},
		{"float of empty string", args{reflect.TypeOf(1.00), ""}, "", true},
		{"string", args{reflect.TypeOf(""), "Gopher"}, "'Gopher'", false},
		{"string with single quotation", args{reflect.TypeOf(""), "'Gopher'"}, "'Gopher'", false},
		{"string of empty string", args{reflect.TypeOf(""), ""}, "''", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotS, err := getValueAsType(tt.args.t, tt.args.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("getValueAsType() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotS != tt.wantS {
				t.Errorf("getValueAsType() = %v, want %v", gotS, tt.wantS)
			}
		})
	}
}
