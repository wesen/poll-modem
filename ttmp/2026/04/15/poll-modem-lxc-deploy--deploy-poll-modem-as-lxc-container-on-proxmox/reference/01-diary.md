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
