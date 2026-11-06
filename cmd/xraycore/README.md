# xraycore

Build geosite.dat and geoip.dat from `data/domains` and `data/ips`.

## Run

```bash
go run ./cmd/xraycore
go run ./cmd/xraycore -root=. -out=release
```

## Flags

```
-root          repo root with data folder, default .
-out           output folder for geosite.dat and geoip.dat, default root/release
```
