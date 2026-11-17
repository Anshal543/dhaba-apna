# Dhaba Orders

Local development. Requires Docker Desktop running.

    cp .env.example .env
    docker compose up -d
    docker compose ps          # both services should say "healthy"

Reset the databases completely (deletes all data):

    docker compose down -v
