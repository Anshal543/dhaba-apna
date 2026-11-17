#!/usr/bin/env bash
# Runs the tests of all three services. Stops at the first failure.
set -euo pipefail
root="$(cd "$(dirname "$0")/.." && pwd)"

echo "==> menu-api (Python)"
(cd "$root/menu-api" && { [ -d .venv ] && . .venv/bin/activate; } && pytest -q)

echo "==> orders-api (Go)"
(cd "$root/orders-api" && go test ./...)

echo "==> kitchen-worker (Java)"
(cd "$root/kitchen-worker" && mvn -q test)

echo "==> stats-api (PHP)"
(cd "$root/stats-api" && { [ -d vendor ] || composer install --no-interaction; } && vendor/bin/phpunit)

echo "==> web (React)"
(cd "$root/web" && { [ -d node_modules ] || npm install; } && npm test)

echo "All tests passed."
