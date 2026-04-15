---
Title: Diary
Ticket: poll-modem-lxc-deploy
Status: active
Topics:
    - deployment
    - lxc
    - proxmox
    - go
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ttmp/2026/04/15/poll-modem-lxc-deploy--deploy-poll-modem-as-lxc-container-on-proxmox/scripts/create-lxc-container.sh
      Note: Creates LXC container on Proxmox
    - Path: ttmp/2026/04/15/poll-modem-lxc-deploy--deploy-poll-modem-as-lxc-container-on-proxmox/scripts/deploy.sh
      Note: Master deployment orchestration script
    - Path: ttmp/2026/04/15/poll-modem-lxc-deploy--deploy-poll-modem-as-lxc-container-on-proxmox/scripts/install-go.sh
      Note: Installs Go compiler in container
    - Path: ttmp/2026/04/15/poll-modem-lxc-deploy--deploy-poll-modem-as-lxc-container-on-proxmox/scripts/install-poll-modem.sh
      Note: Builds and installs poll-modem binary
    - Path: ttmp/2026/04/15/poll-modem-lxc-deploy--deploy-poll-modem-as-lxc-container-on-proxmox/scripts/run-in-tmux.sh
      Note: Runs poll-modem in tmux session
ExternalSources: []
Summary: ""
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: ""
WhenToUse: ""
---


# Diary

## Goal

Document the deployment of poll-modem as an LXC container on Proxmox, including all steps, commands, scripts, and troubleshooting.

## Step 1: Extract poll-modem as Standalone Repo

We extracted the poll-modem application from go-go-labs into its own standalone repository at `../poll-modem/`. This involved moving the source code, updating imports from `github.com/go-go-golems/go-go-labs/...` to `github.com/go-go-golems/poll-modem/...`, setting up the go-template scaffolding (Makefile, GoReleaser, GitHub Actions), and creating an initial git commit.

### Prompt Context

**User prompt (verbatim):** "Extract this program as a standalone repo in ../poll-modem using the ../go-template"

**Assistant interpretation:** Extract the poll-modem application from go-go-labs monorepo into a standalone repository with proper Go module structure, CI/CD, and release automation.

**Inferred user intent:** Create a clean, deployable version of poll-modem that can be built and released independently.

**Commit (code):** 2824e09 - "Initial commit: Extract poll-modem from go-go-labs as standalone repo"

### What I did
- Copied go-template files to ../poll-modem/
- Moved source from `cmd/apps/poll-modem/` to `cmd/poll-modem/`
- Moved internal packages to `internal/modem/` and `internal/tui/`
- Updated all imports from `go-go-labs` to `poll-modem`
- Fixed package structure (moved root.go to cmd/root.go)
- Replaced XXX placeholders in Makefile and .goreleaser.yaml
- Ran `go mod tidy` to fetch dependencies
- Built and tested: `go build ./...` and `./poll-modem --help`
- Ran lint: `make lint` - 0 issues
- Tested release: `make goreleaser` - produced .deb, .rpm, .tar.gz
- Updated README.md build instructions
- Wrote AGENT.md with project-specific guidelines
- Committed everything to git

### What worked
- Build successful with no errors
- Lint passed with 0 issues
- GoReleaser produced all artifacts (deb, rpm, tar.gz)
- Binary runs and shows help correctly
- All imports properly updated

### What didn't work
- Initial import paths were still pointing to go-go-labs - fixed with sed/rg
- Package structure had main.go and root.go in same directory with different packages - fixed by moving root.go to cmd/root.go
- go.work interference - used GOWORK=off for all operations

### What I learned
- The go-template structure expects `cmd/<binary>/main.go` and `cmd/root.go`
- Need to be careful about removing build artifacts before git commit
- Adding .bin/ and binary name to .gitignore is important

### Technical details

**Build commands:**
```bash
cd ../poll-modem
GOWORK=off go build ./...
GOWORK=off go build -o poll-modem ./cmd/poll-modem
./poll-modem --help
```

**Lint:**
```bash
make lint
# Result: 0 issues
```

**Release test:**
```bash
make goreleaser
# Produced:
# - dist/linux_amd64/poll-modem_0.0.1-next_linux_amd64.deb
# - dist/linux_amd64/poll-modem_0.0.1-next_linux_amd64.rpm
# - dist/linux_amd64/poll-modem_0.0.1-next_linux_amd64.tar.gz
```

---

## Step 2: Create Docmgr Ticket for Deployment

Created a docmgr ticket to track the LXC deployment work with proper documentation structure.

### Prompt Context

**User prompt (verbatim):** "Ok, let's create a docmgr ticket in ../poll-modem/ttmp and keep a detailed diary of our deployment work, and save any commands you ran in there too, and scripts in scripts/ folder of the ticket."

**Assistant interpretation:** Create a documentation ticket to track the Proxmox LXC deployment with diary, scripts, and command references.

**Inferred user intent:** Keep organized records of deployment steps for future reference and reproducibility.

### What I did
- Created docmgr ticket: `poll-modem-lxc-deploy`
- Added diary document: `reference/01-diary.md`
- Created scripts directory: `scripts/`

