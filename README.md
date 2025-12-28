# xVault

Automated Backup SaaS (Hub control plane + Worker data plane).

Docs:
- docs/architecture.md
- docs/plan.md
- docs/data-model.md
- docs/dev-start.md

Local dev (Docker Compose):

- Copy `deploy/.env.example` to `deploy/.env`
- Run: `docker compose --env-file deploy/.env -f deploy/docker-compose.yml up --build`

