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
  rm tsoda
  find ./tmp -name *.sqlite* -delete || true
}
# defer cleanup, so it will be executed even after premature exit
trap cleanup EXIT

function test {
  export SODA_DIALECT=$1

  echo ""
  echo "######################################################################"
  echo "### Running unit tests for $SODA_DIALECT"
  ./tsoda drop -e $SODA_DIALECT -c ./database.yml -p ./testdata/migrations
  ./tsoda create -e $SODA_DIALECT -c ./database.yml -p ./testdata/migrations
  ./tsoda migrate -e $SODA_DIALECT -c ./database.yml -p ./testdata/migrations
  go test -cover -race -tags sqlite -count=1 $args ./...
}

function debug_test {
  export SODA_DIALECT=$1

  echo ""
  echo "######################################################################"
  echo "### Running unit tests for $SODA_DIALECT"
  ./tsoda drop -e $SODA_DIALECT -c ./database.yml -p ./testdata/migrations
  ./tsoda create -e $SODA_DIALECT -c ./database.yml -p ./testdata/migrations
  ./tsoda migrate -e $SODA_DIALECT -c ./database.yml -p ./testdata/migrations
  dlv test github.com/gobuffalo/pop
}

dialects="postgres cockroach mysql sqlite"

$COMPOSE up --wait

go build -v -tags sqlite -o tsoda ./soda

for dialect in $dialects; do
	if [ "$DEBUG" = "YES" ]; then
		debug_test ${dialect}
	else
		test ${dialect}
	fi
done