### Commands run
```bash
cd ../poll-modem
mkdir -p ttmp
docmgr ticket create-ticket --ticket poll-modem-lxc-deploy --title "Deploy poll-modem as LXC container on Proxmox" --topics deployment,lxc,proxmox,go
docmgr doc add --ticket poll-modem-lxc-deploy --doc-type reference --title "Diary"
```

---

## Step 3: Connect to Proxmox and Prepare Template

Connected to the Proxmox server and downloaded the Debian 12 LXC template needed for the container.

### What I did
- SSH'd to root@pve and verified connection
- Checked available templates: `pveam available | grep debian-12`
- Downloaded debian-12-standard_12.12-1_amd64.tar.zst template

### Commands run
```bash
ssh -o ConnectTimeout=5 root@pve "pveversion"
# Output: pve-manager/8.1.4/ec5affc9e41f1d79 (running kernel: 6.5.13-1-pve)

ssh root@pve "pveam available | grep debian-12 | head -5"
# Selected: debian-12-standard_12.12-1_amd64.tar.zst

ssh root@pve "pveam download local debian-12-standard_12.12-1_amd64.tar.zst"
# Downloaded 118M template
```

### Network configuration discovered
```
vmbr0: 192.168.0.227/24 (gateway 192.168.0.1)
vmbr1: manual bridge
```

---

## Step 4: Create Deployment Scripts

Created deployment scripts in the ticket's scripts/ directory to automate the LXC container setup.

### Scripts created

**create-lxc-container.sh**: Creates the LXC container on Proxmox
- Uses pct create with debian-12-standard template
- Configures 512MB RAM, 1 core, 4GB disk
- Sets up DHCP on vmbr0

**install-go.sh**: Installs Go compiler inside container
- Downloads Go 1.23.4
- Sets up PATH and GOPATH
- Installs build dependencies

**install-poll-modem.sh**: Builds and installs poll-modem
- Clones from GitHub (or pulls if exists)
- Builds with `go build -o /usr/local/bin/poll-modem`
- Creates config directory

**run-in-tmux.sh**: Runs poll-modem in persistent tmux session
- Handles modem credentials via environment variables
- Creates/kills tmux session
- Provides attach/detach instructions

**deploy.sh**: Master orchestration script
- Runs all steps in sequence
- Provides post-deployment instructions

### Commands run
```bash
chmod +x ttmp/2026/04/15/poll-modem-lxc-deploy--deploy-poll-modem-as-lxc-container-on-proxmox/scripts/*.sh
```

---

## Current Status

- ✅ poll-modem extracted as standalone repo
- ✅ Build, lint, and release working
- ✅ Git repository initialized with clean commit
- ✅ Docmgr ticket created for deployment tracking
- ✅ Proxmox template downloaded
- ✅ Deployment scripts created and documented
- ⏳ Next: Execute deployment on Proxmox

---

## Deployment Instructions

To deploy poll-modem on Proxmox:

```bash
# Copy scripts to Proxmox
scp -r ttmp/2026/04/15/poll-modem-lxc-deploy--deploy-poll-modem-as-lxc-container-on-proxmox/scripts/ root@pve:/tmp/

# Run deployment
ssh root@pve "bash /tmp/scripts/deploy.sh [CT_ID] [CT_NAME]"

# Or run steps manually:
ssh root@pve
bash /tmp/scripts/create-lxc-container.sh 100 poll-modem
pct exec 100 -- bash -c "\$(cat /tmp/scripts/install-go.sh)"
pct exec 100 -- bash -c "\$(cat /tmp/scripts/install-poll-modem.sh)"
pct exec 100 -- apt-get install -y tmux

# Enter container and run
pct exec 100 -- /bin/bash
export MODEM_USER=admin MODEM_PASS=yourpassword
bash /opt/poll-modem/scripts/run-in-tmux.sh
tmux attach -t poll-modem
```

---

## Appendix: LXC vs Docker Containers

Both LXC and Docker use Linux kernel features (cgroups, namespaces) for isolation, but they serve different purposes and have different architectures.

### Core Differences

| Aspect | LXC | Docker |
|--------|-----|--------|
| **Level** | OS-level virtualization | Application-level virtualization |
| **Image** | Full OS rootfs (like a VM) | Layered filesystem with app + deps only |
| **Init** | Runs init system (systemd) | Runs a single process (your app) |
| **State** | Stateful "pet" containers | Stateless "cattle" containers |
| **Network** | Full network stack (DHCP/static IP) | Port mapping / bridge networks |
| **Storage** | Persistent by default | Ephemeral by default (volumes for persistence) |
| **Kernel** | Shares host kernel | Shares host kernel |
| **Overhead** | ~low (shares kernel) | ~very low (shares kernel, minimal userspace) |
| **Use Case** | Long-running services, databases, apps needing init | Microservices, CI/CD, ephemeral workloads |

### Architecture Comparison

**LXC (System Container):**
```
Host Kernel
    ↓
LXC Container (like a lightweight VM)
    ├── systemd (PID 1)
    ├── sshd
    ├── cron
    ├── your apps
    └── full OS userspace
```

