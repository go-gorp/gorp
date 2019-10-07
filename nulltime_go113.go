// Copyright 2012 James Cooper. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// +build go1.13

package gorp

import "database/sql"

// NullTime is provided for backward compatibility
// reasons, to make users of gorp.NullTime switch
// over to go 1.13's built-in sql.NullTime.
type NullTime = sql.NullTime
