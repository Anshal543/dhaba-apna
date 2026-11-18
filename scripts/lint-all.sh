#!/usr/bin/env bash
set -euo pipefail
root="$(cd "$(dirname "$0")/.." && pwd)"

echo "==> menu-api (ruff)"
(cd "$root/menu-api" && ruff check .)

echo "==> orders-api (go vet)"
(cd "$root/orders-api" && go vet ./...)

echo "==> kitchen-worker (javac)"
(cd "$root/kitchen-worker" && mvn -B -q -DskipTests compile)

echo "==> stats-api (php -l)"
while IFS= read -r file; do
  php -l "$file" >/dev/null
done < <(find "$root/stats-api/app" "$root/stats-api/bootstrap" "$root/stats-api/config" "$root/stats-api/routes" "$root/stats-api/public" -name '*.php' -type f)
echo "PHP syntax OK"

echo "==> web (vite build)"
(cd "$root/web" && { [ -d node_modules ] || npm ci; } && npm run build)

echo "Static analysis passed."
