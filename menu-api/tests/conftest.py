import os

# Settings requires a Postgres password. Unit tests never connect to Postgres,
# but the settings object must still be valid (also on CI, where there is no .env).
os.environ.setdefault("POSTGRES_PASSWORD", "test-only")
