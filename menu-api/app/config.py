from functools import lru_cache

from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    """All settings come from environment variables.

    For local development they are also read from ../.env (the file you created
    from .env.example). In Kubernetes they come from a ConfigMap and a Secret.
    """

    model_config = SettingsConfigDict(env_file=("../.env", ".env"), extra="ignore")

    postgres_host: str = "localhost"
    postgres_port: int = 5432
    postgres_user: str = "dhaba"
    postgres_password: str
    postgres_db: str = "dhaba"

    redis_host: str = "localhost"
    redis_port: int = 6379

    menu_cache_ttl_seconds: int = 60

    # Set at image build time (the Git commit). "dev" when running locally.
    app_version: str = "dev"

    @property
    def postgres_dsn(self) -> str:
        return (
            f"host={self.postgres_host} port={self.postgres_port} "
            f"user={self.postgres_user} password={self.postgres_password} "
            f"dbname={self.postgres_db} connect_timeout=3"
        )


@lru_cache
def get_settings() -> Settings:
    return Settings()
