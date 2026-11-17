# Shared contract between services

If you change one of these, change it in every service that uses it. Local values live in the repo-root `.env` (see `.env.example`).

## Environment variables

| Variable | Local value | Used by |
|---|---|---|
| `POSTGRES_HOST` | `localhost` | Docker Compose, menu-api |
| `POSTGRES_PORT` | `5432` | Docker Compose, menu-api |
| `POSTGRES_USER` | `dhaba` | Docker Compose, menu-api |
| `POSTGRES_PASSWORD` | `dhaba_local_dev` | Docker Compose, menu-api |
| `POSTGRES_DB` | `dhaba` | Docker Compose, menu-api |
| `DB_HOST` | `localhost` | orders-api, kitchen-worker |
| `DB_PORT` | `5432` | orders-api, kitchen-worker |
| `DB_NAME` | `dhaba` | orders-api, kitchen-worker |
| `DB_USER` | `dhaba` | orders-api, kitchen-worker |
| `DB_PASSWORD` | `dhaba_local_dev` | orders-api, kitchen-worker |
| `DB_SSLMODE` | `disable` | orders-api |
| `REDIS_HOST` | `localhost` | menu-api, orders-api, kitchen-worker, stats-api |
| `REDIS_PORT` | `6379` | menu-api, orders-api, kitchen-worker, stats-api |
| `REDIS_PASSWORD` | *(empty)* | orders-api, kitchen-worker, stats-api |
| `APP_VERSION` | `dev` | menu-api, orders-api (`/info`) |
| `PORT` | `8080` | orders-api |
| `COOKING_SECONDS` | `8` | kitchen-worker |
| `MENU_API_URL` | `http://localhost:8000` | web (Vite and Nginx) |
| `ORDERS_API_URL` | `http://localhost:8080` | web |
| `STATS_API_URL` | `http://localhost:8001` | web |

`POSTGRES_*` and `DB_*` point at the same database. The Postgres image only accepts `POSTGRES_*`. The Go and Java services only accept `DB_*`.

Pod name for `/info` comes from `HOSTNAME`, which Kubernetes sets to the pod name.

Local ports on one machine: menu-api `8000`, orders-api `8080`, stats-api `8001`, web `5173`. Inside a container network, menu-api and stats-api can each listen on `8000` because they have different hostnames.

## Postgres

See `db/init.sql`.

`menu_items` — `id`, `name`, `description`, `category`, `price_pkr`, `available`. Seeded on first start.

`orders` — `id`, `customer`, `items` (JSON array of `{name, qty, price}`), `status`, `created_at`, `updated_at`. Status moves `new` → `cooking` → `ready`.

## Redis keys

| Key | Type | Written by | Read by | Meaning |
|---|---|---|---|---|
| `menu:all` | string | menu-api | menu-api | Cached menu JSON, 60s TTL |
| `orders:queue` | list | orders-api (`LPUSH`) | kitchen-worker (`RPOP`) | Job JSON: `{"order_id": 42}` |
| `stats:orders:<minute>` | string (counter) | orders-api (`INCR`, 1h TTL) | stats-api | Orders created in that minute. `<minute>` = unix time / 60, integer division |
| `stats:dishes` | sorted set | orders-api (`ZINCRBY`) | stats-api | Score = total quantity ordered, member = dish name |

## HTTP APIs

### menu-api (port 8000 locally)

| Method | Path | Notes |
|---|---|---|
| GET | `/livez` | Liveness |
| GET | `/health` | 200 if Postgres and Redis answer, else 503 |
| GET | `/api/menu/info` | `{"service","version","pod"}` |
| GET | `/api/menu` | JSON array of menu items (`price_pkr`) |
| GET | `/api/menu/{id}` | One item, 404 if missing |

### orders-api (port 8080)

| Method | Path | Notes |
|---|---|---|
| GET | `/health` | Liveness, always 200 |
| GET | `/ready` | 200 if Postgres and Redis answer, else 503 |
| GET | `/info` | `{"service","version","pod"}` |
| POST | `/orders` | Body `{"customer":"Ali","items":[{"name":"Dal","qty":2,"price":5.5}]}` → 201 with the order |
| GET | `/orders` | Latest 50 orders, newest first. `?limit=N` (max 200) |
| GET | `/orders/{id}` | One order, 404 if missing |

### stats-api (port 8001 locally)

| Method | Path | Notes |
|---|---|---|
| GET | `/api/health` | Liveness |
| GET | `/api/stats` | `{"orders_per_minute", "history":[{"minute","orders"}], "top_dishes":[{"name","count"}]}` |

`orders_per_minute` is the last *complete* minute. `history` is the last 10 complete minutes, oldest first.

### kitchen-worker

No port. Pops jobs, sets the order to `cooking`, waits `COOKING_SECONDS`, sets it to `ready`.

### web (port 5173 in dev, port 80 in Nginx)

The browser only talks to `web`. Vite (dev) and Nginx (container) forward `/api/*`:

| Browser path | Goes to |
|---|---|
| `/api/menu`, `/api/menu/{id}` | menu-api `/api/menu` |
| `/api/orders`, `/api/orders/{id}` | orders-api `/orders` |
| `/api/info` | orders-api `/info` |
| `/api/stats` | stats-api `/api/stats` |
