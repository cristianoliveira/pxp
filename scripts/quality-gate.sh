#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
COVERAGE_PROFILE=${PXP_COVERAGE_PROFILE:-"$ROOT_DIR/coverage.out"}
MIN_COVERAGE=${PXP_MIN_COVERAGE:-78.0}

coverage_only=false
complexity_only=false
for arg in "$@"; do
  case "$arg" in
    --coverage-only) coverage_only=true ;;
    --complexity-only) complexity_only=true ;;
    *) echo "usage: $0 [--coverage-only|--complexity-only]" >&2; exit 2 ;;
  esac
done

if [[ "$complexity_only" == false ]]; then
  if [[ ! -f "$COVERAGE_PROFILE" || "${PXP_USE_EXISTING_COVERAGE:-0}" != 1 ]]; then
    (cd "$ROOT_DIR" && go test ./internal/... ./tests/smoke -coverprofile="$COVERAGE_PROFILE")
  fi

  total=$(go tool cover -func="$COVERAGE_PROFILE" | awk '$1 == "total:" {gsub("%", "", $3); print $3}')
  if [[ -z "$total" ]]; then
    echo "quality: could not read total coverage from $COVERAGE_PROFILE" >&2
    exit 1
  fi
  if ! awk -v actual="$total" -v minimum="$MIN_COVERAGE" 'BEGIN { exit !(actual + 0 >= minimum + 0) }'; then
    printf 'quality: coverage %.1f%% is below minimum %.1f%%\n' "$total" "$MIN_COVERAGE" >&2
    exit 1
  fi
  printf 'quality: coverage %.1f%% (minimum %.1f%%)\n' "$total" "$MIN_COVERAGE"
fi

if [[ "$coverage_only" == false ]]; then
  (cd "$ROOT_DIR" && golangci-lint run)
  echo "quality: complexity and function-length thresholds passed"
fi
