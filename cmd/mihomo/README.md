# mihomo

Build mihomo .mrs rule-sets from `data/domains` and `data/ips`.

## Run

```bash
go run ./cmd/mihomo
go run ./cmd/mihomo -root=. -out=release -mihomo-bin=mihomo
```

## Flags

```
-root          repo root with data folder, default .
-out           output folder for .mrs files, default root/release
-mihomo-bin    path to mihomo binary or name in PATH, default mihomo
```
