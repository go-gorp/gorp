#!/bin/sh

set -e 

export GORP_TEST_DSN=gorptest/gorptest/gorptest
export GORP_TEST_DIALECT=mysql
go test

export GORP_TEST_DSN="user=gorptest password=gorptest dbname=gorptest sslmode=disable"
export GORP_TEST_DIALECT=postgres
go test

export GORP_TEST_DSN=/tmp/gorptest.bin
export GORP_TEST_DIALECT=sqlite
go test 
