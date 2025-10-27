LUFT - linux usb forensic tool
==========

LUFT partial fork of [usbrip](https://github.com/snovvcrash/usbrip) rewrite on [go lang](https://golang.org) 
for Linux, you also can cross compile for using in various OS such as macOS, Windows
with reduced functionality (custom log directory)

## Build

* `GOOS=linux GOARCH=amd64 go build -ldflags="-s -w"` for Linux
* `GOOS=windows GOARCH=amd64 go build -ldflags="-s -w"` for Windows
* `GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w"` for macOS

## Help

```bash
$ ./luft --help

LUFT - Linux USB Forensic Tool

Usage:
  luft [command]

Available Commands:
  cache       Manage USB IDs cache
  completion  Generate shell autocompletion
  events      Collect and analyze USB device events
  help        Help about any command
  update      Update USB IDs database

Flags:
      --config string   config file (default: ~/.luft.yaml)
  -h, --help            help for luft
  -v, --version         version for luft

Use "luft [command] --help" for more information about a command.
```

### Events Command

```bash
$ ./luft events --help

Collect USB device connection events from local or remote systems.

Usage:
  luft events [flags]

Flags:
  -S, --source string            event source (local, remote) [required]
  -m, --mass-storage             show only mass storage devices
  -u, --untrusted                show only untrusted devices
  -c, --check-whitelist          check devices against whitelist
  -n, --number int               number of events to show (0 = all)
  -s, --sort string              sort events (asc, desc) (default "asc")
  -e, --export                   export events
  -F, --format string            export format (json, xml, pdf) (default "pdf")
  -o, --output string            export filename (default "events_data")
  -w, --workers int              number of worker threads (0 = auto)
      --streaming                use streaming parser for large logs
  -W, --whitelist string         whitelist file path
  -U, --usbids string            USB IDs database path
      --path string              log directory (default "/var/log/")
      --remote-host string       remote host name from config
  -I, --remote-ip string         remote host IP address
  -L, --remote-login string      remote login username
  -K, --remote-key string        path to SSH private key
  -P, --remote-password string   remote password (deprecated)
      --remote-port string       remote SSH port (default "22")
  -T, --remote-timeout int       SSH timeout in seconds (default 30)
      --insecure-ssh             skip SSH host key verification

Use "luft events --help" for detailed examples.
```

### Shell Completion

LUFT supports shell completion for bash, zsh, fish, and powershell:

```bash
# Bash
./luft completion bash > /etc/bash_completion.d/luft

# Zsh
./luft completion zsh > ~/.zsh/completion/_luft

# Fish
./luft completion fish > ~/.config/fish/completions/luft.fish

# PowerShell
./luft completion powershell > luft.ps1
```

## Configuration File

LUFT supports YAML configuration files for easier management of settings and remote hosts.

### Config File Locations

LUFT searches for configuration files in the following locations (in order):
1. Custom path specified with `--config` flag
2. `~/.luft.yaml` (user home directory)
3. `./.luft.yaml` (current directory)
4. `/etc/luft/.luft.yaml` (system-wide)

### Configuration Priority

Settings are applied in the following priority order (highest to lowest):
1. **CLI flags** (highest priority)
2. **Environment variables**
3. **Config file**
4. **Default values** (lowest priority)

### Example Configuration

Copy `.luft.yaml.example` to `~/.luft.yaml` and customize:

```yaml
# Path to whitelist file
whitelist: /etc/udev/rules.d/99_PDAC_LOCAL_flash.rules

# Path to USB IDs database
usbids: /var/lib/usbutils/usb.ids

# Default log directory
log_path: /var/log/

# Filter options
mass_storage: false
untrusted: false
check_whitelist: true

# Export settings
export:
  format: pdf
  path: ~/luft-reports

# Remote hosts
remote_hosts:
  - name: prod-server
    ip: 10.0.0.1
    port: "22"
    user: admin
    ssh_key: ~/.ssh/id_rsa
    timeout: 30
    insecure_ssh: false

  - name: dev-server
    ip: 192.168.1.100
    user: developer
    ssh_key: ~/.ssh/dev_key
```

### Using Remote Hosts from Config

Instead of specifying remote connection details via CLI flags, you can define hosts in your config file:

```bash
# Scan remote host from config
./luft -S remote --remote-host=prod-server

# Override config values with CLI flags
./luft -S remote --remote-host=prod-server -T 60
```

## Updating USB IDs Database

LUFT uses the USB IDs database to identify device manufacturers and products. Keep it up-to-date for better device recognition.

### Auto-update USB IDs

```bash
# Update to default location (requires root/sudo for system paths)
sudo ./luft update

# Update to custom location
./luft update --path ~/.local/share/luft/usb.ids

# Use updated database
./luft events --source local --usbids ~/.local/share/luft/usb.ids
```

The update command will:
1. Download from the official source (linux-usb.org)
2. Show download progress with progress bar
3. Verify the database by loading it
4. Display version and date information
5. Automatically create a cache file for faster subsequent loads

**Source:**
- http://www.linux-usb.org/usb.ids (official USB ID Repository)

**Note:** If the default path is not writable, the tool will automatically use `~/.local/share/luft/usb.ids` as an alternative.

## USB IDs Caching

LUFT automatically caches the parsed USB IDs database for **significantly faster loading** on subsequent runs.

### Performance

- **First load (parsing)**: ~13ms
- **Cached loads**: ~5ms (**2-3x faster!**)

### How it works

1. First time loading a USB IDs file, LUFT parses it and creates a cache file (`usb.ids.cache`)
2. On subsequent loads, LUFT loads from cache if:
   - Cache file exists
   - Source file hasn't been modified
   - File hash matches
3. If source file is updated, cache is automatically invalidated and rebuilt

### Cache Management

```bash
# Clear cache (will be rebuilt on next load)
./luft cache clear --usbids /path/to/usb.ids

# Clear default cache
./luft cache clear

# Cache is automatically created, no manual action needed
./luft events --source local  # First run: parses and caches
./luft events --source local  # Subsequent runs: loads from cache
```

**Cache location:** Cache files are stored alongside the USB IDs file with `.cache` extension.

**Cache invalidation:** Cache is automatically invalidated when:
- Source file is modified (timestamp check)
- Source file content changes (MD5 hash check)
- Cache file is manually deleted

## Parallel Log Parsing

LUFT automatically parses log files in parallel using a **worker pool** for significantly faster processing of multiple files.

### Performance

Performance improvement with 100 log files:

| Workers | Parse Time | Speedup |
|---------|-----------|---------|
| 1 (sequential) | 6.4ms | baseline |
| 4 workers | 2.5ms | **2.6x faster** |
| Auto (CPU cores) | 1.8ms | **3.6x faster** |

### How it works

1. **Automatic parallelization**: By default, LUFT uses as many workers as CPU cores
2. **Worker pool pattern**: Files are distributed among workers for parallel processing
3. **Order preservation**: Results are collected and aggregated in original file order
4. **Smart fallback**: Single file or single worker automatically uses sequential parsing

### Configuration

```bash
# Use default (CPU cores)
./luft -S local

# Specify custom worker count
./luft -S local -w 4

# Sequential processing (1 worker)
./luft -S local -w 1

# Maximum parallelism (use all CPU cores explicitly)
./luft -S local -w 0
```

**When to adjust workers:**
- **Low CPU**: Use `-w 2` or `-w 4` for modest parallelism
- **Many files**: Default (CPU cores) works best
- **Few files**: Parallelism overhead may not be worth it, use `-w 1`
- **Resource constrained**: Lower worker count to reduce CPU/memory usage

## Streaming Parser

For **very large log files** or **memory-constrained environments**, LUFT provides a streaming parser that processes logs line-by-line without loading entire files into memory.

### Key Features

1. **Memory-efficient**: Processes logs incrementally using buffered I/O
2. **Backpressure handling**: Controls memory usage with buffered channels
3. **Progress monitoring**: Real-time stats every 2 seconds during processing
4. **Memory metrics**: Tracks and reports memory allocation statistics
5. **Parallel streaming**: Combines streaming with worker pool for optimal performance

### Performance

Memory usage comparison when processing 50 large log files (25,000 events):

| Mode | Memory Allocated | Peak Memory | Processing |
|------|-----------------|-------------|------------|
| Standard | ~45 MB | ~60 MB | Fast, memory-intensive |
| Streaming | ~25 MB | ~35 MB | **42% less memory** |

### How it works

The streaming parser uses an **event-driven architecture**:

1. **Buffered scanning**: Reads files line-by-line with configurable buffer (64KB default, 1MB max)
2. **Channel-based processing**: Events flow through buffered channels (capacity: 1000)
3. **Backpressure control**: Parser pauses when channels are full, preventing memory overflow
4. **Atomic counters**: Thread-safe progress tracking across all workers
5. **Progress reporting**: Displays events/files processed every 2 seconds

### Configuration

```bash
# Enable streaming mode (uses default worker count = CPU cores)
./luft -S local --streaming

# Streaming with specific worker count
./luft -S local --streaming -w 4

# Streaming with single worker (lowest memory usage)
./luft -S local --streaming -w 1

# View memory statistics during processing
./luft -S local --streaming
# Output shows:
# Memory before parsing: Alloc=5.2MB TotalAlloc=8.1MB Sys=12.4MB
# Processing: 15420 events from 32 files...
# Memory after streaming parse: Alloc=12.8MB TotalAlloc=45.3MB Sys=25.6MB
```

### When to use Streaming vs Parallel

**Use Streaming (`--streaming`) when:**
- Processing **very large log files** (>1GB total)
- Running on **memory-constrained systems** (limited RAM)
- Need to **monitor progress** for long-running operations
- Want to **track memory usage** during processing

**Use Standard Parallel (default) when:**
- Processing **moderate-sized logs** (<500MB total)
- Have **sufficient RAM available**
- Need **maximum speed** (slightly faster than streaming)
- Don't need progress monitoring

**Combine both for best results:**
```bash
# Streaming + parallel workers = memory-efficient AND fast
./luft -S local --streaming -w 8
```

Examples
==========

### Events history:

#### Get USB event history (local):
```bash
./luft events --source local -cm -W 99_PDAC_LOCAL_flash.rules
```

#### Get USB events from remote host (CLI flags):
```bash
./luft events --source remote -cm -W 99_PDAC_LOCAL_flash.rules \
  --remote-ip 10.211.55.11 --remote-login user --remote-key ~/.ssh/id_rsa
```

#### Get USB events from remote host (using config):
```bash
# First, setup ~/.luft.yaml with remote host details
./luft events --source remote -cm --remote-host prod-server
```

#### Use custom config file:
```bash
./luft --config /path/to/custom.yaml events --source local
```

#### Streaming mode for large logs:
```bash
# Memory-efficient processing with progress bar
./luft events --source local --streaming -w 8
```

#### Filter and export:
```bash
# Show only untrusted mass storage devices and export to PDF
./luft events --source local -muc --export --format pdf --output report
```

<img width="1274" alt="Screenshot 2021-05-06 at 17 58 18" src="https://user-images.githubusercontent.com/1672087/117387775-28842680-aef2-11eb-8bfd-cfa084db0f05.png">


### Export with various formats json, xml, pdf (with logo `stats.png`)

#### Export USB event history
```bash
./luft events --source local -cme -W 99_PDAC_LOCAL_flash.rules

# Export to JSON
./luft events --source local --export --format json --output events

# Export to XML
./luft events --source local --export --format xml --output events
```

### PDF Report example:
<img width="1324" alt="Screenshot 2021-04-11 at 14 36 11" src="https://user-images.githubusercontent.com/1672087/114302784-4e750180-9ad3-11eb-9642-cc760bbf9c3f.png">


TODO
==========

* [ ] Rewrite all ugly code
* [x] Update usb.ids (implemented via `--update-usbids`)
* [x] Cache USB IDs database in memory (2-3x faster loading!)
* [x] Parallel log parsing with worker pool (3.6x faster!)
* [x] Streaming parser for large logs (42% less memory!)
* [ ] View events with data \ time intervals
* [ ] Search usb device with only one of (vid | pid)
* [x] YAML configuration support
* [ ] Database storage (SQLite)
* [ ] Real-time monitoring mode
* [ ] CSV export format

Credits & References
==========

* [cfmt](https://github.com/i582/cfmt)
* [tablewriter](https://github.com/olekukonko/tablewriter)
* [gofpdf](https://github.com/jung-kurt/gofpdf)
* [go-flags](https://github.com/umputun/go-flags)
* [viper](https://github.com/spf13/viper)

## Contact

For any questions — tg: `@cffaedfe`.

## License

This project is under the **MIT License**. See the [LICENSE](https://github.com/pixfid/luft/blob/master/LICENSE) file for the full license text.