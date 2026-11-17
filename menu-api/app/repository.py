import json
import logging
from typing import Any

import redis
from psycopg.rows import dict_row
from psycopg_pool import ConnectionPool

log = logging.getLogger("menu-api")

CACHE_KEY = "menu:all"

MENU_QUERY = """
    SELECT id, name, description, category, price_pkr, available
    FROM menu_items
    ORDER BY id
"""


class MenuRepository:
    """Reads the menu from Postgres and caches the whole list in Redis.

    Redis is an optimisation, not a dependency: if it is down, we read from
    Postgres. If Postgres is down but the cache is warm, we still answer.
    """

    def __init__(self, pool: ConnectionPool, cache: redis.Redis, ttl_seconds: int) -> None:
        self._pool = pool
        self._cache = cache
        self._ttl = ttl_seconds

    def get_menu(self) -> tuple[list[dict[str, Any]], bool]:
        """Return (items, from_cache)."""
        cached = self._read_cache()
        if cached is not None:
            return cached, True
        items = self._read_db()
        self._write_cache(items)
        return items, False

    def check(self) -> dict[str, bool]:
        """Used by /health: can we reach each dependency right now?"""
        result = {"postgres": True, "redis": True}
        try:
            with self._pool.connection(timeout=2) as conn:
                conn.execute("SELECT 1")
        except Exception:  # any failure means "not healthy"
            result["postgres"] = False
        try:
            self._cache.ping()
        except redis.RedisError:
            result["redis"] = False
        return result

    def _read_cache(self) -> list[dict[str, Any]] | None:
        try:
            raw = self._cache.get(CACHE_KEY)
        except redis.RedisError:
            log.warning("redis unavailable, reading menu from postgres")
            return None
        return json.loads(raw) if raw else None

    def _write_cache(self, items: list[dict[str, Any]]) -> None:
        try:
            self._cache.set(CACHE_KEY, json.dumps(items), ex=self._ttl)
        except redis.RedisError:
            log.warning("redis unavailable, menu not cached")

    def _read_db(self) -> list[dict[str, Any]]:
        with self._pool.connection(timeout=3) as conn:
            with conn.cursor(row_factory=dict_row) as cur:
                cur.execute(MENU_QUERY)
                return list(cur.fetchall())

