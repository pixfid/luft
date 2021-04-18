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
   luft {flags}
   luft <command> {flags}

Commands: 
   help                          displays usage information
   ids                           usb.ids database
   local                         local events history.
   remote                        remote events history.
   version                       displays version number

Flags: 
   -h, --help                    displays usage information of the application or a command (default: false)
   -v, --version                 displays version number (default: false)

```

Examples
==========

### Events history:

#### Local events' history view: 
```./luft local history --sort=asc --check```

#### Local events view with external LOG, WHITELIST and USB.IDS:
```./luft local history --log ~/Downloads/log --sort=asc --check --whitelist=99_PDAC_LOCAL_flash.rules --usbids=usb.ids --external```

### Remote events view
```./luft remote history --server=127.0.0.1 --port=22 --login=login --password=password --check --sort=asc --whitelist=99_PDAC_LOCAL_flash.rules```
<img width="1282" alt="Screenshot 2021-04-18 at 23 07 39" src="https://user-images.githubusercontent.com/1672087/115159258-f6259d00-a09a-11eb-90e1-428e0793a1b0.png">


### Export with various formats json, xml, pdf (with logo `stats.png`)
### Export With external LOG, WHITELIST and USB.IDS
```./luft local export --format pdf --log ~/Downloads/log --sort asc --check --whitelist 99_PDAC_LOCAL_flash.rules --usbids usb.ids --external```

### PDF Report example:
<img width="1324" alt="Screenshot 2021-04-11 at 14 36 11" src="https://user-images.githubusercontent.com/1672087/114302784-4e750180-9ad3-11eb-9642-cc760bbf9c3f.png">

### USB.IDS: (`search` and `download` / `update` database)

####  Search device by `vid` & `pid`

```./luft ids search --vid 03f0 --pid 0f0c```

TODO
==========

* [ ] Rewrite all ugly code
* [ ] Update usb.ids
* [ ] View events with data \ time intervals
* [ ] Search usb device with only one of (vid | pid)

Credits & References
==========

* [usbrip](https://github.com/snovvcrash/usbrip)
* [google/gousb](https://github.com/google/gousb)
* [cfmt](github.com/i582/cfmt)
* [commando](github.com/thatisuday/commando)
* [tablewriter](github.com/olekukonko/tablewriter)
* [gofpdf](github.com/jung-kurt/gofpdf)
