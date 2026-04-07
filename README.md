# rsm — Reforger Server Manager

Create and manage Arma Reforger dedicated servers on Linux with a simple CLI tool.

## Features

- Install and update stable or experimental server
- Configure systemd service to enable autostart
- Manage, edit and switch server configurations including mods and savegames

## Requirements

- Linux with systemd
- [steamcmd](https://developer.valvesoftware.com/wiki/SteamCMD)
- `sudo` access (for writing systemd unit files)

---

## Installation

Download the latest binary from [Releases](https://github.com/yakovlev-alex/reforger-server-manager/releases):

```bash
curl -L https://github.com/yakovlev-alex/reforger-server-manager/releases/latest/download/rsm-linux-amd64 -o rsm
chmod +x rsm
sudo mv rsm /usr/local/bin/rsm
```

---

## Quick start

Run the guided setup wizard. It creates an instance, downloads server files via steamcmd, installs the systemd unit, and optionally starts the server.

```bash
rsm init
```

The wizard will ask for:

1. Instance name (default: `main`)
2. Installation directory (default: `<CWD>/main`)
3. OS user to run the server as (default: current user)
4. Max FPS (default: `60`)
5. Extra launch flags (pre-selects `-loadSessionSave`, `-backendLocalStorage`)
6. Periodic automatic restarts (disabled by default; choose 6h, 12h, 1d, 2d or custom)
7. Stable or experimental server build
8. Server configuration (name, ports, max players, admin password, scenario)
9. Whether to download server files now
10. Whether to install the systemd unit and enable autostart

At the end you will have Arma Reforger Server ready to run.

Then run `rsm` to see instance status and common commands, `rsm start` to start the server manually or `rsm enable` to enable autostart. 

> See [official documentation](https://community.bistudio.com/wiki/Arma_Reforger:Server_Hosting) for installation details.

---

## Concepts

**Instance** — one server installation (the `ArmaReforgerServer` binary and its data files). Instance metadata is stored in `rsm.yaml` inside the installation directory.

**Configuration** — a named `config.json` + `profile/` directory pair. One instance can have multiple configurations; exactly one is active at a time.

```
<install_dir>/
  rsm.yaml                        # instance metadata
  ArmaReforgerServer              # game binary (installed by steamcmd)
  configuration/
    vanilla/
      config.json
      profile/
    modded/
      config.json
      profile/
```

The systemd unit is installed at `/etc/systemd/system/rsm-<name>.service`. When periodic restarts are enabled, two additional units are installed: `rsm-<name>-restart.timer` and `rsm-<name>-restart.service`.

---

## Commands

### `rsm init`

Guided first-time setup. Creates an instance, optionally downloads server files, installs the systemd unit, and starts the server.

```bash
rsm init
rsm init myserver
rsm init --experimental    # use the experimental server build
```

---

### `rsm config new`

Create a named server configuration. If no configuration exists yet, runs a full wizard. If one already exists, clones the active configuration.

```bash
rsm config new
rsm config new modded
```

**Wizard prompts** (first configuration only):

| Prompt | Default |
|---|---|
| Configuration name | `default` |
| Server display name | `Reforger Server — <name>` |
| Bind IP address | `0.0.0.0` |
| Public IP address | *(blank — auto-detected)* |
| Game port (UDP) | `2001` |
| Steam query port (UDP) | `17777` |
| Max players | `64` |
| Admin password | *(required)* |
| Private server / game password | *(optional)* |
| Scenario | Everon Game Master |

You are offered the option to open `config.json` in `$EDITOR` at the end.

---

### `rsm config list`

List all configurations, showing which is active.

```bash
rsm config list
```

---

### `rsm config edit`

Open a configuration's `config.json` in `$EDITOR`. Defaults to the active configuration.

```bash
rsm config edit           # edit active config
rsm config edit modded    # edit a named config
```

`$EDITOR` is used if set; otherwise falls back to `$VISUAL`, then `nano`, `vi`, `vim`.

---

### `rsm config use`

Switch the active configuration. Prompts to restart the server if it is currently running.

```bash
rsm config use modded
rsm config use vanilla
```

---

### `rsm config delete`

Delete a named configuration and its profile directory. The active configuration cannot be deleted — switch away first.

```bash
rsm config delete modded
```

---

### `rsm edit`

Interactively edit instance-level settings. Presents a multi-select menu; only the fields you choose are changed.

```bash
rsm edit
```

**Editable options:**

| Option | Description |
|---|---|
| Max FPS | Server tick rate passed as `-maxFPS` |
| Extra launch flags | Multi-select from known flags; custom flags are preserved |
| Periodic restart | Enable/disable and set the restart interval (e.g. `6h`, `12h`, `1d`) |
| Update on restart | Toggle steamcmd update before each restart |
| Experimental branch | Switch between stable and experimental builds |
| Systemd service user | OS user the server process runs as |

After saving, the systemd unit (and restart timer if applicable) are reinstalled automatically. If the server is running, a `rsm restart` hint is printed.

---

### `rsm start` / `rsm stop` / `rsm restart`

Control the server via systemd. `rsm start` and `rsm restart` automatically install the systemd unit if it is not already present (requires sudo).

```bash
rsm start
rsm stop
rsm restart
```

---

### `rsm enable` / `rsm disable`

Enable or disable autostart on boot.

```bash
rsm enable
rsm disable
```

---

### `rsm status`

Show instance details and the live systemd service status. Includes the periodic restart interval and timer state when configured.

```bash
rsm status
```

---

### `rsm logs`

Stream server logs from journalctl.

```bash
rsm logs              # last 50 lines
rsm logs -f           # follow in real time
rsm logs -n 200       # last 200 lines
rsm logs -f -n 0      # follow all from the beginning
```

---

### `rsm install`

Install or verify server files via steamcmd.

```bash
rsm install                  # install / verify
rsm install --experimental   # use the experimental build (this run only)
```

---

### `rsm update`

Update server files via steamcmd.

- **Server is stopped** — runs the update immediately.
- **Server is running** — schedules the update for the next restart. Run `rsm restart` to apply.

```bash
rsm update      # update now or schedule
rsm restart     # apply a scheduled update and restart
```

---

### `rsm service`

Manage the systemd unit directly.

```bash
rsm service install    # (re)generate and install the unit file
rsm service status     # show the rendered unit file and install/enable state
rsm service enable     # same as rsm enable
rsm service disable    # same as rsm disable
```

Use `rsm edit` to change instance settings (max FPS, extra flags, periodic restart, etc.) — it reinstalls the unit automatically. Use `rsm service install` to force-reinstall the unit without changing any settings.

---

### `rsm delete`

Remove an instance from the registry and optionally delete all server files.

```bash
rsm delete
```

---

## Mods

Mods are configured by editing `config.json` directly:

```bash
rsm config edit
```

Add entries to the `game.mods` array:

```json
"game": {
  "mods": [
    { "modId": "596573", "name": "My Mod", "version": "1.0.0" }
  ]
}
```

The `version` field is optional. Mod files are downloaded by the server on startup.

---

## Experimental build

The experimental Arma Reforger server is a separate Steam application (App ID `1890870`) from the stable build (App ID `1874900`). Select it during `rsm init` or with the `--experimental` flag:

```bash
rsm init --experimental
```

The choice is stored in `rsm.yaml` and applies to all future `rsm install` and `rsm update` runs. It can be overridden for a single install with `rsm install --experimental`.

---

## Instance resolution

Every command automatically resolves the target instance without needing a flag:

1. Walk up from the current directory looking for `rsm.yaml`
2. Check if the current directory is inside a registered instance
3. If only one instance is registered, use it automatically
4. Otherwise, `cd` into the instance directory and re-run the command

---

## Systemd service name

Services are named `rsm-<instance-name>.service`. For an instance named `main`:

```bash
systemctl status rsm-main.service
journalctl -u rsm-main.service -f
```

When periodic restarts are enabled, a timer and its companion one-shot service are also installed:

```bash
systemctl status rsm-main-restart.timer    # check timer state and next trigger
systemctl list-timers rsm-main-restart.timer
```