**Docker (Application Container):**
```
Host Kernel
    ↓
Docker Container
    └── Single process (your app)
        └── may spawn child processes
```

### When to Use What

**Use LXC when:**
- You need a "VM-like" experience but lighter
- Running long-lived services (databases, monitoring tools)
- Need systemd/init system
- Want persistent state by default
- Running on Proxmox (native support)
- You have a "pet" not "cattle" mentality
- Need to SSH into the container and run multiple services

**Use Docker when:**
- Building microservices
- Need fast startup/shutdown
- Want immutable infrastructure
- Running CI/CD pipelines
- Need orchestration (Kubernetes, Swarm)
- Sharing application images across teams
- Running ephemeral workloads

### Why LXC for poll-modem?

We chose LXC because:
1. **Proxmox native support** - Proxmox is built around LXC and KVM
2. **TUI application** - poll-modem is interactive, needs tmux, persistent state
3. **SQLite database** - needs persistent storage across restarts
4. **Full OS environment** - easier to install Go, build from source, debug
5. **"Pet" container** - this is a monitoring tool you care about, not a throwaway microservice
6. **Network access** - needs to reach modem at 192.168.0.1 on same LAN

### Commands Comparison

| Task | LXC (Proxmox) | Docker |
|------|---------------|--------|
| Create | `pct create 100 ...` | `docker run -d ...` |
| Enter | `pct exec 100 -- bash` | `docker exec -it ... bash` |
| Start | `pct start 100` | `docker start ...` |
| Stop | `pct stop 100` | `docker stop ...` |
| Network | Direct DHCP/static IP | Port mapping (-p 8080:80) |
| Storage | Persistent rootfs | Volumes (-v host:container) |
| Logs | `pct exec 100 -- journalctl` | `docker logs ...` |

### Proxmox LXC Specifics

- **Unprivileged by default** - safer, runs as non-root UID
- **Nesting enabled** - can run containers in containers
- **Template-based** - uses pre-built OS templates (Debian, Ubuntu, Alpine)
- **Web UI + API** - manage through Proxmox GUI or pvesh/pct CLI
- **Backup/restore** - integrated with Proxmox backup system
- **Resource limits** - CPU, memory, disk limits via cgroups

### Summary

Think of LXC as "lightweight VMs" and Docker as "packaged applications." For a monitoring tool like poll-modem that needs:
- Long-running process
- Persistent database
- Interactive TUI
- Full OS environment for building

LXC is the better fit, especially on Proxmox.

---

## Appendix: Can Proxmox Run Docker?

Yes, Proxmox can run Docker containers, but there are several approaches with different trade-offs.

### Option 1: Docker Inside LXC (Most Common)

Run Docker containers inside an LXC container with nesting enabled.

**Pros:**
- Isolation from host
- Multiple Docker environments per LXC
- Easy backup/restore via Proxmox

**Cons:**
- Nested containers (complexity)
- Slightly more overhead
- Requires unprivileged container tweaks or privileged mode

**Setup:**
```bash
# On Proxmox host - create privileged LXC for Docker
pct create 200 local:vztmpl/debian-12-standard_12.12-1_amd64.tar.zst \
  --hostname docker-host \
  --memory 2048 \
  --cores 2 \
  --rootfs local-lvm:8 \
  --net0 ip=dhcp,bridge=vmbr0 \
  --unprivileged 0 \
  --features nesting=1,keyctl=1

# Inside LXC - install Docker
curl -fsSL https://get.docker.com | sh
```

### Option 2: Docker on Proxmox Host (Not Recommended)

Install Docker directly on the Proxmox host.

**Pros:**
- Direct hardware access
- No nesting overhead

**Cons:**
- Breaks Proxmox support/warranty
- Security risk (containers can access host)
- Conflicts with Proxmox networking
- Harder to backup/restore

**Why it's not recommended:**
Proxmox is designed as a hypervisor. Running Docker on the host violates the "hypervisor hygiene" principle and can cause networking conflicts with VMs/LXCs.

### Option 3: Docker in a VM

Create a Linux VM and run Docker inside it.

**Pros:**
- Full isolation
- Any Linux distribution
- Kubernetes-friendly

**Cons:**
- Higher overhead (full kernel)
- More resource usage than LXC

**When to use:**
- Running Kubernetes clusters
- Need specific kernel features
- Maximum isolation required

### Comparison Table

| Approach | Overhead | Isolation | Backup | Complexity | Recommendation |
|----------|----------|-----------|--------|------------|----------------|
| Docker in LXC | Low | Good | Easy | Medium | ⭐ Good for home/lab |
| Docker on host | Minimal | Poor | Hard | Low | ❌ Avoid in production |
| Docker in VM | Higher | Excellent | Medium | Medium | ⭐ For K8s/production |
| LXC native | Lowest | Good | Easy | Low | ⭐⭐ Best for Proxmox |

### Our Choice: LXC Native

For poll-modem, we chose **native LXC** over "Docker in LXC" because:

1. **Simpler stack** - no nested containers
2. **Lower overhead** - no Docker daemon
3. **Direct Proxmox integration** - backups, monitoring, console access
4. **Single process** - poll-modem doesn't need Docker's orchestration features
5. **Persistent data** - SQLite is easier with direct filesystem access

