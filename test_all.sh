#!/bin/bash -e

# on macs, you may need to:
# export GOBUILDFLAG=-ldflags -linkmode=external

coveralls_testflags="-covermode=count -coverprofile=coverage.out"

echo "Running unit tests"
ginkgo -r -race -randomizeAllSpecs -keepGoing -- -test.run TestGorp

echo "Testing against gomysql"
export GORP_TEST_DSN="gorptest:gorptest@tcp(192.168.254.7:3306)/gorptest"
export GORP_TEST_DIALECT=gomysql
#go test $coveralls_testflags $GOBUILDFLAG $@ .
gotestsum --format testname --no-summary=skipped,failed -- $coveralls_testflags $GOBUILDFLAG .

echo "Testing against postgres"
export GORP_TEST_DSN="host=192.168.254.5 user=gorptest password=gorptest dbname=gorptest sslmode=disable"
export GORP_TEST_DIALECT=postgres
#go test $coveralls_testflags $GOBUILDFLAG $@ .
gotestsum --format testname --no-summary=skipped,failed -- $coveralls_testflags $GOBUILDFLAG .

echo "Testing against mymysql"
export GORP_TEST_DSN="tcp:192.168.254.7:3306*gorptest/gorptest/gorptest"
export GORP_TEST_DIALECT=mysql
gotestsum --format testname --no-summary=skipped,failed -- $coveralls_testflags $GOBUILDFLAG .
#go test $coveralls_testflags $GOBUILDFLAG -test.run=$@ .

echo "Testing against sqlite"
export GORP_TEST_DSN=/tmp/gorptest.bin
export GORP_TEST_DIALECT=sqlite
go test $coveralls_testflags $GOBUILDFLAG $@ .
rm -f /tmp/gorptest.bin

case $(go version) in
  *go1.13*)
    if [ "$(type -p goveralls)" != "" ]; then
	  goveralls -covermode=count -coverprofile=coverage.out -service=travis-ci
    elif [ -x $HOME/gopath/bin/goveralls ]; then
	  $HOME/gopath/bin/goveralls -covermode=count -coverprofile=coverage.out -service=travis-ci
    fi
  ;;
  *) ;;
esac%
