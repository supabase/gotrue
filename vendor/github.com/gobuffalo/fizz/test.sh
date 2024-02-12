#!/bin/bash

set -e

# NOTE: See also docker-compose.yml and database.yml to configure database
# properties.
export MYSQL_PORT=3307
export COCKROACH_PORT=26258

COMPOSE=docker-compose
which docker-compose || COMPOSE="docker compose"

args=$@

function cleanup {
    echo "Cleanup resources..."
    $COMPOSE down
    docker volume prune -f
    find ./tmp -name *.sqlite* -delete || true
}
# defer cleanup, so it will be executed even after premature exit
trap cleanup EXIT

function test {
  export SODA_DIALECT=$1

  echo ""
  echo "######################################################################"
  echo "### Running unit tests for $SODA_DIALECT"
  soda drop -e $SODA_DIALECT
  soda create -e $SODA_DIALECT
  soda migrate -e $SODA_DIALECT -p ./testdata/migrations
  go test -tags sqlite -count=1 $args ./...

  echo ""
  echo "######################################################################"
  echo "### Running e2e tests for $1"
  soda drop -e $SODA_DIALECT
  soda create -e $SODA_DIALECT
  pushd testdata/e2e; go test -tags sqlite,e2e -count=1 $args ./...; popd
}


$COMPOSE up --wait

go install -tags sqlite github.com/gobuffalo/pop/v6/soda@latest

test "sqlite"
test "postgres"
test "cockroach"
test "mysql"

# Does not appear to be implemented in pop:
# test "sqlserver"
