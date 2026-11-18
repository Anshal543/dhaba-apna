# Implementation summary

This document records how Dhaba Orders was built and delivered, from individual services to a CI/CD pipeline that deploys to kind on an EC2 VM.

## 1. Services first, each tested on its own

Five services were created and verified independently before any container or deployment work:

| Service | Stack | Static analysis | Tests |
| --- | --- | --- | --- |
| `menu-api` | Python / FastAPI | `ruff check .` | `pytest -q` |
| `orders-api` | Go | `go vet ./...` | `go test ./...` |
| `kitchen-worker` | Java / Spring Boot | `mvn -DskipTests compile` | `mvn test` |
| `stats-api` | PHP / Laravel | `php -l` | `vendor/bin/phpunit` |
| `web` | React / Vite | `npm run build` | `node --test` |

Two helper scripts were added:

- `scripts/lint-all.sh` — runs all static analysis.
- `scripts/test-all.sh` — runs all unit tests.

Both are used by GitHub Actions, so a merge cannot proceed if any service fails analysis or tests.

## 2. Dockerfiles and local Docker testing

A Dockerfile and `.dockerignore` were added for every service:

- `menu-api/Dockerfile` — Python 3.12 slim, Uvicorn on port 8000.
- `orders-api/Dockerfile` — multi-stage Go 1.24 build, static binary on Alpine, port 8080.
- `kitchen-worker/Dockerfile` — multi-stage Maven/JDK 21 build, JRE runtime.
- `stats-api/Dockerfile` — PHP 8.4 CLI (required by `composer.lock`), Laravel artisan server on port 8001.
- `web/Dockerfile` — Vite build served by Nginx on port 80.

`docker-compose.yml` was extended with an `apps` profile so Postgres and Redis remain the default services:

```bash
docker compose --profile apps up --build -d
```

Verified locally: all containers healthy, health endpoints returned 200, and an order moved `new` → `cooking` → `ready`.

Local fixes made during this phase:

- Removed inline `#` comments from README shell commands; macOS zsh does not enable them by default and Maven received `#` as a lifecycle phase.
- Fixed `web/nginx/default.conf.template` where `proxy_pass ${MENU_API_URL}$request_uri;` caused `no resolver defined to resolve menu-api` and HTTP 502. It is now a literal `proxy_pass ${MENU_API_URL};`.
- Corrected `STATS_API_URL` to use port 8001.

## 3. Kubernetes manifests for kind

Manifests live under `k8s/`:

- `k8s/base/` — ConfigMap, Postgres, Redis, and the five application Deployments/Services.
- `k8s/overlays/staging/` — namespace `dhaba-staging`, staging images.
- `k8s/overlays/production/` — namespace `dhaba-production`, production config patch and Ingress.
- `k8s/traefik/traefik.yaml` — Traefik deployment, RBAC, NodePort Service and IngressClass.

Details:

- Postgres and Redis run inside each namespace on PersistentVolumeClaims, with `init-sql` mounted for the schema and seed data.
- Application images use `imagePullPolicy: IfNotPresent` because they are loaded into kind explicitly.
- Deployments use `strategy: Recreate`, one replica, and `terminationGracePeriodSeconds: 0` to avoid stuck rolling updates on a single-node kind cluster.
- `orders-api` probes have `initialDelaySeconds: 35` because the app waits up to 30 seconds for Postgres at startup.

`kind-config.yaml` maps host ports into the kind node:

- Host `80` → Traefik web entrypoint (NodePort `30080`).
- Host `443` → Traefik websecure entrypoint (NodePort `30443`).

Traefik routes by hostname:

- `staging.anshal.space` → `web` service in `dhaba-staging`.
- `anshal.space` → `web` service in `dhaba-production`.

Let's Encrypt certificates are issued automatically with the HTTP-01 challenge using `LETSENCRYPT_EMAIL` (defaults to `admin@anshal.space`).

`scripts/setup-kind-host.sh` installs Docker, kind and kubectl on a fresh Linux host and creates the cluster.

## 4. CI/CD pipeline

Workflows:

