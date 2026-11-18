#!/usr/bin/env bash
# Runs the tests of all three services. Stops at the first failure.
set -euo pipefail
root="$(cd "$(dirname "$0")/.." && pwd)"

echo "==> menu-api (Python)"
(
  cd "$root/menu-api"
  if [ -d .venv ]; then
    . .venv/bin/activate
  fi
  pytest -q
)

echo "==> orders-api (Go)"
(cd "$root/orders-api" && go test ./...)

echo "==> kitchen-worker (Java)"
(cd "$root/kitchen-worker" && mvn -q test)

echo "==> stats-api (PHP)"
(
  cd "$root/stats-api"
  if [ ! -d vendor ]; then
    composer install --no-interaction
  fi
  vendor/bin/phpunit
)

echo "==> web (React)"
(
  cd "$root/web"
  if [ ! -d node_modules ]; then
    npm install
  fi
  npm test
)

echo "All tests passed."
