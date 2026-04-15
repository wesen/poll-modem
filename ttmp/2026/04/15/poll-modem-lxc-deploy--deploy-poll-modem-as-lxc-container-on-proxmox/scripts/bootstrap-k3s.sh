#!/bin/bash
# Bootstrap k3s + cert-manager + ArgoCD on Proxmox LXC
# Run: pct exec 300 -- bash /tmp/bootstrap-k3s.sh
set -euo pipefail

export DEBIAN_FRONTEND=noninteractive
export KUBECONFIG=/etc/rancher/k3s/k3s.yaml

log() { echo "[$(date +%H:%M:%S)] $*"; }

# --- k3s ---
log "Installing k3s..."
mkdir -p /etc/rancher/k3s
cat > /etc/rancher/k3s/config.yaml <<'EOF'
write-kubeconfig-mode: "0644"
disable:
  - traefik
EOF

curl -sfL https://get.k3s.io | sh -

log "Waiting for k3s node ready..."
until kubectl get nodes -o wide 2>/dev/null | grep -q " Ready"; do
  sleep 5
done
log "k3s ready!"
kubectl get nodes -o wide

# --- cert-manager ---
log "Installing cert-manager..."
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.0/cert-manager.yaml
kubectl wait --for=condition=Available deployment/cert-manager -n cert-manager --timeout=300s
kubectl wait --for=condition=Available deployment/cert-manager-cainjector -n cert-manager --timeout=300s
kubectl wait --for=condition=Available deployment/cert-manager-webhook -n cert-manager --timeout=300s
log "cert-manager ready!"

# --- ArgoCD ---
log "Installing ArgoCD..."
kubectl create namespace argocd --dry-run=client -o yaml | kubectl apply -f -
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml

log "Waiting for ArgoCD..."
kubectl wait --for=condition=Available deployment/argocd-server -n argocd --timeout=300s
kubectl wait --for=condition=Available deployment/argocd-repo-server -n argocd --timeout=300s
kubectl rollout status statefulset/argocd-application-controller -n argocd --timeout=300s
log "ArgoCD ready!"

# --- Summary ---
ARGO_PASS=$(kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d)
log "========================================="
log "  k3s + ArgoCD installation complete!"
log "========================================="
log "  kubectl: export KUBECONFIG=/etc/rancher/k3s/k3s.yaml"
log "  ArgoCD UI: kubectl port-forward svc/argocd-server -n argocd 8080:443"
log "  ArgoCD admin password: ${ARGO_PASS}"
log "  Password also saved to /root/argocd-password"
echo -n "$ARGO_PASS" > /root/argocd-password
log "========================================="
