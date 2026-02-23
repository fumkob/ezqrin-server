#!/usr/bin/env bash
# scripts/run-tests.sh
# Run all tests (unit + integration) and generate coverage report.
#
# Usage:
#   ./scripts/run-tests.sh                    # All tests + coverage
#   ./scripts/run-tests.sh --unit-only        # Unit tests only (no integration tag)
#   ./scripts/run-tests.sh --integration-only # Integration tests only
#
# Environment variables:
#   DB_HOST         PostgreSQL host (default: localhost)
#   REDIS_HOST      Redis host (default: localhost)
#   TEST_REDIS_DB   Redis DB number for tests (default: 1)
#   COVERAGE_DIR    Directory for coverage output (default: ./coverage)

set -euo pipefail

COVERAGE_DIR="${COVERAGE_DIR:-./coverage}"
UNIT_ONLY=false
INTEGRATION_ONLY=false

for arg in "$@"; do
  case $arg in
    --unit-only)        UNIT_ONLY=true ;;
    --integration-only) INTEGRATION_ONLY=true ;;
  esac
done

mkdir -p "$COVERAGE_DIR"

echo "=== ezQRin Server Test Runner ==="
echo "Coverage output: $COVERAGE_DIR"

if [ "$INTEGRATION_ONLY" = false ]; then
  echo ""
  echo "--- Unit Tests ---"
  go test -count=1 -coverprofile="$COVERAGE_DIR/unit.out" ./...
  echo "Unit tests PASSED"
fi

if [ "$UNIT_ONLY" = false ]; then
  echo ""
  echo "--- Integration + E2E Tests ---"
  go test -p 1 -count=1 -tags=integration -coverprofile="$COVERAGE_DIR/integration.out" ./...
  echo "Integration tests PASSED"
fi

# Merge coverage profiles if both were run
if [ "$UNIT_ONLY" = false ] && [ "$INTEGRATION_ONLY" = false ]; then
  echo ""
  echo "--- Merging Coverage Profiles ---"
  cat "$COVERAGE_DIR/unit.out" > "$COVERAGE_DIR/merged.out"
  tail -n +2 "$COVERAGE_DIR/integration.out" >> "$COVERAGE_DIR/merged.out"
  COVERAGE_FILE="$COVERAGE_DIR/merged.out"
else
  COVERAGE_FILE="$COVERAGE_DIR/unit.out"
  [ "$INTEGRATION_ONLY" = true ] && COVERAGE_FILE="$COVERAGE_DIR/integration.out"
fi

echo ""
echo "--- Coverage Summary ---"
go tool cover -func="$COVERAGE_FILE" | tail -1

echo ""
echo "--- Generating HTML Report ---"
go tool cover -html="$COVERAGE_FILE" -o "$COVERAGE_DIR/coverage.html"
echo "HTML report: $COVERAGE_DIR/coverage.html"
