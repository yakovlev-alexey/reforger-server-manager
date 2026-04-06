# AGENTS.md — Development Guide for AI Agents

This file is the authoritative reference for any AI agent working on this codebase.
Read it fully before making any changes.

---

## Project overview

`rsm` is a Linux CLI tool that manages Arma Reforger dedicated server instances.
It is a single statically-linked Go binary.

**Core concepts:**

- **Instance** — one server installation. Metadata lives in `<install_dir>/rsm.yaml`.
- **Configuration** — a named `config.json` + `profile/` directory pair inside the instance.
  Multiple configurations per instance are supported; one is active at a time.
- **Registry** — `~/.config/rsm/registry.yaml`, a YAML map of `name → install_dir`.
  It is the only file that lives outside the install directory.

There is **no global config file** for runtime settings (steamcmd path was removed).
steamcmd is detected from PATH and common locations on every invocation.

---

## Repository layout

```
cmd/rsm/              # CLI entrypoint — all Cobra commands live here
  main.go             # calls Execute()
  root.go             # root command, print helpers, instance/registry view
  instance.go         # rsm init, rsm delete
  config_cmd.go       # rsm config new/list/edit/use/delete
  install.go          # rsm install, rsm update; findSteamCMD() helper
  lifecycle.go        # rsm start/stop/restart/enable/disable/status
  logs.go             # rsm logs
  service.go          # rsm service install/enable/disable/status
  helpers.go          # isInstanceRunning(), regenerateUnit()

internal/config/      # Arma Reforger server config schema
  server.go           # ServerConfig struct (maps to config.json), DefaultServerConfig()
  server_test.go

internal/instance/    # Instance model and persistence
  instance.go         # Instance struct, path helpers, Save(), EnsureConfigDirs()
  manager.go          # Load(), LoadFromDir(), LoadFromCWD(), List(), Register(),
                      # Unregister(), Delete(), ResolveInstance(), registry R/W
  instance_test.go

internal/steam/       # steamcmd wrapper
  steam.go            # Find(), Require(), Install(), Update(), BuildUpdateCommand()
  steam_test.go

internal/systemd/     # systemd unit management
  systemd.go          # GenerateUnit(), InstallUnit(), ReinstallUnit(), IsInstalled(),
                      # EnsureInstalled(), RemoveUnit(), Start/Stop/Restart/Enable/Disable,
                      # IsActive(), IsEnabled(), Status(), sudoNotice()
  service.unit.tmpl   # Go template for the systemd unit file (embedded via go:embed)
  systemd_test.go

internal/logs/        # journalctl streaming
  logs.go             # Stream()

templates/            # Original template copy (not embedded; internal/systemd/ copy is used)
.github/workflows/
  ci.yml              # go vet + go test + go build on push/PR to main
  release.yml         # builds linux/amd64 + linux/arm64 on v* tags, publishes GitHub Release
Makefile              # build, build-linux, test, vet, clean, install, lint
```

---

## Build & test

```bash
# Build for the current platform
make build            # → dist/rsm

# Cross-compile for Linux (the actual deployment target)
make build-linux      # → dist/rsm-linux-amd64

# Run all tests
make test             # go test ./... -v

# Vet
make vet              # go vet ./...

# Install to $GOPATH/bin
make install
```

Version string is injected at build time:
```bash
go build -ldflags="-X main.version=v1.2.3" ./cmd/rsm
```

**All tests must pass before committing.** Run `make test` and `make vet`.

---

## On-disk layout (runtime)

```
<install_dir>/                        # chosen during rsm init
  rsm.yaml                            # instance metadata
  service.unit                        # local copy of generated systemd unit
  ArmaReforgerServer                  # game binary (installed by steamcmd)
  configuration/
    <config-name>/
      config.json                     # Arma Reforger server config
      profile/                        # server profile directory

~/.config/rsm/
  registry.yaml                       # name → install_dir map

/etc/systemd/system/
  rsm-<name>.service                  # installed by rsm service install / rsm start
```

---

## Key design decisions

### Instance resolution order

Every command that needs an instance calls `instance.ResolveInstance("")`.
Resolution priority:

1. Walk up from CWD looking for `rsm.yaml` (finds instance by file presence)
2. Check if CWD is inside any registered `install_dir` (registry fallback)
3. Single entry in registry → use it automatically
4. Error asking the user to `cd` into the right directory

`instance.LoadFromCWD()` is the exported convenience wrapper used by `root.go`.

### No global config

There is **no** `~/.config/rsm/config.yaml`. The steamcmd path is never cached.
`steam.Find()` searches PATH + common locations silently.
`steam.Require()` wraps `Find()` and returns a rich error with install instructions.
Commands that need steamcmd (`rsm install`, `rsm update`, `rsm init`) call
`steam.Require()` directly and fail immediately if not found.
Commands that use steamcmd optionally (unit generation) call `findSteamCMD()` in
`install.go` which returns an empty string on miss.

