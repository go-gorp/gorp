// Copyright 2012 James Cooper. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package gorp

import "testing"

func Test_getZeroValueStringForSQL(t *testing.T) {
	type args struct {
		i interface{}
	}
	tests := []struct {
		name  string
		args  args
		wantS string
	}{
		{"bool", args{i: true}, "false"},
		{"int", args{i: -5}, "0"},
		{"uint", args{i: 100}, "0"},
		{"float", args{i: 12.3}, "0.0"},
		{"string", args{i: "gorp"}, "''"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotS := getZeroValueStringForSQL(tt.args.i); gotS != tt.wantS {
				t.Errorf("getZerovalueStringForSQL() = %v, want %v", gotS, tt.wantS)
			}
		})
	}
}
