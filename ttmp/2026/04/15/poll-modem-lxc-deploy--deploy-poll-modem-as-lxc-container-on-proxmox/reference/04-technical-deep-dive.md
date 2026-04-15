---
title: "ARTICLE - Deploying k3s on Proxmox - A Technical Deep Dive"
aliases:
  - k3s proxmox deep dive
  - proxmox kubernetes homelab
  - cloud-init proxmox vm
tags:
  - article
  - k3s
  - proxmox
  - kubernetes
  - cloud-init
  - tailscale
  - homelab
  - lxc
status: active
type: article
created: 2026-04-15
repo: /home/manuel/code/wesen/corporate-headquarters/poll-modem
---

# ARTICLE - Deploying k3s on Proxmox: A Technical Deep Dive

What happens when you try to run Kubernetes on a home server behind a cable modem? This article traces the full journey — from a simple monitoring tool to a production-grade cluster — documenting every dead end, every wrong assumption, and every lesson learned along the way.

> [!summary]
> 1. **LXC containers look like VMs but aren't** — k3s expects a real kernel; fighting LXC restrictions costs more time than just using a VM
> 2. **Cable modems are hostile to virtual NICs** — your carefully bridged Proxmox network may not work the way you expect
> 3. **Cloud-init is the right abstraction** — once you get it working, a single YAML file replaces an hour of manual SSH commands
> 4. **Tailscale is the missing network layer** — it solves the "cable modem doesn't route to my VMs" problem entirely

## Why this note exists

I spent a full day deploying k3s on a Proxmox home server. I tried LXC containers, QEMU VMs, three different network configurations, and two cloud image formats before arriving at a setup that actually works. This article captures the full technical narrative so that the next person (or future me) can skip the dead ends.

## The starting point

The goal was simple: deploy `poll-modem`, a Go TUI that scrapes a cable modem's web interface and stores channel quality data in SQLite. The tool was already working — tested against a real Technicolor CGM4331COM on Cox, showing that channels 27 and 37 had millions of uncorrectable errors while the rest were clean.

The deployment target was a Proxmox server at home, sitting behind that same cable modem. The Proxmox box is a physical machine running Proxmox 8.1.4 on Debian, with one NIC bridged as `vmbr0` at `192.168.0.227/24`.

## Dead End #1: LXC with the binary directly

The first plan was the simplest: create an LXC container, install Go, compile poll-modem inside it, run it in tmux. LXC containers are lightweight, Proxmox manages them natively, and they boot in seconds.

This actually worked fine for running the binary. But then the scope expanded — the goal became running k3s with ArgoCD for a proper GitOps platform. That's where LXC stopped being simple.

## Dead End #2: k3s inside LXC

k3s is designed to run on a full Linux kernel. Inside an LXC container, you share the host's kernel but with restricted access. Here's what broke, in order:

**`modprobe overlay` failed.** k3s needs the overlay filesystem module for container images. Inside LXC, `modprobe` can't load kernel modules — it gets `FATAL: Module overlay not found in directory /lib/modules/...`. The fix is to load modules on the Proxmox host instead. That works for `overlay` but not for everything k3s needs.

**`/dev/kmsg` missing.** kubelet reads `/dev/kmsg` (the kernel message buffer) to monitor OOM kills and other kernel events. LXC doesn't expose this device by default. The fix is `mknod /dev/kmsg c 1 11` inside the container — but it has to be done on every boot.

**`/proc/sys` read-only.** kubelet writes to `/proc/sys/kernel/panic`, `/proc/sys/kernel/panic_on_oops`, and `/proc/sys/vm/overcommit_memory`. Inside LXC, `/proc/sys` is mounted read-only by default. The fix is `lxc.mount.auto: proc:rw sys:rw` in the container config, which weakens isolation significantly.

**AppArmor confinement.** k3s needs capabilities that LXC's AppArmor profile blocks. The fix is `lxc.apparmor.profile: unconfined` — essentially disabling AppArmor for the container.

