# Migration Guide: CLI Changes

## Overview

LUFT has migrated from `go-flags` to **Cobra** for better CLI organization and flexibility. The new structure uses **subcommands** for different operations.

## Breaking Changes

### Command Structure

**Old (go-flags):**
```bash
luft -S local -cm -W whitelist.rules
luft --update-usbids
luft --clear-cache
```

**New (Cobra):**
```bash
luft events --source local -cm -W whitelist.rules
luft update
luft cache clear
```

## Migration Guide

### Basic Event Collection

**Before:**
```bash
./luft -S local
```

**After:**
```bash
./luft events --source local
```

### With Filters

**Before:**
```bash
./luft -S local -cm -u -W whitelist.rules
```

**After:**
```bash
./luft events --source local -cm -u -W whitelist.rules
```

### Remote Collection

**Before:**
```bash
./luft -S remote -I 10.0.0.1 -L user -K ~/.ssh/id_rsa
```

**After:**
```bash
./luft events --source remote --remote-ip 10.0.0.1 --remote-login user --remote-key ~/.ssh/id_rsa
```

Or using config:
```bash
./luft events --source remote --remote-host prod-server
```

### Export

**Before:**
```bash
./luft -S local -e -F pdf -N report
```

**After:**
```bash
./luft events --source local --export --format pdf --output report
```

### Update USB IDs

**Before:**
```bash
sudo ./luft --update-usbids
```

**After:**
```bash
sudo ./luft update
```

With custom path:
```bash
./luft update --path ~/.local/share/luft/usb.ids
```

**Note:** The update command now uses the official USB ID Repository source (http://www.linux-usb.org/usb.ids) for reliability and consistency.

### Clear Cache

**Before:**
```bash
./luft --clear-cache
```

**After:**
```bash
./luft cache clear
```

With custom USB IDs path:
```bash
./luft cache clear --usbids ~/usb.ids
```

## New Features

### Improved Help System

Get help for any command:
```bash
./luft --help
./luft events --help
./luft update --help
./luft cache --help
./luft cache clear --help
```

### Shell Completion

Generate shell completion scripts:
```bash
./luft completion bash > /etc/bash_completion.d/luft
./luft completion zsh > ~/.zsh/completion/_luft
./luft completion fish > ~/.config/fish/completions/luft.fish
./luft completion powershell > luft.ps1
```

### Subcommands

Commands are now organized hierarchically:
- `luft events` - Collect and analyze events
  - `--source local` - Local analysis
  - `--source remote` - Remote analysis
- `luft update` - Update USB IDs database
- `luft cache` - Manage cache
  - `luft cache clear` - Clear cache

### Better Flag Organization

Flags are now grouped logically:
- **Source flags**: `--source`, `--path`, `--remote-host`
- **Filter flags**: `--mass-storage`, `--untrusted`, `--check-whitelist`
- **Export flags**: `--export`, `--format`, `--output`
- **Performance flags**: `--workers`, `--streaming`
- **Remote flags**: `--remote-ip`, `--remote-login`, `--remote-key`

### Global Flags

Some flags are available for all commands:
- `--config` - Config file path
- `--help` - Show help
- `--version` - Show version

## Flag Aliases

Short flags remain the same:
- `-S` → `--source`
- `-m` → `--mass-storage`
- `-u` → `--untrusted`
- `-c` → `--check-whitelist`
- `-e` → `--export`
- `-F` → `--format`
- `-o` → `--output`
- `-w` → `--workers`
- `-W` → `--whitelist`
- `-U` → `--usbids`
- `-I` → `--remote-ip`
- `-L` → `--remote-login`
- `-K` → `--remote-key`
- `-T` → `--remote-timeout`

## Config File

Config file format remains **unchanged**. All config file features work as before.

## Environment Variables

Environment variables still work but need to be used with the `events` command:
```bash
LUFT_CONFIG=~/.luft.yaml ./luft events --source local
```

## Benefits

1. **Better Organization**: Commands are grouped logically
2. **Clearer Help**: Each command has detailed help with examples
3. **Shell Completion**: Auto-complete commands and flags
4. **Extensibility**: Easy to add new commands
5. **Standard Tool**: Cobra is industry standard (used by kubectl, docker, etc.)
6. **Better Validation**: Flag validation per command
7. **Man Pages**: Can generate man pages (coming soon)

## Troubleshooting

### "Required flag not specified"

Make sure you're using the `events` command:
```bash
# Wrong
./luft --source local

# Correct
./luft events --source local
```

### Old scripts breaking

Update your scripts to use the new command structure. You can create an alias for backward compatibility:
```bash
alias luft-old='luft events'
```

### Missing subcommand

If you forget the subcommand, you'll see:
```bash
$ ./luft
LUFT - Linux USB Forensic Tool
...
Use "luft [command] --help" for more information about a command.
```

Simply add the appropriate subcommand (`events`, `update`, or `cache`).
