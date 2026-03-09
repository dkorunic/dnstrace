# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

This project uses [go-task](https://taskfile.dev/) for automation. Key tasks:

```sh
task build          # Format + build with PGO, trimpath, and version ldflags
task build-debug    # Format + build with -race (for debugging)
task lint           # Format + run golangci-lint
task fmt            # Run go mod tidy, gci, gofumpt, betteralign
task update         # go get -u + go mod tidy
task check          # gomajor list (check for major version updates)
task release        # goreleaser release --clean
```

Direct Go commands:

```sh
go build ./...
go vet ./...
golangci-lint run --timeout 5m
```

Run tests with `go test ./...`.

## Architecture

`dnstrace` emulates iterative DNS resolution (as a recursor would perform), traversing the DNS authoritative hierarchy from root servers down to the authoritative NS for the queried name.

**Package layout:**

- `main.go` — CLI flag parsing, entry point; calls `doDNSQuery` for each (qname, qtype) pair
- `resolver.go` — Core logic: `doDNSQuery` is a recursive function that walks the DNS delegation chain, handles CNAME following, UDP truncation fallback to EDNS/TCP, and accumulates RTT
- `hints/hints.go` — Hardcoded root nameserver hints (a–m.root-servers.net); `GetRand()` picks one randomly using a CSPRNG-backed `math/rand`
- `cache/cache.go` — Thread-safe in-memory A record cache (keyed by lowercase FQDN); `GetRand()` returns a random A record for a name to spread load; used to resolve glue records without re-querying
- `ipify.go` — Fetches the public IP from ipify.org when `-client=external` is specified

**Resolution flow in `doDNSQuery`:**
1. Check `aCache` for cached A records (avoids redundant lookups for glue)
2. Build and send DNS query (UDP by default, TCP if `-tcp`; EDNS OPT if `-edns` or `-client`)
3. On truncation/timeout, fall back: first to EDNS 4096, then TCP (controlled by `-fallback`)
4. On ANSWER: return if qtype matches; follow CNAME (in-zone continues with current NS, out-of-zone restarts from root)
5. On AUTHORITY (delegation): resolve NS glue from cache or by sub-query (`-ignoresub` skips sub-queries); recurse to next NS

**Global state** (`resolver.go`): `roots *hints.Hints` and `aCache *cache.Cache` are package-level vars initialized in `main()`.

## Build details

- `CGO_ENABLED=0` is used for production builds (`task build`); only `task build-debug` sets `CGO_ENABLED=1` (required for `-race`)
- Version info is injected via ldflags: `GitTag`, `GitCommit`, `GitDirty`, `BuildTime` in package `main`
- PGO (`-pgo=auto`) is used in production builds

## Linting

`golangci-lint` v2 config (`.golangci.yml`) enables all linters with selective disables. Formatters enforced: `gci`, `gofmt`, `gofumpt`, `goimports`. Run `task fmt` before committing to satisfy formatter checks.