- `.github/workflows/ci-cd.yml` — triggers on push and PR to `main`; runs static analysis and unit tests in parallel; on push, builds and pushes five images to ECR; then calls the deploy workflow for staging and production.
- `.github/workflows/deploy.yml` — reusable deploy workflow. Production waits for an approval on the `production` GitHub environment.

The pipeline does the following:

1. Analyze and test every service.
2. Log in to ECR and build/push images tagged with the commit SHA.
3. Run the deploy job on a self-hosted GitHub runner installed on the EC2 VM.
4. Verify the kind cluster is reachable and recreate it if necessary.
5. Log in to ECR with `aws-actions/amazon-ecr-login@v2` (no AWS CLI needed on the runner).
6. Prune old Docker and kind images to control disk usage.
7. Pull each image, load it into kind with `kind load docker-image`, and apply the overlay with `kubectl apply -k`.
8. Create the namespace, `init-sql` ConfigMap and `dhaba-secrets` Secret.
9. Wait for rollouts and print full pod diagnostics on failure.

ECR repositories are created by:

```bash
./scripts/create-ecr-repos.sh
```

Authentication uses an IAM user access key stored in GitHub Secrets; OIDC is intentionally not used.

GitHub repository configuration:

- Variables: `AWS_REGION`, `AWS_ACCOUNT_ID`, `KIND_CLUSTER_NAME`.
- Secrets: `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `POSTGRES_PASSWORD`, `APP_KEY`. `REDIS_PASSWORD` is optional.
- `POSTGRES_PASSWORD` is reused as `DB_PASSWORD` for the services, so only one database password is required.
- Environments: `staging` and `production`, with required reviewers on `production`.

Deploy jobs are skipped until `KIND_CLUSTER_NAME` is set.

## 5. EC2 VM and kind instead of EKS

EKS was the original target but was replaced with a simpler setup:

- One x86_64 EC2 VM (Ubuntu 24.04 or Amazon Linux 2023, at least `t3.large`, root volume at least 30Gi).
- Docker, kind and kubectl installed by `scripts/setup-kind-host.sh`.
- A self-hosted GitHub Actions runner registered with the `kind` label.
- Inbound ports open for SSH (22), staging web (80) and production web (8080).

After a successful deploy:

- Staging: `https://staging.anshal.space/healthz`
- Production: `https://anshal.space/healthz`

DNS `A` records for both hostnames point to the EC2 public IP, and the security group allows inbound 80 and 443.

## 6. Problems found and fixed along the way

- Maven error `Unknown lifecycle phase "#"` — caused by inline shell comments in README commands.
- Web HTTP 502 — nginx `no resolver defined to resolve menu-api`; fixed by removing `$request_uri` from `proxy_pass`.
- PHP dependency failure — `composer.lock` requires PHP 8.4.1+, so the Docker image and CI use PHP 8.4.
- Unit tests failing silently in CI — `[ -d .venv ] && ...` exited under `set -e` when no local venv existed; replaced with explicit `if` blocks.
- `Missing variable: AWS_ACCOUNT_ID` — the variable was used to build `ECR_REGISTRY` but not exported to the deploy step; added to the job `env`.
- `aws: command not found` on the self-hosted runner — replaced AWS CLI login with `aws-actions/amazon-ecr-login@v2`.
- Stuck rollouts / `old replicas pending termination` — reduced replicas, forced termination, added cleanup of stuck pods and image pruning.
- `connection refused` to `127.0.0.1:<port>` — stale kubeconfig after the kind node was deleted; the workflow now recreates the cluster when unreachable.
- `password authentication failed for user "dhaba"` — `DB_PASSWORD` and `POSTGRES_PASSWORD` were configured as different secrets; the apps now derive `DB_PASSWORD` from `POSTGRES_PASSWORD`.
- `CrashLoopBackOff` from liveness probes firing before the app finished its Postgres wait — added startup delay to `orders-api` probes.

## 7. Team workflow

- Jira is used to track tickets and progress for each piece of work.
- Slack is used for GitHub notifications and deployment messages. Configure the GitHub → Slack app/webhook in repository or organization settings; no Slack credentials are stored in this codebase.
