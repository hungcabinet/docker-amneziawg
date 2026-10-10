# Panel

Web UI for the docker-amneziawg fork. `PANEL_SOURCE=mock` serves a fixed scene for local layout work. `live` reads `awg show all dump`, `/config/peer*` and host `ping`/`curl`.

Do not add a `.env` file here or in the repository root. Pass variables in the shell.

```powershell
$env:PANEL_PASSWORD = "change-me"
$env:PANEL_SOURCE = "mock"
go run ./cmd/panel
```

Then open `http://127.0.0.1:8080`.

| Variable | Default | Role |
|---|---|---|
| `PANEL_PASSWORD` | (none) | First-user password. Required only when the SQLite user does not exist yet. Later starts leave the stored hash alone. |
| `PANEL_USER` | `admin` | First-user login. |
| `PANEL_SOURCE` | `mock` | `mock` or `live`. |
| `PANEL_LISTEN` | `:8080` | HTTP bind address. The container defaults to `:8888`. |
| `PANEL_DB_PATH` | `data/panel.db` | SQLite file, relative to the working directory. |
| `PANEL_CONFIG_DIR` | `/config` | Peer directories and client `.conf` files. |
| `PANEL_DIRECT_INTERFACE` | `eth0` | Direct uplink shown on the exits page. |
| `PANEL_PING_TARGETS` | `1.1.1.1,8.8.8.8` | Default ping addresses. |
| `PANEL_PING_TARGETS_<IFACE>` | (none) | Per-interface override, e.g. `PANEL_PING_TARGETS_WG1`. |