**`nf_conntrack_max` permission denied.** Even with writable `/proc/sys`, some sysctls are namespace-restricted. The container can't set connection tracking limits.

Each fix was a layer of hacks. After enabling unconfined AppArmor, writable proc/sys, device access, and a mknod for /dev/kmsg, the container was barely contained anymore — it was a VM with extra steps.

**The lesson:** k3s in LXC works after enough hacks, but you've defeated the purpose of containerization. Just use a VM.

## Dead End #3: QEMU VM with Debian cloud image

Having abandoned LXC, I created a proper QEMU VM with the Debian 12 cloud image (`debian-12-cloud.qcow2` imported into Proxmox). This should have been the answer. Instead:

```
Error: Could not retrieve NBP file size from HTTP server.
Error: Server response timeout.
BdsDxe: failed to load Boot0004 "UEFI HTTPv4"
```

The VM tried to PXE boot instead of booting from the imported disk. The serial console showed no useful output. The QEMU guest agent wasn't installed (chicken-and-egg — you need the VM booted to install it). SSH never came up.

I spent time trying to access the serial console via `socat` through the Proxmox host, which sort of worked but only showed the PXE boot error, not a running system.

**Root cause:** The Debian cloud image expected a specific cloud-init nocloud datasource that wasn't properly configured. The Proxmox cloud-init ISO (ide2) was attached but the OVMF firmware was trying network boot first.

**The lesson:** Not all cloud images work equally well with Proxmox. Ubuntu cloud images have better Proxmox integration than Debian's.

## What worked: Ubuntu Noble cloud image

The Proxmox server already had `noble-server-cloudimg-amd64.img` in `/var/lib/vz/template/iso/`. One `qm importdisk` later, the VM booted, cloud-init ran, SSH keys were injected, and I had a working Ubuntu machine with DHCP in about 60 seconds.

No kernel module hacks. No `/dev/kmsg` mknod. No AppArmor workarounds. No serial console debugging. It just worked.

```bash
qm create 301 --name k3s-server \
  --memory 8192 --cores 4 --cpu host \
  --net0 virtio,bridge=vmbr0 \
  --bios ovmf --machine q35 --agent enabled=1

qm importdisk 301 /var/lib/vz/template/iso/noble-server-cloudimg-amd64.img local-lvm

qm set 301 \
  --scsihw virtio-scsi-pci --scsi0 local-lvm:vm-301-disk-0 \
  --efidisk0 local-lvm:1,efitype=4m,pre-enrolled-keys=0 \
  --ide2 local-lvm:cloudinit \
  --boot order=scsi0 \
  --serial0 socket --vga serial0 \
  --ciuser ubuntu --sshkeys /root/.ssh/authorized_keys \
  --ipconfig0 ip=dhcp

qm resize 301 scsi0 30G
qm start 301
```

Three minutes later: SSH in, run the k3s bootstrap script, and the cluster is ready.

## The network problem nobody warns you about

Here's the thing about running VMs behind a cable modem: **the cable modem's DHCP server sees virtual MAC addresses, and it may not treat them the same as real devices.**

In my setup, the Proxmox host has one physical NIC (`eno1`) bridged as `vmbr0` at `192.168.0.227`. VMs and LXC containers on vmbr0 get virtual MAC addresses (`bc:24:11:*` from Proxmox). The cable modem at `192.168.0.1` is the gateway and DHCP server.

What I discovered:

- **VMs on vmbr0**: Get DHCP leases from the cable modem. Can reach the gateway and the internet. Can be reached from the Proxmox host. Cannot be reached from my dev machine on the same LAN (the cable modem apparently doesn't bridge traffic between the physical NIC's MAC and virtual MACs at L3).

- **LXC containers on vmbr0**: Don't get DHCP from the cable modem. Can reach the Proxmox host but not the gateway. The container's virtual MAC is invisible to the cable modem at L2 — pings to the gateway go into the void.

