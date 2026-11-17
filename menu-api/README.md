# menu-api (Python / FastAPI)

Serves the menu from Postgres, cached in Redis for 60 seconds.

| Endpoint | Purpose |
|---|---|
| `GET /api/menu` | All dishes. Header `X-Cache: HIT` or `MISS` |
| `GET /api/menu/{id}` | One dish |
| `GET /api/menu/info` | Service name, version, pod name |
| `GET /health` | Readiness: checks Postgres and Redis (503 if either is down) |
| `GET /livez` | Liveness: process is up, checks nothing else |

Run locally (from this folder, with `docker compose up -d` running in the parent):

    python3.12 -m venv .venv && source .venv/bin/activate
    pip install -r requirements-dev.txt
    pytest
    ruff check .
    uvicorn app.main:app --reload --port 8000
