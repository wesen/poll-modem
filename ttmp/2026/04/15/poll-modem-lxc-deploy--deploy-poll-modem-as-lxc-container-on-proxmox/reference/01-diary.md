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