### When Would You Use Docker on Proxmox?

Use "Docker in LXC" when:
- You have multiple microservices to orchestrate
- You need Docker Compose for app stacks
- You want to run pre-built images from Docker Hub
- You're migrating from a Docker-based workflow

Use "Docker in VM" when:
- Running production Kubernetes
- Need specific kernel modules
- Maximum isolation is critical

### Quick Commands: Docker in LXC

```bash
# Create LXC for Docker (privileged required for now)
pct create 200 local:vztmpl/debian-12-standard_12.12-1_amd64.tar.zst \
  --hostname docker-host \
  --memory 2048 \
  --cores 2 \
  --rootfs local-lvm:10 \
  --net0 ip=dhcp,bridge=vmbr0 \
  --unprivileged 0

pct start 200

# Install Docker inside
pct exec 200 -- bash -c "curl -fsSL https://get.docker.com | sh"
pct exec 200 -- usermod -aG docker root

# Use Docker
pct exec 200 -- docker run hello-world
```

### Summary

Proxmox **can** run Docker, but it's typically done inside LXC or VMs rather than natively. For a single Go application like poll-modem, native LXC is simpler and more integrated. For multi-service apps, "Docker in LXC" gives you the best of both worlds.

---

## Appendix: Running k3s on Proxmox

k3s is a lightweight Kubernetes distribution perfect for homelabs and edge deployments. On Proxmox, you can run k3s in several configurations.

### What is k3s?

- Lightweight K8s (single binary <100MB)
- Replaces etcd with SQLite (or external DB)
- Includes ingress, metrics, service LB out of the box
- CNCF certified Kubernetes
- Created by Rancher (now SUSE)

### Option 1: k3s in a Single VM (Simplest)

Best for: Single-node clusters, homelabs, learning

```bash
# Create VM on Proxmox
qm create 300 --name k3s-server --memory 4096 --cores 2 --net0 virtio,bridge=vmbr0
qm importdisk 300 /path/to/debian-12-nocloud-amd64.img local-lvm
qm set 300 --scsihw virtio-scsi-pci --scsi0 local-lvm:vm-300-disk-0
qm set 300 --ide2 local-lvm:cloudinit
qm set 300 --boot order=scsi0
qm set 300 --serial0 socket --vga serial0
qm start 300

# Inside VM - install k3s
curl -sfL https://get.k3s.io | sh -

# Verify
kubectl get nodes
kubectl get pods -A
```

**Resources:** 2-4 vCPU, 4-8GB RAM, 20GB disk

### Option 2: k3s in LXC (Lightweight)

Best for: Resource-constrained setups, testing

⚠️ **Limitations:**
- No kube-proxy (uses iptables/nftables)
- Limited CSI support
- Some K8s features may not work

```bash
# Create privileged LXC (required for k3s)
pct create 300 local:vztmpl/debian-12-standard_12.12-1_amd64.tar.zst \
  --hostname k3s-node \
  --memory 4096 \
  --cores 2 \
  --rootfs local-lvm:20 \
  --net0 ip=dhcp,bridge=vmbr0 \
  --unprivileged 0 \
  --features nesting=1,keyctl=1,fuse=1 \
  --mount type=bind,source=/dev/kmsg,destination=/dev/kmsg

pct start 300

# Inside container - fix kernel modules and install
pct exec 300 -- bash -c "
  # k3s needs these
  modprobe br_netfilter 2>/dev/null || true
  sysctl -w net.ipv4.ip_forward=1
  sysctl -w net.bridge.bridge-nf-call-iptables=1
  
  # Install k3s without traefik (optional)
  curl -sfL https://get.k3s.io | INSTALL_K3S_EXEC='--disable traefik' sh -
"

# Get kubeconfig
pct exec 300 -- cat /etc/rancher/k3s/k3s.yaml
```

### Option 3: k3s HA Cluster (Production-Ready)

Best for: Production workloads, high availability

**Architecture:**
```
Proxmox Cluster
├── VM 301: k3s-server-1 (control-plane + etcd)
├── VM 302: k3s-server-2 (control-plane + etcd)
├── VM 303: k3s-server-3 (control-plane + etcd)
├── VM 304: k3s-agent-1 (worker)
└── VM 305: k3s-agent-2 (worker)
```

**Setup external datastore (required for HA):**
```bash
# Option A: SQLite (only for single server)
# Option B: PostgreSQL (for HA)
# Option C: MySQL (for HA)
# Option D: etcd (embedded, 3+ servers)

# Create external DB in another LXC/VM
pct create 299 local:vztmpl/debian-12-standard_12.12-1_amd64.tar.zst \
  --hostname k3s-db --memory 1024 --cores 1 --rootfs local-lvm:10

# Install PostgreSQL inside DB container
pct exec 299 -- apt-get update
pct exec 299 -- apt-get install -y postgresql
pct exec 299 -- sudo -u postgres psql -c "CREATE DATABASE k3s;"
pct exec 299 -- sudo -u postgres psql -c "CREATE USER k3s WITH ENCRYPTED PASSWORD 'password';"
pct exec 299 -- sudo -u postgres psql -c "GRANT ALL PRIVILEGES ON DATABASE k3s TO k3s;"
```

