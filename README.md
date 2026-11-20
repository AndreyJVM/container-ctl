<div align="center">

# Dockpulse

[![CI](https://github.com/AndreyJVM/dockpulse/actions/workflows/ci.yml/badge.svg)](https://github.com/AndreyJVM/dockpulse/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/AndreyJVM/dockpulse?color=blue&logo=github)](https://github.com/AndreyJVM/dockpulse/releases)
[![Go Version](https://img.shields.io/badge/Go-1.27.3+-00ADD8?logo=go)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/Platform-Linux-lightgrey)](https://github.com/AndreyJVM/dockpulse/releases)

**A fast, lightweight terminal TUI for Docker containers and resources.**<br>

[Contributing](CONTRIBUTING.md)

---

</div>

### Key Features

- **Docker Containers**:
  - Live CPU% and memory stats with dynamic color coding.
  - Interactive sorting by CPU%, RAM, Name, or Status (`s`, `o`).
  - Live Network I/O (RX/TX) and Block I/O (Read/Write) metrics in details panel and inspect modal.
  - Healthcheck statuses (`[healthy]`, `[unhealthy]`, `[starting]`).
  - Search & filter by name, image, ID, or health status (`/`).
  - Logs viewer with scroll support (`L`).
  - Detailed container inspection modal (`Enter` / `I`).
  - Embedded interactive terminal (`E` — `/bin/bash` or `/bin/sh`) without leaving Dockpulse, plus a full-screen fallback.
  - Power control: Start / Stop (`S`), Restart (`P`), Force Kill (`K` — `SIGKILL`).
  - Safe container removal with confirmation (`x`, `d`).
  - Docker System Prune modal (`C`) with disk reclaim estimates and confirmation.
  - Socket support: Docker daemon, Rootless Docker, and Podman (<2 ms latency).
- **Docker Images**:
  - Local image inventory with Image ID, Repository, Tag, Size, Created timestamp, and Digest.
  - Status indicators: `[In Use]`, `[Unused]`, and `[Dangling]` (`<none>:<none>`).
  - Search & filter by repository, tag, ID, or digest (`/`).
  - Selected image details panel (full ID, digest, all tags, container usage count).
  - Safe image removal with confirmation (`x`, `d`), including warning for in-use images.
- **Docker Compose**: Automatic project/service grouping, aggregated stack health, live stack logs (`L`), service control (`U` / `S` / `P`), and stack teardown (`D` / down) with confirmation.
- **Docker Networks**: Network driver, scope, connected container count, subnet, gateway, and Compose ownership.
- **Docker Volumes**: Volume driver, scope, reference count, disk usage, mountpoint, and Compose ownership.
- **Docker Swarm**: Cluster node roles (Leader, Manager, Worker), service listings, replica counts with degradation highlights, and live service logs (`L`). Dynamically shown when Swarm is active on the daemon (or forced with `--show-swarm`).
- **English Interface**: Consistent English UI, CLI help, diagnostics, and documentation.
- **Zero Dependencies**: Standalone binary (<12 MB) with zero external runtime dependencies. Runs on Linux, macOS, and FreeBSD.

### Installation

#### Docker (Zero Installation)

Run Dockpulse directly without installing any local dependencies:

```bash
docker run --rm -it -v /var/run/docker.sock:/var/run/docker.sock ghcr.io/andreyjvm/dockpulse:latest
```

#### Linux Packages (.deb / .rpm / .apk)

Pre-built packages for **amd64** and **arm64** are available on the [Releases](https://github.com/AndreyJVM/dockpulse/releases) page:

```bash
# Debian / Ubuntu / Linux Mint (.deb)
sudo dpkg -i dockpulse_0.13.0_amd64.deb

# RHEL / Fedora / Rocky Linux (.rpm)
sudo rpm -i dockpulse_0.13.0_amd64.rpm

# Alpine Linux (.apk)
apk add --allow-untrusted dockpulse_0.13.0_amd64.apk
```

#### Pre-built Binaries (Linux, macOS, FreeBSD)

Download the archive for your operating system and architecture from [Releases](https://github.com/AndreyJVM/dockpulse/releases):

```bash
# Example: Linux AMD64
tar -xzf dockpulse_0.13.0_linux_amd64.tar.gz
./dockpulse
```

#### Build from Source (Go 1.27+)

```bash
git clone https://github.com/AndreyJVM/dockpulse.git
cd dockpulse
make build
./dockpulse
```

### Command-line Options

```text
Usage:
  dockpulse [options]

Options:
  -i, --interval <duration>  Auto-refresh interval (default: 2s, env: DOCKPULSE_INTERVAL)
  -t, --tab <1-6|name>       Initial active tab (1: Docker, 2: Images, 3: Compose, 4: Networks, 5: Volumes, 6: Swarm) (default: 1, env: DOCKPULSE_TAB)
      --show-swarm           Always display Swarm tab even if inactive (env: DOCKPULSE_SHOW_SWARM)
  -v, --version              Show version information
  -h, --help                 Show help message
```

### Hotkeys

| Key | Action |
|---|---|
| `1`..`5` (or `6`) / `Tab` / `←`/`→` | Switch tabs (Docker, Images, Compose, Networks, Volumes, Swarm) |
| `r`, `R` | Force immediate refresh |
| `↑` / `↓` (or `k` / `j`) | Navigate rows / select items |
| `/` | Search & filter (`Esc` to clear) |
| `s`, `S`, `o`, `O` | Sort containers (CPU, Memory, Name, Status) |
| `L` | View logs (scroll with `↑`/`↓`/`PgUp`/`PgDn`) |
| `Enter` / `I` | Inspect container (details, mounts, network, ports, health) |
| `E` | Open or focus the embedded container terminal (`/bin/bash` or `/bin/sh`) |
| `Ctrl+]` | Return focus from the embedded terminal to Dockpulse |
| `Ctrl+D` | Exit the current shell session while the terminal is focused |
| `Ctrl+X` | Close an unfocused embedded terminal |
| `S` | Start / Stop container or Compose stack |
| `P` | Restart container or Compose stack |
| `K` | Force kill (`SIGKILL`) container |
| `x`, `d` | Remove container or image (with confirmation) |
| `U` | Start Compose stack |
| `D` | Compose down (`docker compose down`) with confirmation |
| `C` | Docker system prune (cleanup unused resources) |
| `?` | Toggle help / hotkey shortcuts modal |
| `q` / `Esc` / `Ctrl+C` | Quit application |

### Community & Governance

- **Contributing**: Please review [CONTRIBUTING.md](CONTRIBUTING.md) for development workflow, architectural guidelines, and test instructions.
- **Code of Conduct**: We follow the [Contributor Covenant](CODE_OF_CONDUCT.md) to ensure an open and welcoming community.
- **Security Policy**: Read [SECURITY.md](SECURITY.md) for instructions on reporting vulnerabilities responsibly.
- **License**: Dockpulse is released under the [MIT License](LICENSE).

---

### Dashboard Views

| Tab | View | Description |
|:---:|---|---|
| `1` | **Docker** | Live container stats, CPU & RAM bars, health badges, ports, and embedded shell (`E`) |
| `2` | **Images** | Local image inventory, digest, tags, sizes, usage status, and image pruning |
| `3` | **Compose** | Service orchestration grouping, live project logs (`L`), and stack control (`U`/`S`/`D`) |
| `4` | **Networks** | Network drivers, IPAM subnets, gateways, and attached container list |
| `5` | **Volumes** | Volume drivers, disk usage, mount points, and container references |
| `6` | **Swarm** *(adaptive)* | Cluster node inventory, roles, service replica distribution, and task logs |

> Detailed screenshots and visual guides are available in [docs/screenshots/](docs/screenshots/).

