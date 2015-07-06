#!/bin/bash

# on macs, you may need to:
# export GOBUILDFLAG=-ldflags -linkmode=external

coveralls_testflags="-v -covermode=count -coverprofile=coverage.out"

set -e

echo "Testing against mysql"
export GORP_TEST_DSN=gorptest/gorptest/gorptest
export GORP_TEST_DIALECT=mysql
go test $coveralls_testflags $GOBUILDFLAG $@ .

echo "Testing against gomysql"
export GORP_TEST_DSN=gorptest:gorptest@/gorptest
export GORP_TEST_DIALECT=gomysql
go test $coveralls_testflags $GOBUILDFLAG $@ .

echo "Testing against postgres"
export GORP_TEST_DSN="user=gorptest password=gorptest dbname=gorptest sslmode=disable"
export GORP_TEST_DIALECT=postgres
go test $coveralls_testflags $GOBUILDFLAG $@ .

echo "Testing against sqlite"
export GORP_TEST_DSN=/tmp/gorptest.bin
export GORP_TEST_DIALECT=sqlite
go test $coveralls_testflags $GOBUILDFLAG $@ .

if [[ `go version` == *"1.4"* ]]
then
	$HOME/gopath/bin/goveralls -covermode=count -coverprofile=coverage.out -service=travis-ci -repotoken $COVERALLS_TOKEN
fi
