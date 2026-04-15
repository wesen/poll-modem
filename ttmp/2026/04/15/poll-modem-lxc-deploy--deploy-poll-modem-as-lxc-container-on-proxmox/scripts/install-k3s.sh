#!/bin/bash
# Install k3s and ArgoCD in LXC container
# Run with: pct exec 300 -- bash -s < install-k3s.sh

set -euxo pipefail

export DEBIAN_FRONTEND=noninteractive

# Update and install dependencies
apt-get update
apt-get install -y \
  ca-certificates \
  curl \
  git \
  jq \
  socat \
  conntrack

# Install k3s (without Traefik since we'll use ArgoCD/nginx)
curl -sfL https://get.k3s.io | INSTALL_K3S_EXEC="server --disable traefik" sh -

# Wait for k3s to be ready
export KUBECONFIG=/etc/rancher/k3s/k3s.yaml
until kubectl get nodes >/dev/null 2>&1; do
  echo "Waiting for k3s..."
  sleep 5
done

echo "k3s is ready!"
kubectl get nodes

# Install cert-manager
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.0/cert-manager.yaml
kubectl wait --for=condition=Available deployment/cert-manager -n cert-manager --timeout=300s
kubectl wait --for=condition=Available deployment/cert-manager-cainjector -n cert-manager --timeout=300s
kubectl wait --for=condition=Available deployment/cert-manager-webhook -n cert-manager --timeout=300s

echo "cert-manager is ready!"

# Install ArgoCD
kubectl create namespace argocd --dry-run=client -o yaml | kubectl apply -f -
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml

# Wait for ArgoCD to be ready
kubectl wait --for=condition=Available deployment/argocd-server -n argocd --timeout=300s
kubectl wait --for=condition=Available deployment/argocd-repo-server -n argocd --timeout=300s
kubectl rollout status statefulset/argocd-application-controller -n argocd --timeout=300s

echo "ArgoCD is ready!"

# Get initial admin password
echo "ArgoCD initial admin password:"
kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d
echo

# Create local-path storage class for PVCs
cat <<'EOF' | kubectl apply -f -
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: local-path
provisioner: rancher.io/local-path
reclaimPolicy: Delete
volumeBindingMode: WaitForFirstConsumer
EOF

echo ""
echo "==================================="
echo "k3s + ArgoCD installation complete!"
echo "==================================="
echo ""
echo "kubectl: kubectl get nodes"
echo "ArgoCD UI: kubectl port-forward svc/argocd-server -n argocd 8080:443"
echo "ArgoCD CLI: curl -sSL -o argocd-linux-amd64 https://github.com/argoproj/argo-cd/releases/latest/download/argocd-linux-amd64"
echo ""
