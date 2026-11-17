"""Unit tests: no Postgres or Redis needed."""

import psycopg
import redis
from fastapi.testclient import TestClient

from app.main import app, get_repository
from app.repository import CACHE_KEY, MenuRepository

ITEMS = [
    {
        "id": 1,
        "name": "Chicken Karahi",
        "description": "Wok-cooked chicken",
        "category": "Mains",
        "price_pkr": 1800,
        "available": True,
    },
    {
        "id": 2,
        "name": "Butter Naan",
        "description": "Tandoor bread",
        "category": "Bread",
        "price_pkr": 80,
        "available": True,
    },
]


class FakeRepo:
    """Stands in for MenuRepository in the HTTP tests."""

    def __init__(self, items=None, from_cache=False, db_down=False, checks=None):
        self.items = ITEMS if items is None else items
        self.from_cache = from_cache
        self.db_down = db_down
        self.checks = checks or {"postgres": True, "redis": True}

    def get_menu(self):
        if self.db_down:
            raise psycopg.OperationalError("connection refused")
        return self.items, self.from_cache

    def check(self):
        return self.checks


def client_with(repo: FakeRepo) -> TestClient:
    app.dependency_overrides[get_repository] = lambda: repo
    # No "with" block: we do not want the real startup (it connects to Postgres).
    return TestClient(app)


def teardown_function():
    app.dependency_overrides.clear()


def test_menu_lists_items_and_reports_cache_miss():
    resp = client_with(FakeRepo()).get("/api/menu")
    assert resp.status_code == 200
    assert [i["name"] for i in resp.json()] == ["Chicken Karahi", "Butter Naan"]
    assert resp.headers["X-Cache"] == "MISS"


def test_menu_reports_cache_hit():
    resp = client_with(FakeRepo(from_cache=True)).get("/api/menu")
    assert resp.headers["X-Cache"] == "HIT"


def test_menu_item_by_id():
    resp = client_with(FakeRepo()).get("/api/menu/2")
    assert resp.status_code == 200
    assert resp.json()["name"] == "Butter Naan"


def test_unknown_menu_item_is_404():
    assert client_with(FakeRepo()).get("/api/menu/999").status_code == 404


def test_menu_is_503_when_database_is_down_and_cache_is_cold():
    resp = client_with(FakeRepo(db_down=True)).get("/api/menu")
    assert resp.status_code == 503


def test_info_shows_service_and_version():
    body = client_with(FakeRepo()).get("/api/menu/info").json()
    assert body["service"] == "menu-api"
    assert "version" in body


def test_livez_never_checks_dependencies():
    resp = client_with(FakeRepo(checks={"postgres": False, "redis": False})).get("/livez")
    assert resp.status_code == 200


def test_health_ok():
    resp = client_with(FakeRepo()).get("/health")
    assert resp.status_code == 200
    assert resp.json()["status"] == "ok"


def test_health_is_503_when_a_dependency_is_down():
    repo = FakeRepo(checks={"postgres": True, "redis": False})
    resp = client_with(repo).get("/health")
    assert resp.status_code == 503
    assert resp.json()["checks"]["redis"] is False


# --- caching logic of the real repository, with a fake Redis ---------------


class FakeCache:
    def __init__(self, broken=False):
        self.store = {}
        self.broken = broken

    def get(self, key):
        if self.broken:
            raise redis.ConnectionError("down")
        return self.store.get(key)

    def set(self, key, value, ex=None):
        if self.broken:
            raise redis.ConnectionError("down")
        self.store[key] = value


def make_repo(cache, db_calls):
    repo = MenuRepository(pool=None, cache=cache, ttl_seconds=60)

    def fake_read_db():
        db_calls.append(1)
        return ITEMS

    repo._read_db = fake_read_db
    return repo


def test_second_read_comes_from_cache():
    cache, calls = FakeCache(), []
    repo = make_repo(cache, calls)
    _, first_from_cache = repo.get_menu()
    items, second_from_cache = repo.get_menu()
    assert (first_from_cache, second_from_cache) == (False, True)
    assert items == ITEMS
    assert len(calls) == 1
    assert CACHE_KEY in cache.store


def test_menu_still_works_when_redis_is_down():
    calls = []
    repo = make_repo(FakeCache(broken=True), calls)
    items, from_cache = repo.get_menu()
    assert items == ITEMS
    assert from_cache is False
