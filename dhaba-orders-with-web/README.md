# Dhaba Orders

Customers order from the menu, the kitchen screen shows each order moving from *new* to *cooking* to *ready*, and a live stats panel shows orders per minute and the top dishes.

| Service | Stack | Folder | Status |
|---|---|---|---|
| `web` | React (Vite) on Nginx | `services/web` | code + tests |
| `menu-api` | Python (FastAPI) | – | already done |
| `orders-api` | Go | `services/orders-api` | code + tests |
| `kitchen-worker` | Java (Spring Boot) | `services/kitchen-worker` | code + tests |
| `stats-api` | PHP (Laravel) | `services/stats-api` | code + tests |
| `postgres`, `redis` | – | `db/init.sql` for the schema | already done |

How the services talk to each other (env vars, Redis keys, HTTP endpoints): [`docs/CONTRACT.md`](docs/CONTRACT.md).

## Run the tests

```bash
./scripts/test-all.sh          # all three
cd services/orders-api     && go test ./...
cd services/kitchen-worker && mvn test
cd services/stats-api      && composer install && vendor/bin/phpunit
cd services/web            && npm install && npm test
```

Requirements: Go 1.24+, JDK 21 + Maven 3.9+, PHP 8.2+ + Composer, Node 22+.

The tests need no database or Redis: each service talks to them through small interfaces that the tests replace with fakes.

## First-time setup

```bash
cd services/orders-api && go mod tidy        # writes go.sum (adds the Postgres driver)
```

## Run a service locally

Needs Postgres (with `db/init.sql` applied) and Redis reachable on the `DB_*` / `REDIS_*` variables from the contract.

```bash
cd services/orders-api     && go run .                      # :8080
cd services/kitchen-worker && mvn spring-boot:run           # no port
cd services/stats-api      && php artisan serve --port=8001 # :8001
cd services/web            && npm run dev                   # :5173, proxies /api/* to the services above
```

Try it:

```bash
curl -X POST localhost:8080/orders -H 'Content-Type: application/json' \
  -d '{"customer":"Ali","items":[{"name":"Chicken Karahi","qty":2,"price":12.5}]}'
curl localhost:8080/info
curl localhost:8001/api/stats
```

## Known limitations

- The queue is a plain Redis list. If kitchen-worker crashes after taking a job, that order stays in `cooking`. Fine for now; a reliable queue (e.g. `BLMOVE` to a processing list) would fix it.
- `stats:dishes` is an all-time total and never expires.
- The Docker images, Kubernetes manifests and CI are not included.
