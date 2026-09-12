#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
GATE="$ROOT_DIR/scripts/quality-gate.sh"
TMP_DIR=$(mktemp -d "${TMPDIR:-/tmp}/pxp-quality.XXXXXX")
trap 'rm -rf "$TMP_DIR"' EXIT

cat >"$TMP_DIR/fixture.go" <<'EOF'
package fixture

func Covered() {}
EOF
cat >"$TMP_DIR/pass.cover" <<EOF
mode: set
$TMP_DIR/fixture.go:3.1,3.19 1 1
EOF
cat >"$TMP_DIR/fail.cover" <<EOF
mode: set
$TMP_DIR/fixture.go:3.1,3.19 1 0
EOF

PXP_COVERAGE_PROFILE="$TMP_DIR/pass.cover" PXP_USE_EXISTING_COVERAGE=1 \
  PXP_MIN_COVERAGE=100 "$GATE" --coverage-only
if PXP_COVERAGE_PROFILE="$TMP_DIR/fail.cover" PXP_USE_EXISTING_COVERAGE=1 \
  PXP_MIN_COVERAGE=100 "$GATE" --coverage-only; then
  echo 'quality test: expected coverage failure' >&2
  exit 1
fi

mkdir -p "$TMP_DIR/complexity"
cat >"$TMP_DIR/complexity/go.mod" <<EOF
module example.com/quality-fixture

go 1.25
EOF
cat >"$TMP_DIR/complexity/pass.go" <<'EOF'
package fixture

func Small() int {
	return 1
}
EOF
{
  printf 'package fixture\n\nfunc TooComplex(value int) int {\n'
  for index in $(seq 1 41); do
    printf '\tif value == %d { return %d }\n' "$index" "$index"
  done
  printf '\treturn 0\n}\n'
} >"$TMP_DIR/complexity/fail.go"
{
  printf 'package fixture\n\nfunc TooLong(value int) int {\n'
  for index in $(seq 1 220); do
    printf '\tvalue += %d\n' "$index"
  done
  printf '\treturn value\n}\n'
} >"$TMP_DIR/complexity/long.go"

lint_output=$(cd "$TMP_DIR/complexity" && golangci-lint run --config "$ROOT_DIR/.golangci.yml" --issues-exit-code 0 ./... 2>&1)
if grep -q cyclop <<<"$lint_output"; then
  echo 'quality: complexity failure boundary passed'
else
  printf '%s\n' "$lint_output" >&2
  echo 'quality test: expected complexity failure' >&2
  exit 1
fi

if grep -q funlen <<<"$lint_output"; then
  echo 'quality: function-length failure boundary passed'
else
  printf '%s\n' "$lint_output" >&2
  echo 'quality test: expected function-length failure' >&2
  exit 1
fi

echo 'quality: boundary tests passed'
