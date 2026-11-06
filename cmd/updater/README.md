# updater

Update source lists from `sources.yaml`.

## Run

```bash
go run ./cmd/updater
go run ./cmd/updater -dry-run -only=domains -config=sources.yaml
```

## Flags

```
-dry-run       show changes but do not write files, default false
-only          update only file paths that end with this string, default empty
-config        path to sources config file, default sources.yaml
```
