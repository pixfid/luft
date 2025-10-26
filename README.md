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

```
$ ./luft -h

Usage:
  luft [OPTIONS]

Application Options:
      --config=                               path to config file (YAML) [$LUFT_CONFIG]
      --update-usbids                         download and update USB IDs database
  -m, --masstorage                            show only mass storage devices [$MASSTORAGE]
  -u, --untrusted                             show only untrusted devices [$UNTRUSTED]
  -n, --number=                               number of events to show [$NUMBER]
  -s, --sort=[asc|desc]                       sort events (default: asc) [$SORT]
  -e, --export                                export events [$EXPORT]
  -c, --check                                 check devices for whitelist [$CHECK]
  -W, --whitelist=                            external whitelist path [$WHITELIST]
  -U, --usbids=                               usbids path (default: /var/lib/usbutils/usb.ids) [$USBIDS]

events:
  -S, --events.source=[local|remote|database] events target
      --events.remote-host=                   remote host name from config file [$REMOTE_HOST]
      --events.path=                          log directory (default: /var/log/)

export:
  -F, --events.export.format=[json|xml|pdf]   events export format (default: pdf) [$EVENTS_EXPORT_FORMAT]
  -N, --events.export.filename=               events export file name (default: events_data) [$EVENTS_EXPORT_FILENAME]

remote:
  -I, --events.remote.ip=                     ip address [$EVENTS_REMOTE_IP]
      --events.remote.port=                   ssh port (default: 22) [$EVENTS_REMOTE_PORT]
  -L, --events.remote.login=                  login [$EVENTS_REMOTE_LOGIN]
  -P, --events.remote.password=               password (deprecated, use SSH key instead) [$EVENTS_REMOTE_PASSWORD]
  -K, --events.remote.ssh-key=                path to SSH private key (recommended) [$EVENTS_REMOTE_SSH_KEY]
  -T, --events.remote.timeout=                SSH connection timeout in seconds (default: 30) [$EVENTS_REMOTE_TIMEOUT]
      --events.remote.insecure-ssh            skip SSH host key verification (NOT RECOMMENDED) [$EVENTS_REMOTE_INSECURE_SSH]

Help Options:
  -h, --help                                  Show this help message
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
sudo ./luft --update-usbids

# Update to custom location
./luft --update-usbids --usbids=~/.local/share/luft/usb.ids

# Use updated database
./luft -S local --usbids=~/.local/share/luft/usb.ids
```

The update command will:
1. Try multiple sources (usb-ids.gowly.com, GitHub, linux-usb.org)
2. Show download progress
3. Verify the database by loading it
4. Display version and date information

**Sources (in order of priority):**
- https://usb-ids.gowly.com/usb.ids
- https://raw.githubusercontent.com/gentoo/hwids/master/usb.ids
- http://www.linux-usb.org/usb.ids

**Note:** If the default path is not writable, the tool will automatically use `~/.local/share/luft/usb.ids` as an alternative.

Examples
==========

### Events history:

#### Get USB event history (local):
```bash
./luft -cm -S=local -W=99_PDAC_LOCAL_flash.rules
```

#### Get USB events from remote host (CLI flags):
```bash
./luft -cm -W=99_PDAC_LOCAL_flash.rules -S=remote -I=10.211.55.11 -L=user -K=~/.ssh/id_rsa
```

#### Get USB events from remote host (using config):
```bash
# First, setup ~/.luft.yaml with remote host details
./luft -cm -S=remote --remote-host=prod-server
```

#### Use custom config file:
```bash
./luft --config=/path/to/custom.yaml -S=local
```

<img width="1274" alt="Screenshot 2021-05-06 at 17 58 18" src="https://user-images.githubusercontent.com/1672087/117387775-28842680-aef2-11eb-8bfd-cfa084db0f05.png">


### Export with various formats json, xml, pdf (with logo `stats.png`)

#### Export USB event history
```./luft -cmE -S=local -W=99_PDAC_LOCAL_flash.rules```

### PDF Report example:
<img width="1324" alt="Screenshot 2021-04-11 at 14 36 11" src="https://user-images.githubusercontent.com/1672087/114302784-4e750180-9ad3-11eb-9642-cc760bbf9c3f.png">


TODO
==========

* [ ] Rewrite all ugly code
* [x] Update usb.ids (implemented via `--update-usbids`)
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