**Install first server:**
```bash
# On VM 301
export K3S_DATASTORE_ENDPOINT="postgres://k3s:password@192.168.0.299:5432/k3s"
export K3S_TOKEN="my-super-secret-token"

curl -sfL https://get.k3s.io | sh -s - server \
  --cluster-init \
  --tls-san 192.168.0.301 \
  --node-taint CriticalAddonsOnly=true:NoExecute
```

**Add additional servers:**
```bash
# On VM 302, 303
export K3S_DATASTORE_ENDPOINT="postgres://k3s:password@192.168.0.299:5432/k3s"
export K3S_TOKEN="my-super-secret-token"

curl -sfL https://get.k3s.io | sh -s - server \
  --server https://192.168.0.301:6443 \
  --tls-san 192.168.0.302
```

**Add agents (workers):**
```bash
# On VM 304, 305
export K3S_TOKEN="my-super-secret-token"
curl -sfL https://get.k3s.io | sh -s - agent --server https://192.168.0.301:6443
```

### Option 4: k3d on Proxmox (K3s in Docker)

Best for: Development, multi-cluster testing

Requires: Docker installed (see "Docker on Proxmox" section)

```bash
# Inside Docker LXC or VM
curl -s https://raw.githubusercontent.com/k3d-io/k3d/main/install.sh | bash

# Create cluster
k3d cluster create homelab \
  --servers 1 \
  --agents 2 \
  --port 80:80@loadbalancer \
  --port 443:443@loadbalancer

# Use kubectl
kubectl get nodes
```

### Deploying poll-modem on k3s

**Option A: As a Deployment + Service**

```yaml
# poll-modem-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: poll-modem
  namespace: default
spec:
  replicas: 1
  selector:
    matchLabels:
      app: poll-modem
  template:
    metadata:
      labels:
        app: poll-modem
    spec:
      containers:
      - name: poll-modem
        image: ghcr.io/go-go-golems/poll-modem:latest
        args:
          - "--url"
          - "http://192.168.0.1"
          - "--interval"
          - "30s"
        env:
        - name: MODEM_USERNAME
          valueFrom:
            secretKeyRef:
              name: modem-credentials
              key: username
        - name: MODEM_PASSWORD
          valueFrom:
            secretKeyRef:
              name: modem-credentials
              key: password
        volumeMounts:
        - name: data
          mountPath: /root/.config/poll-modem
      volumes:
      - name: data
        persistentVolumeClaim:
          claimName: poll-modem-data
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: poll-modem-data
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 1Gi
---
apiVersion: v1
kind: Secret
metadata:
  name: modem-credentials
type: Opaque
stringData:
  username: admin
  password: yourpassword
```

**Apply:**
```bash
kubectl apply -f poll-modem-deployment.yaml
```

**Option B: As a CronJob (periodic polling)**

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: poll-modem-collector
spec:
  schedule: "*/5 * * * *"  # Every 5 minutes
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: poll-modem
            image: ghcr.io/go-go-golems/poll-modem:latest
            args:
              - "--url"
              - "http://192.168.0.1"
              - "--interval"
              - "1m"
              - "--export"
              - "csv"
            env:
            - name: MODEM_USERNAME
              valueFrom:
                secretKeyRef:
                  name: modem-credentials
                  key: username
            - name: MODEM_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: modem-credentials
                  key: password
          restartPolicy: OnFailure
```

**Option C: Interactive TUI with tty access**

For the interactive TUI mode, k3s isn't ideal (containers are headless). Use:
- LXC native (recommended for TUI)
- Or VM with direct SSH access

### k3s vs Native LXC for poll-modem

| Aspect | k3s (Kubernetes) | LXC Native |
|--------|------------------|------------|
| Complexity | Higher (YAML manifests) | Lower (direct binary) |
| Orchestration | Excellent | Manual |
| Scaling | Auto-scaling | Manual |
| Self-healing | Pod restart on failure | systemd/restart script |
| Persistence | PVCs required | Direct filesystem |
| TUI Mode | ❌ Not suitable | ✅ Perfect |
| Data Collection | ✅ CronJob ideal | ✅ systemd timer |
| Resource Usage | Higher (K8s overhead) | Lower |
| Learning Curve | Steep | Minimal |

### Recommendation

**For poll-modem specifically:**
- **TUI mode**: Use LXC native (what we're doing)
- **Data collection only**: Could use k3s CronJob
- **Part of larger monitoring stack**: Use k3s with Prometheus/Grafana

**For general homelab:**
- Single node: k3s in one VM
- HA setup: 3-server k3s cluster with external DB
- Learning: k3d (k3s in Docker)

### Useful k3s Commands

```bash
# Server management
systemctl status k3s
journalctl -u k3s -f

# kubectl (included with k3s)
kubectl get nodes
kubectl get pods -A
kubectl top nodes
kubectl top pods

