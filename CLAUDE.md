# CLAUDE.md

## Project

Custom Docker BuildKit frontend ("Dockerbetter") — a Go binary that acts as a BuildKit frontend, enabling custom DSL instructions like `APT install`.

## Build & Test

- **CI uses `wow-look-at-my/go-toolchain@latest`** — do NOT use raw `go build`/`go test`/`go vet` in GitHub Actions. Always use the go-toolchain action.
- Local dev: `go build ./...`, `go test ./...`, `go vet ./...`
- Docker image: `docker build -t docker-frontend .`

## Structure

- `cmd/frontend/` — entry point (`main.go`)
- `pkg/parser/` — Dockerfile DSL parser
- `pkg/builder/` — BuildKit build function (reads entrypoint, orchestrates parse + convert)
- `pkg/converter/` — converts parsed AST into LLB state chains
- `Dockerfile` — builds the frontend image (two-stage: golang builder → alpine runtime)

## Key Dependencies

- `github.com/moby/buildkit` — BuildKit LLB and gateway client
- `github.com/opencontainers/image-spec` — OCI image spec types
