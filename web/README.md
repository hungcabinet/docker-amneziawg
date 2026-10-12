# Panel

Web UI bundled in this fork. In the container it starts as `live`: peers from `awg show all dump` and `/config/peer*`, exits from client tunnels plus the direct uplink, on-demand public IPv4 and IPv6 via `curl`.

An interface is the server or a client tunnel according to `/config/wg_confs/<name>.conf`: a conf with an `Endpoint` line is a tunnel. Interfaces without a conf there are guessed from the dump, which mistakes a server for a tunnel once its peers have connected.

Peers show the address they last connected from and tunnels the server they connect to, so an `[IPv6]:port` endpoint means that hop runs over IPv6.

Listen on **8888/tcp** (8080 is CoreDNS health). Map it on the host and set a first-run password.

```yaml
services:
  amneziawg:
    image: ghcr.io/hungcabinet/docker-amneziawg:latest
    ports:
      - "8888:8888/tcp"
    environment:
      - PANEL_PASSWORD=change-me
      # - PANEL_USER=admin
      # - PANEL_DIRECT_INTERFACE=eth0
      # - PANEL_PING_TARGETS=1.1.1.1,8.8.8.8
      # - PANEL_PING_TARGETS_WGUP1=9.9.9.9
```

Open `http://<host>:8888`. `PANEL_USER` / `PANEL_PASSWORD` create the first SQLite user only. Later restarts keep `/config/panel/panel.db`; changing login or password in the UI is not overwritten by env.

## Container defaults

`svc-panel` sets these if you omit them. The Go binary defaults (used by `go run`) are different — see Local debug.

| Variable | Container | Role |
|---|---|---|
| `PANEL_SOURCE` | `live` | `live` reads the host. `mock` is a fixed scene for layout work. |
| `PANEL_LISTEN` | `:8888` | HTTP bind address. |
| `PANEL_DB_PATH` | `/config/panel/panel.db` | SQLite file on the config volume. |
| `PANEL_CONFIG_DIR` | `/config` | Peer directories (`peer_*`) and client `.conf` files. |
| `PANEL_USER` | `admin` | First-user login. Ignored after the user exists. |
| `PANEL_PASSWORD` | (none) | First-user password. Required only on the first start. |
| `PANEL_DIRECT_INTERFACE` | `eth0` | Direct uplink on the exits page (`eth0@if44` is matched as `eth0`). |
| `PANEL_PING_TARGETS` | `1.1.1.1,8.8.8.8` | Default ping addresses. IPv6 literals work too, e.g. `1.1.1.1,2606:4700:4700::1111`; through a tunnel they need an IPv6 address and `::/0` in its `AllowedIPs`. |
| `PANEL_PING_TARGETS_<IFACE>` | (none) | Per-interface override. The suffix is the interface name in upper case, e.g. `PANEL_PING_TARGETS_WG0`. |

Do not put a `.env` in this `web/` tree or the repository root. Pass variables through Compose or `-e`.

## Local debug

Binary defaults when you run outside the container: `PANEL_SOURCE=mock`, `PANEL_LISTEN=:8080`, `PANEL_DB_PATH=data/panel.db`.

```powershell
$env:PANEL_PASSWORD = "change-me"
$env:PANEL_SOURCE = "mock"
go run ./cmd/panel
```

Then open `http://127.0.0.1:8080`. Use `PANEL_SOURCE=live` only on a host that has `awg`, `ip`, `ping`, `curl`, and a real `/config`.
