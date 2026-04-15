---
Title: Session Assessment
Ticket: poll-modem-lxc-deploy
DocType: reference
Status: active
Topics:
  - deployment
  - assessment
  - proxmox
  - k3s
---

# Session Assessment

## Current Situation

### What exists and works

1. **`../poll-modem/` standalone repo** — Clean extraction from go-go-labs. Builds, lints, releases (deb/rpm/tar.gz). 4 git commits. This is solid.

2. **poll-modem tested against real modem** — Successfully ran in tmux on the dev machine, connected to cable modem at 192.168.0.1 with credentials from 1Password ("coxwifi"). Got real data: 33/34 downstream channels locked, channels 27 and 37 showing massive uncorrectable errors.

3. **Docmgr ticket + diary** — Ticket `poll-modem-lxc-deploy` exists in `../poll-modem/ttmp/` with a long diary and scripts directory.

4. **Proxmox server** — `root@pve` accessible via SSH. Proxmox 8.1.4. Debian 12 LXC template downloaded. Network: `vmbr0` at `192.168.0.227/24`, gateway `192.168.0.1`.

5. **LXC container 300** — Currently running on Proxmox:
   - Hostname: `k3s-server`
   - IP: `192.168.0.210/24`
   - 4 cores, 8GB RAM, 30GB disk
   - Privileged (required for k3s)
   - Features: nesting, keyctl, fuse enabled
   - SSH active, root password set (`changeme`)
   - **No packages installed yet** (curl, git missing)
   - **k3s NOT installed**
   - **ArgoCD NOT installed**

6. **Reference project** — `/home/manuel/code/wesen/2026-03-27--hetzner-k3s/` has a proven cloud-init + k3s + ArgoCD bootstrap pattern that works on Hetzner.

### What's broken or messy

1. **Stale ARP problem** — Every time container 300 is destroyed and recreated on the same IP, the Proxmox host's ARP cache has a stale MAC entry. `ip neigh flush 192.168.0.210` fixes it, but this bit us twice. The VM attempt (same ID 300, same IP) left a stale entry that carried over to the LXC.

2. **Scripts are stale** — The scripts in `ttmp/.../scripts/` were written for the original LXC-native-poll-modem plan (install-go.sh, install-poll-modem.sh, run-in-tmux.sh). They're now irrelevant since we pivoted to k3s + ArgoCD. Only `create-lxc-container.sh` is close to useful, and even it has wrong specs (512MB RAM, DHCP).

3. **Diary is a dump, not a document** — The diary grew to ~500 lines of append-only text including three "appendix" sections (LXC vs Docker, Docker on Proxmox, k3s on Proxmox) that are reference material, not diary entries. The step numbering became incoherent when we pivoted from LXC-native to VM to LXC-with-k3s.

4. **VM attempt wasted time** — We spent significant time trying to boot a Debian cloud image as a QEMU VM (ID 300). It never booted properly (PXE boot loop, no serial console output, no SSH). The root cause was likely a mismatch between the cloud image expectations and our Proxmox cloud-init config. We destroyed it and went to LXC instead.

5. **No gw= parameter on the container** — The container was created with `ip=192.168.0.210/24` but the gateway wasn't added to `ipconfig0` in the second creation. The route exists (`default via 192.168.0.1 dev eth0 onlink`) but it came from DHCP or was auto-derived — should be explicit.

6. **Container is empty** — Container 300 is running but has no curl, no k3s, no ArgoCD. We tried `apt-get install` twice but it timed out/was aborted.

## What's Good

- The repo extraction was clean and methodical
- The Proxmox server is well set up (vmbr0 bridged to eno1, LXC templates available)
- The hetzner-k3s reference gives us a proven cloud-init pattern to adapt
- The container (300) has the right specs for k3s (privileged, nesting, keyctl, fuse)
- poll-modem itself works and produces real data

## What's Bad

- **Scope creep without updating artifacts** — We went from "LXC native poll-modem" → "VM with k3s" → "LXC with k3s + ArgoCD" without rewriting the scripts or cleaning the diary
- **No cloud-init for LXC** — We tried `--cicustom` but it only works for VMs. For LXC, we need to either use `pct exec` or write a snippet that Proxmox's cloud-init for LXC supports
- **Too many SSH hops** — Earlier attempts used nested SSH (`ssh root@pve "ssh root@192.168.0.210 ..."`) instead of `pct exec 300 -- ...`. User correctly called this out
- **Impatient package installs** — `apt-get install` was aborted/timed out twice. We never waited for it to finish
- **Diary appendices are noise** — They're reference material that should be separate docs, not diary steps

## Overall Approach Going Forward

### The proven pattern (from hetzner-k3s)

The hetzner reference does this cleanly:

1. **Terraform** creates the VM (we don't need Terraform for Proxmox LXC)
2. **cloud-init** runs a bootstrap script that installs k3s + cert-manager + ArgoCD
3. **ArgoCD** reconciles everything else from a GitOps repo

For Proxmox LXC, we should:

1. **Keep container 300** — it has the right specs, just needs packages and k3s
2. **Write one clean bootstrap script** — adapted from `cloud-init.yaml.tftpl`, run via `pct exec 300 -- bash bootstrap.sh`
3. **Install k3s + ArgoCD** — follow the hetzner pattern but simpler (no Terraform, no Tailscale, no Docker build step)
4. **Create a GitOps directory** in the poll-modem repo with ArgoCD Application manifests for poll-modem
5. **Build and push a container image** — either locally into k3s or via GHCR

### Specific next steps (in order)

1. **Flush ARP**, verify `pct exec 300` works
2. **Install packages**: `curl git jq socat conntrack iptables`
3. **Write and run bootstrap script** inside the container:
   - Install k3s (server, disable traefik)
   - Wait for k3s ready
   - Install cert-manager
   - Install ArgoCD
4. **Get kubeconfig** — `pct exec 300 -- cat /etc/rancher/k3s/k3s.yaml`
5. **Write K8s manifests for poll-modem** — Deployment, Service, PVC, Secret for credentials
6. **Create ArgoCD Application** to reconcile from the poll-modem repo
7. **Clean up the ticket** — rewrite diary, remove stale scripts, update tasks

### What to delete

- `scripts/create-lxc-container.sh` — specs are wrong (512MB, DHCP)
- `scripts/install-go.sh` — we don't need Go in the container, we need a Docker image
- `scripts/install-poll-modem.sh` — same reason
- `scripts/run-in-tmux.sh` — not relevant for k8s deployment
- `scripts/deploy.sh` — orchestrates the above, all wrong now
- `scripts/poll-modem.service` — not relevant for k8s
- All three diary appendices — move to separate reference docs or remove

### Container image strategy

poll-modem uses CGO (go-sqlite3), so we need `CGO_ENABLED=1` and a proper build chain. Options:

- **Build in Dockerfile** with `golang:1.23` builder stage, copy binary to `debian:12-slim`
- **Use GoReleaser** — already configured, can push to GHCR
- **Import into k3s** — `docker save` + `k3s ctr images import` (hetzner pattern)

Best path: **GoReleaser → GHCR → ArgoCD pulls from GHCR**. Same as hetzner pattern.
