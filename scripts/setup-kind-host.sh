#!/usr/bin/env bash
set -euo pipefail

[ -f kind-config.yaml ] || { echo "Run this script from the repository root" >&2; exit 1; }

KIND_CLUSTER_NAME="${KIND_CLUSTER_NAME:-dhaba}"
KIND_VERSION="${KIND_VERSION:-v0.30.0}"
KUBECTL_VERSION="${KUBECTL_VERSION:-v1.32.0}"

case "$(uname -m)" in
  x86_64) PLATFORM=amd64 ;;
  aarch64|arm64) PLATFORM=arm64 ;;
  *) echo "Unsupported architecture" >&2; exit 1 ;;
esac

. /etc/os-release

if ! command -v docker >/dev/null 2>&1; then
  case "$ID" in
    ubuntu|debian)
      sudo apt-get update
      sudo apt-get install -y docker.io
      ;;
    amzn)
      sudo dnf install -y docker
      ;;
    *)
      echo "Unsupported distribution: $ID" >&2
      exit 1
      ;;
  esac
fi

sudo systemctl enable --now docker

if ! docker info >/dev/null 2>&1; then
  if [ "${DOCKER_GROUP_REEXEC:-}" != "1" ]; then
    sudo usermod -aG docker "$USER"
    export DOCKER_GROUP_REEXEC=1
    exec sg docker -c "$0"
  fi
  echo "Cannot access Docker as $USER" >&2
  exit 1
fi

if ! command -v kind >/dev/null 2>&1; then
  sudo curl -fsSL -o /usr/local/bin/kind "https://kind.sigs.k8s.io/dl/${KIND_VERSION}/kind-linux-${PLATFORM}"
  sudo chmod +x /usr/local/bin/kind
fi

if ! command -v kubectl >/dev/null 2>&1; then
  sudo curl -fsSL -o /usr/local/bin/kubectl "https://dl.k8s.io/release/${KUBECTL_VERSION}/bin/linux/${PLATFORM}/kubectl"
  sudo chmod +x /usr/local/bin/kubectl
fi

if ! kind get clusters | grep -qx "$KIND_CLUSTER_NAME"; then
  kind create cluster --name "$KIND_CLUSTER_NAME" --config kind-config.yaml
fi

kubectl cluster-info --context "kind-$KIND_CLUSTER_NAME"
echo "Cluster '$KIND_CLUSTER_NAME' is ready"
