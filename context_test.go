// Copyright 2012 James Cooper. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

//go:build integration
// +build integration

package borp_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Drivers that don't support cancellation.
var unsupportedDrivers map[string]bool = map[string]bool{
	"mymysql": true,
}

type SleepDialect interface {
	// string to sleep for d duration
	SleepClause(d time.Duration) string
}

func TestWithNotCanceledContext(t *testing.T) {
	dbmap := initDBMap(t)
	defer dropAndClose(dbmap)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := dbmap.ExecContext(ctx, "SELECT 1")
	assert.Nil(t, err)
}

func TestWithCanceledContext(t *testing.T) {
	dialect, driver := dialectAndDriver()
	if unsupportedDrivers[driver] {
		t.Skipf("Cancellation is not yet supported by all drivers. Not known to be supported in %s.", driver)
	}

	sleepDialect, ok := dialect.(SleepDialect)
	if !ok {
		t.Skipf("Sleep is not supported in all dialects. Not known to be supported in %s.", driver)
	}

	dbmap := initDBMap(t)
	defer dropAndClose(dbmap)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	startTime := time.Now()

	_, err := dbmap.ExecContext(ctx, "SELECT "+sleepDialect.SleepClause(1*time.Second))

	if d := time.Since(startTime); d > 500*time.Millisecond {
		t.Errorf("too long execution time: %s", d)
	}

	switch driver {
	default:
		if err != context.DeadlineExceeded {
			t.Errorf("expected context.DeadlineExceeded, got %v", err)
		}
	}
}
