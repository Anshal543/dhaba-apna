# Dhaba Orders

Customers order from the menu, the kitchen screen moves each order from *new* to *cooking* to *ready*, and a stats panel shows orders per minute and the top dishes.

| Service | Stack | Folder | Port |
|---|---|---|---|
| `web` | React (Vite) on Nginx | `web` | 5173 |
| `menu-api` | Python (FastAPI) | `menu-api` | 8000 |
| `orders-api` | Go | `orders-api` | 8080 |
| `kitchen-worker` | Java (Spring Boot) | `kitchen-worker` | none |
| `stats-api` | PHP (Laravel) | `stats-api` | 8001 |
| `postgres`, `redis` | – | `db/init.sql` | 5432, 6379 |

How the services talk to each other: [`docs/CONTRACT.md`](docs/CONTRACT.md).

## First-time setup

```bash
cp .env.example .env          # already done if .env exists
docker compose up -d          # Postgres (schema + seed) and Redis
docker compose ps             # both services should say "healthy"
```

`db/init.sql` runs only when the Postgres volume is created. To apply a schema change, reset the data:

```bash
docker compose down -v
docker compose up -d
```

There is one env file at the repo root. menu-api and stats-api read it themselves. The Go and Java services read the process environment, so export it before starting them:

```bash
set -a && source .env && set +a
```

## Run a service locally

```bash
cd menu-api        && source .venv/bin/activate && uvicorn app.main:app --reload --port 8000
cd orders-api      && go run .                         # :8080
cd kitchen-worker  && mvn spring-boot:run              # no port
cd stats-api       && php artisan serve --port=8001    # :8001
cd web             && npm run dev                      # :5173
```

## Tests

```bash
./scripts/test-all.sh
```

Requirements: Go 1.24+, JDK 21 + Maven 3.9+, PHP 8.2+ + Composer, Python 3.12, Node 22+.

The tests need no database or Redis: each service talks to them through small interfaces that the tests replace with fakes.
