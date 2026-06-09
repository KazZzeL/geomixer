# geomixer

Command-line tool to merge and transform Xray-compatible geosite and geoip rule files.

## Features

- **Merge** multiple geoip/geosite sources into single output files
- **Filter** by IP type (IPv4/IPv6) and categories
- **Transform** domain attributes (append, delete, reset)
- **Subdomain pruning** — removes child entries when a parent domain rule exists
- **Supports both** Xray protobuf (`.dat`) and plain-text formats

## Installation

```bash
go install github.com/KazZzeL/geomixer@latest
```

Or download a pre-built binary from [releases](https://github.com/KazZzeL/geomixer/releases).

## Quick start

### 1. Create a config file

Config supports **JSON** and **YAML** (auto-detected by extension).

```yaml
# config.yaml
geosite:
  inputs:
    - name: v2fly
      kind: geofile
      url: https://github.com/v2fly/domain-list-community/releases/latest/download/dlc.dat

  outputs:
    - name: geosite.dat
      categories:
        - name: cn
          steps:
            - action: add
              input: v2fly
              include: ["cn"]
        - name: proxy
          steps:
            - action: add
              input: v2fly
              include: ["google", "youtube", "netflix"]

geoip:
  inputs:
    - name: v2fly
      kind: geofile
      url: https://github.com/v2fly/geoip/releases/latest/download/geoip.dat

    - name: proxy-add
      kind: plain
      path: "./input/proxy-add.txt"

    - name: proxy-del
      kind: list
      list:
        - "1.1.1.1/32"
        - "1.0.0.1"

  outputs:
    - name: geoip.dat
      categories:
        - name: cn
          steps:
            - action: add
              input: v2fly
              include: ["cn"]
        - name: private
          steps:
            - action: add
              input: v2fly
              include: ["private"]
        - name: proxy
            - action: add
              input: proxy-add
            - action: del
              input: proxy-del
```

### 2. Run

```bash
geomixer mix config.yaml -o ./output
```

### 3. Check output

```bash
ls output/
# geosite.dat  geoip.dat
```

## Configuration reference

Full JSON Schema is generated via `go generate` and [committed to the repo](https://github.com/KazZzeL/geomixer/blob/master/jsonschema/geomixer.schema.json). Reference it in your editor for autocompletion and validation:

```yaml
# config.yaml
# yaml-language-server: $schema=https://raw.githubusercontent.com/KazZzeL/geomixer/master/jsonschema/geomixer.schema.json
```

```json
{
  "$schema": "https://raw.githubusercontent.com/KazZzeL/geomixer/master/jsonschema/geomixer.schema.json",
  "geosite": { ... },
  "geoip": { ... }
}
```

Or regenerate locally: `geomixer schema`

### Input kinds

| Kind | Description | Source |
|---|---|---|
| `geofile` | v2fly protobuf format | `path` or `url` |
| `plain` | Plain text (one CIDR/domain per line) | `path` or `url` |
| `list` | Inline array | `list` field |

### Step options

| Option | Type | Description |
|---|---|---|
| `ignoreIPv4` | `bool` | Skip IPv4 CIDRs (geoip only) |
| `ignoreIPv6` | `bool` | Skip IPv6 CIDRs (geoip only) |
| `resetAttributes` | `bool` | Clear all domain attributes before appending |
| `deleteAttributes` | `[]string` | Remove specific attribute keys |
| `appendAttributes` | `[]string` | Add boolean attributes (e.g. `"ads"`, `"tracking"`) |

### Validation

- Input names must be unique within a runner
- Step `include` and `exclude` must not overlap
- Output category names must be unique
- At least one IP type must be enabled for geoip categories

## CLI

```
geomixer mix [flags] <config>
  -o, --output string     Output directory (default "./output")
  -s, --geosite-only      Process geosite only
  -i, --geoip-only        Process geoip only
      --min-tls string    TLS minimum version (default "1.3")
      --http-timeout      HTTP timeout (default 15s)

geomixer schema [flags]
  -o, --output string     Output directory (default "./jsonschema")
  -f, --filename string   Schema filename (default "geomixer.schema.json")
```

## Benchmarks

**Data sources:** v2fly/geoip (22 MB) + v2fly/domain-list-community (2.1 MB).
**Machine:** Intel Core i7-9750H @ 2.6 GHz, 16 GB RAM, Go 1.26.

One geoip + one geosite category, combined sequential run:

| Size | Time | Memory | Allocs |
|---|---|---|---|
| 22 MB + 2.1 MB | ~940 ms | ~930 MB | ~3.3M |

> Memory dominated by `proto.Unmarshal` (full protobuf tree) and `IPSetBuilder.IPSet()`. For reference, the full pipeline completes in ~1 s on a 6-core laptop CPU.

## Development

**Requirements:** Go 1.26+

```bash
# Lint
golangci-lint run

# Test
go test -race -count=1 ./...

# Regenerate schema (must stay in sync with code)
go generate ./...
git diff --exit-code jsonschema/

# Benchmarks
go test -bench=. -benchmem ./...
```

### Commit convention

This project uses [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: add ignoreIPv4 option
fix: handle nil options in buildPrefixes
chore: bump goreleaser to v2
```

## CI/CD

| Event | Workflow | Actions |
|---|---|---|
| Push / PR | `ci.yml` | commitlint → golangci-lint → `go test -race` → schema freshness check |
| Tag `v*` | `release.yml` | Test → goreleaser (generate, build, archive, changelog) |

## License

MIT
