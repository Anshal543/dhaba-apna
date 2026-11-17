import logging
import os
from collections.abc import AsyncIterator
from contextlib import asynccontextmanager
from typing import Annotated, Any

import psycopg
import redis
from fastapi import Depends, FastAPI, HTTPException, Request, Response
from psycopg_pool import ConnectionPool
from pydantic import BaseModel

from app.config import get_settings
from app.repository import MenuRepository

logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(name)s %(message)s")
log = logging.getLogger("menu-api")


class MenuItem(BaseModel):
    id: int
    name: str
    description: str
    category: str
    price_pkr: int
    available: bool


@asynccontextmanager
async def lifespan(app: FastAPI) -> AsyncIterator[None]:
    settings = get_settings()
    # pool.open() does not wait for a connection: the app starts even if Postgres is
    # not ready yet, and /health reports the problem instead of the process crashing.
    pool = ConnectionPool(settings.postgres_dsn, min_size=1, max_size=5, open=False)
    pool.open()
    cache = redis.Redis(
        host=settings.redis_host,
        port=settings.redis_port,
        decode_responses=True,
        socket_connect_timeout=2,
        socket_timeout=2,
    )
    app.state.repo = MenuRepository(pool, cache, settings.menu_cache_ttl_seconds)
    log.info("menu-api %s started", settings.app_version)
    yield
    pool.close()
    cache.close()


app = FastAPI(title="menu-api", lifespan=lifespan)


def get_repository(request: Request) -> MenuRepository:
    return request.app.state.repo


Repo = Annotated[MenuRepository, Depends(get_repository)]


@app.get("/livez")
def livez() -> dict[str, str]:
    """Liveness: the process is up. Deliberately checks no dependencies."""
    return {"status": "alive"}


@app.get("/health")
def health(repo: Repo, response: Response) -> dict[str, Any]:
    """Readiness: can we reach Postgres and Redis right now?"""
    checks = repo.check()
    healthy = all(checks.values())
    if not healthy:
        response.status_code = 503
    return {
        "status": "ok" if healthy else "degraded",
        "checks": checks,
        "version": get_settings().app_version,
    }


@app.get("/api/menu/info")
def info() -> dict[str, str | None]:
    return {
        "service": "menu-api",
        "version": get_settings().app_version,
        "pod": os.getenv("HOSTNAME"),
    }


@app.get("/api/menu", response_model=list[MenuItem])
def list_menu(repo: Repo, response: Response) -> list[dict[str, Any]]:
    return _load_menu(repo, response)


@app.get("/api/menu/{item_id}", response_model=MenuItem)
def get_item(item_id: int, repo: Repo, response: Response) -> dict[str, Any]:
    items = _load_menu(repo, response)
    for item in items:
        if item["id"] == item_id:
            return item
    raise HTTPException(status_code=404, detail="menu item not found")


def _load_menu(repo: MenuRepository, response: Response) -> list[dict[str, Any]]:
    try:
        items, from_cache = repo.get_menu()
    except psycopg.Error as exc:
        log.error("menu unavailable: %s", exc)
        raise HTTPException(status_code=503, detail="menu temporarily unavailable") from exc
    response.headers["X-Cache"] = "HIT" if from_cache else "MISS"
    return items
