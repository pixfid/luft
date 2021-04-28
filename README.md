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
  -m, --masstorage                            show only mass storage devices [$MASSTORAGE]
  -u, --untrusted                             show only untrusted devices [$UNTRUSTED]
  -n, --number=                               number of events to show [$NUMBER]
  -s, --sort=[asc|desc]                       sort events (default: asc) [$SORT]
  -e, --export                                export events [$EXPORT]
  -c, --check                                 check devices for whitelist [$CHECK]
  -E, --extusbids                             external usbids data base [$EXTUSBIDS]
  -W, --whitelist=                            whitelist path [$WHITELIST]
  -U, --usbids=                               usbids path (default: /var/lib/usbutils/usb.ids) [$USBIDS]

events:
  -S, --events.source=[local|remote|database] events target
      --events.path=                          log directory (default: /var/log/)

export:
  -F, --events.export.format=[json|xml|pdf]   events export format (default: pdf) [$EVENTS_EXPORT_FORMAT]

remote:
  -I, --events.remote.ip=                     ip address [$EVENTS_REMOTE_IP]
      --events.remote.port=                   ssh port (default: 22) [$EVENTS_REMOTE_PORT]
  -L, --events.remote.login=                  login [$EVENTS_REMOTE_LOGIN]
  -P, --events.remote.password=               password [$EVENTS_REMOTE_PASSWORD]

Help Options:
  -h, --help                                  Show this help message


```

Examples
==========

### Events history:

#### Get USB event history:
```./luft -cm -S=local -W=99_PDAC_LOCAL_flash.rules```

#### Get USB events history from remote host:
```./luft -cm -W=99_PDAC_LOCAL_flash.rules -S=remote -I=10.211.55.11 -L=user -P=password```

<img width="1282" alt="Screenshot 2021-04-18 at 23 07 39" src="https://user-images.githubusercontent.com/1672087/115159258-f6259d00-a09a-11eb-90e1-428e0793a1b0.png">


### Export with various formats json, xml, pdf (with logo `stats.png`)

#### Export USB event history
```./luft -cmE -S=local -W=99_PDAC_LOCAL_flash.rules```

### PDF Report example:
<img width="1324" alt="Screenshot 2021-04-11 at 14 36 11" src="https://user-images.githubusercontent.com/1672087/114302784-4e750180-9ad3-11eb-9642-cc760bbf9c3f.png">


TODO
==========

* [ ] Rewrite all ugly code
* [ ] Update usb.ids
* [ ] View events with data \ time intervals
* [ ] Search usb device with only one of (vid | pid)

Credits & References
==========

* [cfmt](https://github.com/i582/cfmt)
* [tablewriter](https://github.com/olekukonko/tablewriter)
* [gofpdf](https://github.com/jung-kurt/gofpdf)
* [go-flags](https://github.com/umputun/go-flags)

## Contact

For any questions — tg: `@cffaedfe`.

## License

This project is under the **MIT License**. See the [LICENSE](https://github.com/pixfid/luft/blob/master/LICENSE) file for the full license text.