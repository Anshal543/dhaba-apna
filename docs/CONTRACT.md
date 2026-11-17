# Shared contract between services

Everything the three new services agree on. If you change one of these, change it in every service.

## Environment variables (same names in all services)

| Variable | Default | Used by |
|---|---|---|
| `DB_HOST` | `localhost` | orders-api, kitchen-worker |
| `DB_PORT` | `5432` | orders-api, kitchen-worker |
| `DB_NAME` | `dhaba` | orders-api, kitchen-worker |
| `DB_USER` | `dhaba` | orders-api, kitchen-worker |
| `DB_PASSWORD` | `dhaba` | orders-api, kitchen-worker |
| `REDIS_HOST` | `localhost` | all three |
| `REDIS_PORT` | `6379` | all three |
| `REDIS_PASSWORD` | *(empty)* | all three |
| `APP_VERSION` | `dev` | orders-api (`/info`) |
| `PORT` | `8080` | orders-api |
| `COOKING_SECONDS` | `8` | kitchen-worker |

Pod name for `/info` comes from `HOSTNAME`, which Kubernetes sets to the pod name.

## Postgres

Table `orders` — see `db/init.sql`. Status moves `new` → `cooking` → `ready`.

## Redis keys

| Key | Type | Written by | Read by | Meaning |
|---|---|---|---|---|
| `orders:queue` | list | orders-api (`LPUSH`) | kitchen-worker (`RPOP`) | Job JSON: `{"order_id": 42}` |
| `stats:orders:<minute>` | string (counter) | orders-api (`INCR`, 1h TTL) | stats-api | Orders created in that minute. `<minute>` = unix time / 60, integer division |
| `stats:dishes` | sorted set | orders-api (`ZINCRBY`) | stats-api | Score = total quantity ordered, member = dish name |

## HTTP APIs

### orders-api (port 8080)

| Method | Path | Notes |
|---|---|---|
| GET | `/health` | Liveness, always 200 |
| GET | `/ready` | 200 if Postgres and Redis answer, else 503 |
| GET | `/info` | `{"service","version","pod"}` |
| POST | `/orders` | Body `{"customer":"Ali","items":[{"name":"Dal","qty":2,"price":5.5}]}` → 201 with the order |
| GET | `/orders` | Latest 50 orders, newest first. `?limit=N` (max 200) |
| GET | `/orders/{id}` | One order, 404 if missing |

### stats-api (port 8000)

| Method | Path | Notes |
|---|---|---|
| GET | `/api/health` | Liveness |
| GET | `/api/stats` | `{"orders_per_minute", "history":[{"minute","orders"}], "top_dishes":[{"name","count"}]}` |

`orders_per_minute` is the last *complete* minute. `history` is the last 10 complete minutes, oldest first.

### kitchen-worker

No port. Pops jobs, sets the order to `cooking`, waits `COOKING_SECONDS`, sets it to `ready`.
