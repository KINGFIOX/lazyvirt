# lazyvirt

A lazygit / lazydocker-style TUI for managing libvirt virtual machines, backed
by the `virsh` CLI.

```
┌─Domains (2)──────────────────────┐┌─dominfo────────────────────────┐
│    NAME          STATE  AUTOSTART ││ Id:         1                 │
│ ▸  resolute      running disable  ││ Name:       resolute          │
│    demo        shut off enable    ││ State:      running           │
│                                  ││ Autostart:  disable           │
└──────────────────────────────────┘│ CPU(s):     2                 │
┌─Command Log───────────────────────────────────────────────────────┐
│ 12:01:33  shutdown resolute → Domain resolute is being shutdown   │
└───────────────────────────────────────────────────────────────────┘
 j/k select • s start • S shutdown • r reboot • x force off • a autostart …
```

## Build

```bash
go build -o lazyvirt .
```

## Keybindings

| Key | Action | virsh command |
| --- | --- | --- |
| `j/k` `↑/↓` | Select domain | — |
| `s` | Start | `virsh start <name>` |
| `S` | Graceful shutdown | `virsh shutdown <name>` |
| `r` | Reboot | `virsh reboot <name>` |
| `x` | Hard power-off (confirm) | `virsh destroy <name>` |
| `a` | Toggle autostart (inverts current state) | `virsh autostart <name>` / `--disable` |
| `d` | Undefine (confirm) | `virsh undefine <name>` |
| `U` | Undefine with disks (confirm, dangerous) | `virsh undefine <name> --nvram --remove-all-storage` |
| `R` | Manual refresh | — |
| `q` / `Ctrl+C` | Quit | — |

- The list auto-refreshes every 5 seconds and immediately after every
  operation; the selection is tracked by domain name, so reordering never
  moves it.
- All operations run asynchronously (30s timeout).
- Every virsh invocation is echoed in the Command Log panel — the full command
  line followed by its output (raw stderr on failure) — including the initial
  `virsh list --all` / `virsh dominfo` fetches at startup; only the periodic
  background refresh stays silent.
- When virsh cannot connect, an error banner replaces the list and the UI
  keeps retrying instead of exiting.

## Other usage

```bash
lazyvirt --list    # non-interactive domain listing (smoke test / scripting)

# manage a remote libvirt host (the backend is virsh, so the URI just passes
# through):
LIBVIRT_DEFAULT_URI=qemu+ssh://user@host/system ./lazyvirt
```

## Tests

```bash
go test ./...                                  # unit tests (parser, rendering, key flow)
go test -tags integration ./internal/virsh/    # integration: full flow on a throwaway
                                               # test domain, never touches existing domains
```

## Layout

```
main.go               entry point + --list mode
internal/virsh/       virsh command wrappers and output parsing (30s timeouts,
                      stderr captured)
internal/tui/         bubbletea three-panel UI: model / update / view / styles
```

## License

[MIT](LICENSE)
