# Dhaba Orders

Customers order from the menu, the kitchen screen moves each order from *new* to *cooking* to *ready*, and a stats panel shows orders per minute and the top dishes.

| Service | Stack | Folder | Port |
|---|---|---|---|
| `web` | React (Vite) on Nginx | `web` | 5173 |
| `menu-api` | Python (FastAPI) | `menu-api` | 8000 |
| `orders-api` | Go | `orders-api` | 8080 |
| `kitchen-worker` | Java (Spring Boot) | `kitchen-worker` | none |
| `stats-api` | PHP (Laravel) | `stats-api` | 8001 |
| `postgres`, `redis` | – | `db/init.sql` | 5432, 6379 |

How the services talk to each other: [`docs/CONTRACT.md`](docs/CONTRACT.md).

## First-time setup

```bash
cp .env.example .env
docker compose up -d
docker compose ps
```

`cp` is already done if `.env` exists. `docker compose up -d` starts Postgres (schema + seed) and Redis. Both services should show `healthy` in `docker compose ps`.

`db/init.sql` runs only when the Postgres volume is created. To apply a schema change, reset the data:

```bash
docker compose down -v
docker compose up -d
```

There is one env file at the repo root. menu-api and stats-api read it themselves. The Go and Java services read the process environment, so export it before starting them:

```bash
set -a && source .env && set +a
```

## Run a service locally

Run each command from the repo root in its own terminal. Ports are listed in the table above; `kitchen-worker` is a worker and listens on no port.

```bash
cd menu-api        && source .venv/bin/activate && uvicorn app.main:app --reload --port 8000
cd orders-api      && go run .
cd kitchen-worker  && mvn spring-boot:run
cd stats-api       && php artisan serve --port=8001
cd web             && npm run dev
```

## Run everything in Docker

Build and start all five services together with Postgres and Redis:

```bash
docker compose --profile apps up --build -d
```

Then open http://localhost:5173. Health endpoints are at `:8000/livez`, `:8080/health`, and `:8001/api/health`.

To watch logs or stop the stack:

```bash
docker compose --profile apps logs -f
docker compose --profile apps down
```

Add `-v` to `down` to also delete the Postgres and Redis data. Without `--profile apps`, only Postgres and Redis start, which is the setup used by "Run a service locally" above.

## Tests

```bash
./scripts/test-all.sh
```

Static analysis runs separately:

```bash
./scripts/lint-all.sh
```

## CI/CD to AWS

`.github/workflows/ci-cd.yml` runs on every push to `main` and on pull requests. Static analysis and unit tests run in parallel; on a push to `main`, both must pass before images are built and pushed to ECR with the commit SHA as the tag. Staging is deployed automatically from `k8s/overlays/staging`. Production uses `k8s/overlays/production` and waits for an approval on the `production` GitHub environment.

One-time AWS setup, with AWS CLI credentials configured:

```bash
./scripts/create-ecr-repos.sh
```

The pipeline authenticates with an IAM user access key stored in GitHub Secrets, so no OIDC provider is needed. Create an IAM user with programmatic access and attach this inline policy:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ecr:GetAuthorizationToken", "ecr:BatchCheckLayerAvailability",
        "ecr:GetDownloadUrlForLayer", "ecr:BatchGetImage", "ecr:PutImage",
        "ecr:InitiateLayerUpload", "ecr:UploadLayerPart", "ecr:CompleteLayerUpload",
        "ecr:DescribeRepositories", "ecr:CreateRepository"
      ],
      "Resource": "*"
    },
    {
      "Effect": "Allow",
      "Action": ["eks:DescribeCluster", "eks:ListClusters"],
      "Resource": "*"
    }
  ]
}
```

Add the repo configuration with the GitHub CLI (run from the repo root):

```bash
gh variable set AWS_REGION --body us-east-1
gh variable set AWS_ACCOUNT_ID --body 123456789012
gh secret set AWS_ACCESS_KEY_ID
gh secret set AWS_SECRET_ACCESS_KEY
gh secret set POSTGRES_PASSWORD
gh secret set DB_PASSWORD
gh secret set APP_KEY --body "$(php -r 'echo "base64:".base64_encode(random_bytes(32));')"
```

`AWS_REGION` and `AWS_ACCOUNT_ID` are variables; everything else is a secret. `REDIS_PASSWORD` is optional. Create the environments with `gh api -X POST repos/:owner/:repo/environments -f name=staging` and the same for `production`, then add required reviewers to `production` in Settings → Environments; reviewers cannot be set by the CLI easily.

EKS is not covered by the AWS Free Tier. Until the clusters exist, leave `EKS_CLUSTER_STAGING` and `EKS_CLUSTER_PRODUCTION` unset and the deploy jobs are skipped automatically. When a cluster is ready, set the variable and add the IAM user to the cluster access entries so `kubectl` works:

```bash
eksctl create cluster --name dhaba-staging --region us-east-1 --node-type t3.small --nodes 1
eksctl create iamidentitymapping --cluster dhaba-staging \
  --arn arn:aws:iam::123456789012:user/github-actions \
  --group system:masters --username github-actions
gh variable set EKS_CLUSTER_STAGING --body dhaba-staging
```

Repeat for production. The node instance role needs ECR pull access. Long-lived access keys are less secure than OIDC, so rotate them and never commit them.

The manifests run Postgres and Redis inside each namespace on persistent volumes. Use RDS and ElastiCache instead for real production workloads.

Requirements: Go 1.24+, JDK 21 + Maven 3.9+, PHP 8.4.1+ + Composer, Python 3.12, Node 22+.

The tests need no database or Redis: each service talks to them through small interfaces that the tests replace with fakes.
