#!/usr/bin/env bash
# Creates one ECR repository per service under the dhaba-tactech/ prefix.
# Each repo: scan on push, lifecycle rule keeping only the newest 20 images.
# Safe to re-run: existing repositories are left alone (settings are re-applied
# idempotently) and the script exits 0.
#
# Region/profile come from your AWS CLI configuration (AWS_REGION, AWS_PROFILE, etc.).

set -euo pipefail

PREFIX="dhaba-tactech"

SERVICES=(
  web
  menu-api
  orders-api
  kitchen-worker
  stats-api
)

KEEP_LAST=20

LIFECYCLE_POLICY=$(cat <<EOF
{
  "rules": [
    {
      "rulePriority": 1,
      "description": "Keep only the newest ${KEEP_LAST} images",
      "selection": {
        "tagStatus": "any",
        "countType": "imageCountMoreThan",
        "countNumber": ${KEEP_LAST}
      },
      "action": { "type": "expire" }
    }
  ]
}
EOF
)

command -v aws >/dev/null 2>&1 || { echo "ERROR: aws CLI not found" >&2; exit 1; }

created=0
existing=0

for svc in "${SERVICES[@]}"; do
  repo="${PREFIX}/${svc}"

  if err=$(aws ecr describe-repositories --repository-names "$repo" 2>&1 >/dev/null); then
    echo "Repository ${repo} already exists"
    existing=$((existing + 1))
  elif [[ "$err" == *RepositoryNotFoundException* ]]; then
    aws ecr create-repository \
      --repository-name "$repo" \
      --image-scanning-configuration scanOnPush=true \
      --output text --query 'repository.repositoryUri' >/dev/null
    echo "Created repository ${repo}"
    created=$((created + 1))
  else
    echo "ERROR checking ${repo}: ${err}" >&2
    exit 1
  fi

  # Both calls are idempotent, so they also repair drift on existing repos.
  aws ecr put-image-scanning-configuration \
    --repository-name "$repo" \
    --image-scanning-configuration scanOnPush=true >/dev/null
  aws ecr put-lifecycle-policy \
    --repository-name "$repo" \
    --lifecycle-policy-text "$LIFECYCLE_POLICY" >/dev/null
done

echo "Done: ${created} created, ${existing} already existed (${#SERVICES[@]} total)."