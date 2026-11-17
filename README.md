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
cp .env.example .env
docker compose up -d
docker compose ps
```

`cp` is already done if `.env` exists. `docker compose up -d` starts Postgres (schema + seed) and Redis. Both services should show `healthy` in `docker compose ps`.

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

Run each command from the repo root in its own terminal. Ports are listed in the table above; `kitchen-worker` is a worker and listens on no port.

```bash
cd menu-api        && source .venv/bin/activate && uvicorn app.main:app --reload --port 8000
cd orders-api      && go run .
cd kitchen-worker  && mvn spring-boot:run
cd stats-api       && php artisan serve --port=8001
cd web             && npm run dev
```

## Run everything in Docker

Build and start all five services together with Postgres and Redis:

```bash
docker compose --profile apps up --build -d
```

Then open http://localhost:5173. Health endpoints are at `:8000/livez`, `:8080/health`, and `:8001/api/health`.

To watch logs or stop the stack:

```bash
docker compose --profile apps logs -f
docker compose --profile apps down
```

Add `-v` to `down` to also delete the Postgres and Redis data. Without `--profile apps`, only Postgres and Redis start, which is the setup used by "Run a service locally" above.

## Tests

```bash
./scripts/test-all.sh
```

Requirements: Go 1.24+, JDK 21 + Maven 3.9+, PHP 8.2+ + Composer, Python 3.12, Node 22+.

The tests need no database or Redis: each service talks to them through small interfaces that the tests replace with fakes.