- **LXC containers on vmbr1**: Work with NAT through the Proxmox host. Can reach the internet. Can't reach the cable modem's management interface directly (which is on vmbr0's network).

The **stale ARP problem** compounded this: every time I destroyed and recreated a container or VM on the same IP, the Proxmox host's ARP cache retained the old MAC address. Symptom: the host can't reach the new container until `ip neigh flush <ip>` clears the stale entry.

**The solution:** Tailscale. Install it on the VM, join the tailnet, and access everything through the Tailscale network. The cable modem's L2 restrictions become irrelevant.

## Cloud-init: the right abstraction

Once the VM approach worked manually, the next step was automating it. Proxmox supports custom cloud-init via `--cicustom user=local:snippets/<file>.yaml`. The cloud-init template does everything the manual bootstrap did:

```yaml
#cloud-config
package_update: true
package_upgrade: true
packages:
  - ca-certificates
  - curl
  - git
  - jq
  - qemu-guest-agent

write_files:
  - path: /etc/rancher/k3s/config.yaml
    content: |
      write-kubeconfig-mode: "0644"
      disable:
        - traefik
      tls-san:
        - k3s-server
        - k3s-server.tail879302.ts.net

runcmd:
  - [/usr/local/bin/bootstrap-k3s.sh]
```

### The `--cicustom` trap

Proxmox has built-in cloud-init support with flags like `--sshkeys`, `--ciuser`, `--ipconfig0`. When you add `--cicustom user=...`, the custom user-data **overrides** the built-in user-data. This means SSH keys specified with `--sshkeys` are silently ignored — the VM boots with an empty `~/.ssh/authorized_keys`.

The fix is to include `ssh_authorized_keys:` explicitly in the custom cloud-init YAML. This caught me during validation: the VM booted, k3s was running, ArgoCD was ready, but SSH was locked out.

### The RSA key problem

The Proxmox host only has an RSA SSH key (`id_rsa`). Ubuntu Noble (24.04) disabled RSA SHA-1 authentication by default. Even with the RSA public key in `authorized_keys`, SSH connections from the Proxmox host fail with "Permission denied (publickey)".

The fix: either generate an ed25519 key on the Proxmox host, or add `PubkeyAcceptedAlgorithms +ssh-rsa` to the VM's sshd config. For the cloud-init template, I included both the ed25519 key (for direct Tailscale access from my dev machine) and the RSA key (for Proxmox host access during bootstrap).

### The TLS SAN problem

k3s generates a TLS certificate for the API server on first boot. The certificate includes SANs for `127.0.0.1`, `localhost`, and the node's hostname. When you access the API from a different hostname (like the Tailscale DNS name), kubectl fails with:

```
tls: failed to verify certificate: x509: certificate is valid for k3s-server, not k3s-proxmox
```

The fix is to pre-declare all hostnames in the k3s config before first boot. The cloud-init template includes:

```yaml
tls-san:
  - k3s-server
  - k3s-server.tail879302.ts.net
  - k3s-proxmox
  - k3s-proxmox.tail879302.ts.net
```

This is fragile — if you rename the machine in Tailscale, you need to update the config and restart k3s to regenerate the certificate.

## The Tailscale layer

Tailscale solves three problems at once:

1. **Direct access from dev machine** — The cable modem won't route to VM MACs, but Tailscale creates a separate overlay network. SSH and kubectl go through Tailscale, bypassing the cable modem entirely.

2. **MagicDNS** — Machines on the tailnet get automatic DNS names (`k3s-proxmox`, `k3s-server.tail879302.ts.net`). No need to remember IPs or maintain DNS records for internal access.

3. **Funnel for public access** — Tailscale Funnel can expose specific ports publicly through Tailscale's edge network, without opening cable modem ports. This is the plan for `*.crib.scapegoat.dev`.

The cloud-init installs `tailscaled` but intentionally does not run `tailscale up`. The hetzner k3s template does the same — auth keys should not be embedded in cloud-init user-data. The join step is manual (or driven by a secret manager).

## What the hetzner reference taught me

A previous project (`~/code/wesen/2026-03-27--hetzner-k3s/`) had already solved the k3s + ArgoCD + GitOps pattern on Hetzner Cloud. The architecture was:

```
Terraform → Hetzner VM + firewall
cloud-init → K3s + ArgoCD bootstrap
GitOps repo → ArgoCD Applications + Kustomize packages
ArgoCD → cluster reconciliation
```

The key differences between Hetzner and Proxmox:

| Aspect | Hetzner | Proxmox (home) |
|--------|---------|-----------------|
| Public IP | ✅ Yes | ❌ Behind cable modem |
| Terraform | ✅ `hcloud` provider | ❌ Manual `qm create` |
| Cloud image | Ubuntu from Hetzner CDN | Manual upload |
| SSH access | Direct via public IP | Via Tailscale only |
| TLS/Ingress | Let's Encrypt + public DNS | Tailscale Funnel + custom DNS |
| Cost | ~€5/month | Free (own hardware) |

The Proxmox setup reuses the same cloud-init pattern but replaces Terraform with a shell script and replaces public IP access with Tailscale.

## Common failure modes

### Stale ARP after recreating VMs/containers

Destroying and recreating a VM or container on the same IP leaves a stale ARP entry on the Proxmox host. The new device has a different MAC but the host still tries the old one.

**Fix:** `ip neigh flush <ip>` on the Proxmox host after recreating.

### Cloud-init SSH keys ignored with --cicustom

Using `--cicustom` overrides the built-in Proxmox cloud-init, including SSH key injection.

**Fix:** Add `ssh_authorized_keys:` to the custom cloud-init YAML.

### k3s TLS certificate doesn't include Tailscale hostname

The k3s API certificate is generated on first boot and only includes the hostname and IPs known at that time. Tailscale hostnames are not known until after Tailscale is configured.

**Fix:** Pre-declare all possible Tailscale hostnames in `tls-san` in the k3s config before first boot.

### Cloud image won't boot (PXE loop)

Some cloud images don't boot properly with Proxmox's OVMF firmware, falling back to PXE boot.

**Fix:** Use Ubuntu Noble cloud images, which have proven Proxmox compatibility. Ensure `--boot order=scsi0` is set correctly.

### apt-get locks from aborted SSH sessions

Aborted SSH sessions can leave `apt-get` holding dpkg locks. The process is gone but the lock files remain.

**Fix:** `rm -f /var/lib/apt/lists/lock /var/lib/dpkg/lock* /var/cache/apt/archives/lock && dpkg --configure -a`

## Working rules

1. **Use VMs, not LXC, for k3s.** The kernel module and sysctl hacks required for k3s in LXC defeat the purpose of containerization.

2. **Always include Tailscale SANs in the k3s config.** You can't add them after first boot without restarting k3s and regenerating certificates.

3. **Never put Tailscale auth keys in cloud-init.** Use a secret manager or manual join.

4. **Flush ARP after recreating VMs on the same IP.** `ip neigh flush <ip>` on the Proxmox host.

5. **Include all SSH keys in custom cloud-init.** `--cicustom` overrides `--sshkeys`.

6. **Use Ubuntu Noble cloud images with Proxmox.** Better compatibility than Debian cloud images.

7. **Test the cloud-init template by destroying and recreating.** The validation cycle is: destroy → create → wait 3 min → check /etc/motd.

## Related notes

- [[PROJ - poll-modem k3s Cluster on Proxmox]] — project note for the cluster setup
- Hetzner k3s reference: `/home/manuel/code/wesen/2026-03-27--hetzner-k3s/`
- DNS Terraform: `/home/manuel/code/wesen/terraform/dns/zones/scapegoat-dev/envs/prod/`