### systemd unit lifecycle

The unit file is written to `/etc/systemd/system/` by:
- `rsm service install` (explicit, always reinstalls)
- `rsm start` (auto-installs if not present)
- `rsm enable` (auto-installs if not present)

Every `sudo` invocation is preceded by a `[sudo] <reason>` notice on stderr via
`systemd.sudoNotice()`. This ensures the user always knows why a password prompt
appears.

### Configuration cloning

`rsm config new` behaves differently depending on whether an active config exists:
- **No active config** (first setup) → full interactive wizard
- **Active config exists** → silently copies `active/config.json` to the new name,
  prints a summary of copied values, offers to open in `$EDITOR` (default: yes)

### `rsm` with no subcommand

- **CWD is inside an instance** → focused instance view: status row, configs table,
  instance-specific common commands
- **CWD is outside any instance** → registry list + generic common commands
  (or getting-started if registry is empty)

---

## Adding a new command

1. Create a new file in `cmd/rsm/` (e.g. `cmd/rsm/mycommand.go`).
2. Define a `*cobra.Command` variable and register it in `func init()` with
   `rootCmd.AddCommand(...)` or as a subcommand of an existing group.
3. Use `instance.ResolveInstance("")` to identify the target instance.
4. Use the existing print helpers: `printSuccess`, `printInfo`, `printWarning`,
   `printError`, `printNextStep`.
5. Add tests if the command has non-trivial logic in `internal/`.

### Resolving an instance in a command

```go
resolved, err := instance.ResolveInstance("")
if err != nil {
    return err
}
inst, err := instance.Load(resolved)
if err != nil {
    return err
}
```

---

## Adding a new internal package

Place it under `internal/<name>/`. Keep packages focused:

| Package | Responsibility |
|---|---|
| `internal/config` | Arma Reforger `config.json` schema only |
| `internal/instance` | Instance struct, persistence, registry, resolution |
| `internal/steam` | steamcmd detection and execution |
| `internal/systemd` | systemd unit rendering and management |
| `internal/logs` | journalctl streaming |

Do not introduce a global config file or any persistent state beyond `rsm.yaml`
(per-instance) and `registry.yaml` (name → path map).

---

## Table rendering

`tabwriter` is **not used** for output tables because ANSI color codes inflate
byte widths and break column alignment.

Instead, compute column widths from **plain strings**, then pad manually and
colorise the padded string. The pattern used throughout:

```go
// Compensate for invisible ANSI bytes when using %-*s padding:
//   visual_width = plain_len
//   printf_field_width = desired_visual_width + (colorised_len - plain_len)
fmt.Printf("%-*s%-*s\n",
    w0+gap+len(colorStr)-len(plainStr), colorStr,
    w1+gap, otherColumn,
)
```

---

## Sudo operations

All privileged operations go through `internal/systemd`. The `sudoNotice()`
function there prints `  [sudo] <reason>` to stderr before every `sudo` exec.

Never call `sudo` directly from `cmd/` layer. Add it to the systemd package
if a new privileged operation is needed.

---

## Testing conventions

- Tests live in `internal/<pkg>/<pkg>_test.go` using the `_test` package suffix.
- Tests that touch the filesystem use `t.TempDir()` for isolation.
- Tests that use the registry or `HOME`-relative paths set `t.Setenv("HOME", tmpDir)`.
- Instance tests call both `inst.Save()` and `instance.Register(inst)` — both are
  required for full persistence.
- The `systemd` package tests do not require a running Linux system; they test
  unit file generation only (no actual `systemctl` calls).
- Do not add tests for `cmd/rsm/` commands that require interactive prompts
  (survey library). Test logic in `internal/` packages instead.

---

## Arma Reforger specifics

- **Steam App ID:** `1874900`
- **Experimental branch:** `-beta experiment` in steamcmd
- **Default scenario:** `{ECC61978EDCC2B5A}Missions/23_Campaign.conf` (Everon Game Master)
- **Default ports:** game `2001/udp`, A2S query `17777/udp`
- **Server binary:** `ArmaReforgerServer` (inside `install_dir`)
- **Launch flags used:** `-config`, `-profile`, `-maxFPS`, `-loadSessionSave`,
  `-backendLocalStorage`, `-logStats`
- **Logs:** via systemd journal; filter with `journalctl -u rsm-<name>.service`

---

## Release process

Releases are automated via GitHub Actions (`.github/workflows/release.yml`).

```bash
git tag v1.2.3
git push origin v1.2.3
```

This triggers a build of `rsm-linux-amd64` and `rsm-linux-arm64`, generates
`checksums.txt` (SHA256), and publishes a GitHub Release with install instructions.

Pre-release versions (e.g. `v1.2.3-beta.1`) are automatically marked as pre-release
because the tag contains a hyphen.