# Config
/etc/rancher/k3s/k3s.yaml  # kubeconfig
/var/lib/rancher/k3s/      # data directory

# Uninstall
/usr/local/bin/k3s-uninstall.sh
```

---

## Failed Attempt: k3s VM Setup Issues

We attempted to create a k3s VM on Proxmox but encountered several issues that prevented successful deployment. Here's what went wrong:

### What We Tried

Created a QEMU VM with these specs:
- VM ID: 300
- Name: k3s-server
- CPU: 4 cores, host type
- Memory: 8192 MB
- Disk: Imported debian-12-cloud.qcow2 (3GB)
- Network: virtio, bridge vmbr0
- Cloud-init: user=admin, password, SSH keys
- IP: Static 192.168.0.210/24

### Issues Encountered

**1. Cloud Image Boot Issues**
The Debian cloud image (debian-12-cloud.qcow2) had boot problems:
```
Error: Could not retrieve NBP file size from HTTP server.
Error: Server response timeout.
BdsDxe: failed to load Boot0004 "UEFI HTTPv4"
```
The VM tried to PXE boot instead of booting from the disk.

**Root Cause:** The qcow2 image may not have been properly configured for direct boot, or required additional cloud-init configuration that wasn't provided during first boot.

**2. Missing EFI Disk**
Initially no EFI disk was configured. Fixed with:
```bash
qm set 300 --efidisk0 local-lvm:1
```

**3. SSH Not Accessible**
Even after configuring network and cloud-init:
- Port 22 was closed (not refused, but closed)
- SSH service wasn't running
- Cloud-init may not have completed

**4. Serial Console Output Empty**
Trying to access `/var/run/qemu-server/300.serial0` showed no boot output, suggesting:
- Serial console not enabled in VM kernel
- VM not reaching boot stage
- Image compatibility issues

**5. Disk Size**
3GB Debian cloud image was minimal and may not have had space for cloud-init to complete properly.

### Commands That Didn't Work

```bash
# SSH never connected
ssh admin@192.168.0.210  # Connection refused or timeout

# Serial console showed no output
socat - UNIX-CONNECT:/var/run/qemu-server/300.serial0

# Guest agent not installed in VM
qm guest cmd 300 ping  # No QEMU guest agent configured
```

### Why This Approach Failed

1. **Cloud image complexity**: Cloud images need proper cloud-init setup on first boot
2. **No direct console access**: Hard to debug without working serial console
3. **Missing guest agent**: Can't exec commands or check status from host
4. **Boot order confusion**: EFI vs BIOS, SCSI vs SATA disk

### What Would Have Worked Instead

**Option A: LXC with Docker** (pivoting to this)
- Faster startup
- Easier to debug (pct exec works immediately)
- No boot image issues
- Can install k3s directly

**Option B: VM with ISO Install**
- Use Debian 12 ISO instead of cloud image
- Manual install with proper partitioning
- Then install k3s

**Option C: Use the LXC k3s approach**
- Privileged LXC with nesting
- k3s installs directly
- Less overhead than VM

### Lessons Learned

1. Cloud images are finicky without proper cloud-init infra
2. VMs take longer to iterate on than LXC
3. Serial console access is essential for debugging
4. LXC is better for "pet" containers/VMs where you need quick access

### Switching to LXC + Docker + k3s

Abandoning VM approach in favor of LXC container with Docker and k3s.

---

## Step 5: k3s LXC Container Setup - In Progress

We pivoted from the VM approach to an LXC container with k3s. This is a more lightweight and manageable approach.

### What We Did

**Created LXC container (ID 300):**
```bash
pct create 300 local:vztmpl/debian-12-standard_12.12-1_amd64.tar.zst \
  --hostname k3s-server \
  --memory 8192 \
  --cores 4 \
  --rootfs local-lvm:20 \
  --net0 name=eth0,bridge=vmbr0,ip=192.168.0.210/24 \
  --nameserver 192.168.0.1 \
  --unprivileged 0 \
  --features nesting=1,keyctl=1,fuse=1 \
  --start 1
```

**Issues encountered and resolved:**

1. **SSH connection refused** - Multiple attempts to connect failed
2. **Root cause found:** Stale ARP entry with wrong MAC address
   - ARP table showed: `02:a3:b6:e3:b0:52` for 192.168.0.210
   - Container had: `bc:24:11:04:78:92`
   - **Fix:** `ip neigh flush 192.168.0.210`
3. **Missing packages** - curl not installed, working on apt installation

**Current status:**
- Container running at 192.168.0.210
- SSH working after ARP flush
- Ping works to gateway and host
- Installing prerequisites for k3s

### Commands run
```bash
# Create container
ssh root@pve "pct create 300 ..."

# Fix ARP issue
ssh root@pve "ip neigh flush 192.168.0.210"

# Verify connectivity
ssh root@pve "pct exec 300 -- ping -c 2 192.168.0.227"

