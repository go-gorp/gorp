#!/bin/sh

# on macs, you may need to:
# export GOBUILDFLAG=-ldflags -linkmode=external

# Using "-n", the environment variables will be set but tests will not run.
# You can "dot" this script, then use go test directly, which will give verbose progress info.
if [ "$1" = "-n" ]; then
    NOOP=":"
    shift
fi

set -e

testMysql() {
    export GORP_TEST_DSN=gorptest/gorptest/gorptest
    export GORP_TEST_DIALECT=mysql
    $NOOP go test $GOBUILDFLAG .

    export GORP_TEST_DSN=gorptest:gorptest@/gorptest
    export GORP_TEST_DIALECT=gomysql
    $NOOP go test $GOBUILDFLAG .
}

testPostgresql() {
    export GORP_TEST_DSN="user=gorptest password=gorptest dbname=gorptest sslmode=disable"
    export GORP_TEST_DIALECT=postgres
    $NOOP go test $GOBUILDFLAG .
}

testSqlite() {
    export GORP_TEST_DSN=/tmp/gorptest.bin
    export GORP_TEST_DIALECT=sqlite
    $NOOP go test $GOBUILDFLAG .
}

case "$1" in
  mysql)           testMysql ;;
  psql | postgresql) testPostgresql ;;
  sqlite)          testSqlite ;;
  *)               testMysql ; testPostgresql ; testSqlite ;;
esac
