# singbox

Build sing-box .srs rule-sets from `data/domains` and `data/ips`.

## Run

```bash
go run ./cmd/singbox
go run ./cmd/singbox -root=. -out=release -singbox-bin=sing-box
```

## Flags

```
-root          repo root with data folder, default .
-out           output folder for .srs files, default root/release
-singbox-bin   path to sing-box binary or name in PATH, default sing-box
```