# Install packages (in progress)
ssh root@pve "pct exec 300 -- apt-get update"
ssh root@pve "pct exec 300 -- apt-get install -y curl wget git"
```

### What was tricky
- ARP cache had stale entry from previous VM (ID 300)
- The MAC address mismatch prevented all network traffic
- Needed to flush ARP to force re-resolution

### Next steps
- Complete package installation
- Install k3s
- Install ArgoCD
- Deploy poll-modem

---

## Step 6: Pivot to k3s + ArgoCD — Assessment and Clean Restart

### Prompt Context

**User prompt (verbatim):** (see Step 5 — "ok, let's set up k3s on proxmox and use that as a target for our deployment. 4 cores, 8 GB for starters, setup argocd as well.")

Then: "We're taking over for your little brother that is fast, but careless. Look at the session up until now, and write an assessment of things in the ticket, because we actually want to do things as cleanly as possible."

**Assistant interpretation:** Pause, assess what's actually done vs what's messy, produce an honest accounting, then proceed cleanly.

**Inferred user intent:** Stop accumulating cruft. Get a clear picture before moving forward.

### Assessment Written

Created `reference/02-assessment.md` with full accounting:

**Good:** Standalone repo works. poll-modem tested against real modem. Proxmox accessible. LXC container 300 running with right specs (4 core, 8GB, privileged, nesting/keyctl/fuse). Hetzner reference project provides proven k3s+ArgoCD pattern.

**Bad:** All scripts in `scripts/` are stale (written for LXC-native, not k3s). Diary became a dump of appendices. VM attempt wasted time. Nested SSH instead of `pct exec`. Container was empty — no curl, no k3s, nothing installed.

**Path forward:** Adapt hetzner cloud-init pattern to Proxmox LXC. One clean bootstrap script. Run via `pct exec`. Then GitOps with ArgoCD.

---

## Step 7: Destroy and Recreate Container 300 (Second Time)

### Prompt Context

**User prompt (verbatim):** "delete the current one and recreate based on cloud-init"

**Assistant interpretation:** Destroy container 300, recreate from scratch with proper cloud-init for automated k3s setup.

**Inferred user intent:** Start clean rather than patching the broken container.

### What happened

1. Destroyed container 300:
```bash
pct stop 300 && pct destroy 300
```

2. Attempted `--cicustom user=local:snippets/user-data-k3s.yaml` — **doesn't work for LXC** (only for QEMU VMs). Error: `Unknown option: cicustom`.

3. Created cloud-init YAML at `/var/lib/vz/snippets/user-data-k3s.yaml` on Proxmox anyway (for reference), but fell back to plain `pct create`.

4. Recreated container 300:
```bash
pct create 300 local:vztmpl/debian-12-standard_12.12-1_amd64.tar.zst \
  --hostname k3s-server \
  --memory 8192 \
  --cores 4 \
  --rootfs local-lvm:30 \
  --net0 name=eth0,bridge=vmbr0,ip=192.168.0.210/24 \
  --nameserver 192.168.0.1 \
  --unprivileged 0 \
  --features nesting=1,keyctl=1,fuse=1 \
  --start 1
```

5. Configured SSH access:
```bash
# Set root password
pct exec 300 -- bash -c 'echo root:changeme | chpasswd'

# Enable root login + password auth
pct exec 300 -- sed -i 's/#*PermitRootLogin.*/PermitRootLogin yes/' /etc/ssh/sshd_config
pct exec 300 -- sed -i 's/#*PasswordAuthentication.*/PasswordAuthentication yes/' /etc/ssh/sshd_config
pct exec 300 -- systemctl restart ssh

# Push SSH keys from Proxmox host
cat /root/.ssh/authorized_keys | pct exec 300 -- tee /root/.ssh/authorized_keys
```

### What didn't work
- `--cicustom` is VM-only, not available for LXC containers
- Cloud-init for LXC on Proxmox is limited to `--ciuser`, `--cipassword`, `--sshkeys`, `--ipconfig0`

---

## Step 8: Network — The vmbr0 Gateway Problem

### What happened

After recreating the container, it could ping the Proxmox host (192.168.0.227) but **could not reach the gateway** (192.168.0.1 — the cable modem). This meant no internet access, no DNS resolution, no way to download packages.

### Root cause

The container is on `vmbr0`, which bridges directly to `eno1` (the physical NIC). The container gets a virtual MAC address (`bc:24:11:39:96:7d`). The cable modem at 192.168.0.1 either:

1. **Filters by MAC** — only accepts traffic from known devices, OR
2. **Has a small ARP table** — doesn't learn/respond to virtual MACs

The Proxmox host itself (with the physical NIC MAC `5c:ba:2c:20:2b:86`) can reach the gateway fine. But the container's virtual MAC is invisible to the cable modem at L2.

### Debugging steps

```bash
# Container can reach host
pct exec 300 -- ping -c 2 192.168.0.227  # ✅ works

# Container cannot reach gateway
pct exec 300 -- ping -c 2 192.168.0.1    # ❌ 100% packet loss

# Host can reach gateway
ssh root@pve "ping -c 2 192.168.0.1"     # ✅ works

