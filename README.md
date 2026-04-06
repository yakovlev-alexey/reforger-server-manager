# rsm — Reforger Server Manager

A CLI utility for managing Arma Reforger dedicated server instances on Linux.

## Features

- **Instance management** — create and manage multiple named server installations
- **Configuration management** — each instance supports multiple named configurations (config.json + profile directory pairs)
- **steamcmd integration** — install, verify, and update server files; schedule updates on restart
- **systemd integration** — generate and install systemd service units; enable/disable autostart
- **Log streaming** — tail server logs via `journalctl`
- **Interactive wizards** — guided setup for instances and configurations with sensible defaults

## Concepts

```
Instance = one server installation (ArmaReforgerServer binary)
  └── Configuration = named config.json + profile/ directory pair
        ├── vanilla/config.json + vanilla/profile/
        └── modded/config.json  + modded/profile/
```

One instance has exactly one **active configuration** at a time. Switching configurations restarts the server.

## Installation

### Build from source

```bash
git clone https://github.com/yakovlev-alex/reforger-server-manager
cd reforger-server-manager
make build
sudo cp dist/rsm /usr/local/bin/rsm
```

Or install directly to `$GOPATH/bin`:

```bash
make install
```

## Quick Start

```bash
# 1. First-time setup (detects steamcmd, creates first instance)
rsm init

# 2. Or create an instance manually
rsm instance new main

# 3. Start the server
rsm start

# 4. Follow logs
rsm logs -f

# 5. Check status
rsm status
```

## Commands

### Global flags

| Flag | Short | Description |
|---|---|---|
| `--instance <name>` | `-i` | Target instance (auto-resolved if only one exists) |

---

### `rsm init`

First-time setup. Detects or prompts for steamcmd path, saves to `~/.config/rsm/config.yaml`, and optionally creates the first instance.

---

### `rsm instance`

| Command | Description |
|---|---|
| `rsm instance new [name]` | Create a new instance (wizard) |
| `rsm instance list` | List all instances with status |
| `rsm instance status [name]` | Detailed status of an instance |
| `rsm instance delete <name>` | Delete an instance |

---

### `rsm config`

| Command | Description |
|---|---|
| `rsm config new [name]` | Create a new named configuration (wizard) |
| `rsm config list` | List configurations, showing which is active |
| `rsm config edit <name>` | Open config.json in `$EDITOR` |
| `rsm config use <name>` | Switch active configuration (restarts if running) |
| `rsm config delete <name>` | Delete a named configuration |

---

### Server lifecycle

| Command | Description |
|---|---|
| `rsm start` | Start the server via systemd |
| `rsm stop` | Stop the server |
| `rsm restart` | Restart (runs steamcmd update first if scheduled) |
| `rsm enable` | Enable autostart on boot |
| `rsm disable` | Disable autostart |
| `rsm status` | Show systemd status |
| `rsm logs [-f] [-n N]` | Stream logs from journalctl |

---

### Steam updates

| Command | Description |
|---|---|
| `rsm install` | Install/verify server files via steamcmd |
| `rsm update` | Schedule update on next restart (or update immediately if stopped) |

---

## Directory Layout

```
~/.config/rsm/
├── config.yaml                        # Global rsm settings (steamcmd path)
└── instances/
    └── <instance-name>/
        ├── instance.yaml              # Instance metadata
        ├── service.unit               # Generated systemd unit (reference copy)
        └── configs/
            ├── vanilla/
            │   ├── config.json        # Arma Reforger server config
            │   └── profile/           # Server profile directory
            └── modded/
                ├── config.json
                └── profile/
```

systemd unit is installed at: `/etc/systemd/system/rsm-<instance-name>.service`

---

## Configuration Wizard

`rsm config new` prompts for:

- Server display name
- Bind IP / public IP
- Game port (default: `2001`)
- Steam query port (default: `17777`)
- Max players (default: `64`)
- Admin password
- Game password (optional)
- Scenario (default: Everon Game Master)

At the end, you are offered to open the generated `config.json` in `$EDITOR` for further customization. The `game.mods` array is included but empty — add mod entries manually.

### Mod entry format

```json
{
  "game": {
    "mods": [
      { "modId": "596573", "name": "My Mod" }
    ]
  }
}
```

---

## systemd service name

Services are named `rsm-<instance-name>.service`. For an instance named `main`:

```bash
systemctl status rsm-main.service
journalctl -u rsm-main.service -f
```

---

## Switching configurations

```bash
# Create a second configuration
rsm config new modded -i main

# Switch to it (restarts server if running)
rsm config use modded -i main

# Switch back
rsm config use vanilla -i main
```

---

## Update on restart

```bash
# Schedule a steamcmd update for next restart
rsm update -i main

# Apply immediately (restarts the server)
rsm restart -i main
```

---

## Requirements

- Linux (systemd)
- [steamcmd](https://developer.valvesoftware.com/wiki/SteamCMD) (for install/update)
- `sudo` access (for writing to `/etc/systemd/system/`)
