# Installation

## Binary

- Download from https://github.com/mdouchement/openfand/releases/latest
  - Or compile via Taskfile or `go build cmd/openfand/main.go`
- Put the binary in `/usr/sbin/openfand`.

## Configuration

This file contains the configuration of openfand:
- `/etc/openfand/openfand.yml`

```yaml
socket: /run/openfand/openfand.sock

debug: false

fan_settings:
  fan1:
    label: FrontTop
    fan_step_up: 1s
    fan_step_down: 1s

...
```
> cf. `config.sample.yml` at the root of this repository.

## Systemd

`/lib/systemd/system/openfand.service`

```toml
[Unit]
Description=openfand
After=network.target

[Service]
Restart=on-failure
KillSignal=SIGINT

ExecStart=/usr/sbin/openfand

[Install]
WantedBy=multi-user.target
```

> Logs: `journalctl --unit openfand`