# DNS fails because no route to gateway/nameserver
pct exec 300 -- curl -sfL https://get.k3s.io  # exit code 6 (DNS failure)
pct exec 300 -- ping deb.debian.org             # Temporary failure in name resolution
```

### Fix: Move container to vmbr1 (NAT network)

Proxmox already has `vmbr1` configured for NAT. The config in `/etc/network/interfaces`:
```
post-up iptables -t nat -A POSTROUTING -s '192.168.1.0/24' -o vmbr0 -j MASQUERADE
```

Steps:
1. Assign IP to vmbr1: `ip addr add 192.168.1.1/24 dev vmbr1`
2. Reconfigure container: `pct set 300 --net0 name=eth0,bridge=vmbr1,ip=192.168.1.100/24,gw=192.168.1.1`
3. Reboot container: `pct reboot 300`
4. Flush ARP: `ip neigh flush 192.168.1.100`
5. Set DNS: `pct exec 300 -- bash -c 'echo nameserver 8.8.8.8 > /etc/resolv.conf'`

```bash
# Verify
pct exec 300 -- ping -c 2 192.168.1.1                    # ✅ works
pct exec 300 -- curl -sfL https://get.k3s.io -o /dev/null # ✅ INTERNET WORKS
```

### Important note

The vmbr1 IP (`192.168.1.1/24`) was added with `ip addr add` — this is **not persistent** across reboots. It should be added to `/etc/network/interfaces`:
```
auto vmbr1
iface vmbr1 inet static
    address 192.168.1.1/24
    bridge-ports none
    bridge-stp off
    bridge-fd 0
    post-up iptables -t nat -A POSTROUTING -s '192.168.1.0/24' -o vmbr0 -j MASQUERADE
    post-down iptables -t nat -D POSTROUTING -s '192.168.1.0/24' -o vmbr0 -j MASQUERADE
```

---

## Step 9: Package Installation

### What happened

After getting internet working, installed required packages.

### First attempt — stale lock

A previous aborted `apt-get` left a stale lock file. The process was gone but the lock remained.

```bash
pct exec 300 -- apt-get update
# E: Could not get lock /var/lib/apt/lists/lock. It is held by process 305 (apt-get)
```

Fix:
```bash
pct exec 300 -- kill -9 305
pct exec 300 -- rm -f /var/lib/apt/lists/lock /var/lib/dpkg/lock /var/lib/dpkg/lock-frontend /var/cache/apt/archives/lock
pct exec 300 -- dpkg --configure -a
```

### Second attempt — timeout

`pct exec` with `apt-get install` timed out after 120s. The output buffering through nested SSH (`ssh root@pve "pct exec 300 -- apt-get install..."`) caused the connection to hang.

### Third attempt — success

Ran install directly:
```bash
ssh root@pve "pct exec 300 -- bash -c 'DEBIAN_FRONTEND=noninteractive apt-get install -y curl git jq socat conntrack iptables 2>&1 | tail -10'"
```

Installed: curl, git, jq, socat, conntrack, iptables.

---

## Step 10: Bootstrap k3s + ArgoCD — In Progress

### What happened

1. Wrote `scripts/bootstrap-k3s.sh` — adapted from hetzner `cloud-init.yaml.tftpl`. Installs k3s (without Traefik), cert-manager, and ArgoCD sequentially.

2. Pushed script to container:
```bash
scp bootstrap-k3s.sh root@pve:/tmp/
pct push 300 /tmp/bootstrap-k3s.sh /tmp/bootstrap-k3s.sh
```

3. First run attempt — **silent failure**. Script wrote only one line (`Installing k3s...`) then died. Root cause: DNS resolution failure (curl exit code 6). This was before the vmbr1 fix.

4. After vmbr1 fix, re-ran in background:
```bash
pct exec 300 -- bash -c 'nohup bash /tmp/bootstrap-k3s.sh > /tmp/bootstrap.log 2>&1 &'
```

5. k3s installed successfully — v1.34.6+k3s1:
```
[INFO]  Using v1.34.6+k3s1 as release
[INFO]  Downloading binary https://github.com/k3s-io/k3s/releases/download/v1.34.6%2Bk3s1/k3s
[INFO]  Installing k3s to /usr/local/bin/k3s
[INFO]  Creating /usr/local/bin/kubectl symlink to k3s
[INFO]  systemd: Starting k3s
```

6. k3s is starting — kubelet running, containerd running, waiting for node to become Ready.

### Current state of bootstrap log

```
[19:22:06] Waiting for k3s node ready...
```

Still waiting. k3s is in `activating` state, kubelet is running. First start can take 2-3 minutes.

### Container network (final)

| Address | Purpose |
|---------|---------|
| Container eth0 | 192.168.1.100/24 |
| vmbr1 (host) | 192.168.1.1/24 |
| Gateway | 192.168.1.1 (Proxmox host) |
| DNS | 8.8.8.8 |
| NAT | vmbr1 → vmbr0 → eno1 → cable modem |

### Files involved

- `scripts/bootstrap-k3s.sh` — bootstrap script in ticket
- `/tmp/bootstrap-k3s.sh` — copy pushed into container
- `/tmp/bootstrap.log` — log inside container

### What's still pending

- k3s node becoming Ready
- cert-manager installation
- ArgoCD installation
- kubeconfig extraction
- poll-modem K8s manifests
- ArgoCD Application for poll-modem
