# docker-frontend

A custom [BuildKit](https://github.com/moby/buildkit) frontend that extends the Dockerfile DSL with additional instructions like `APT`.

## Usage

Point your Dockerfile's `syntax` directive at the frontend image:

```dockerfile
# syntax=ghcr.io/wow-look-at-my/docker-frontend:master
FROM ubuntu:22.04

APT install curl git jq

COPY . /app
WORKDIR /app
CMD ["./start.sh"]
```

Then build normally:

```bash
docker build -t myimage .
```

## Custom Instructions

### `APT install`

Installs packages via `apt-get` with persistent cache mounts so repeated builds skip re-downloading.

```dockerfile
APT install curl git
```

This expands to:

```bash
rm -f /etc/apt/apt.conf.d/docker-clean && \
  apt-get update && \
  apt-get install -y --no-install-recommends curl git
```

Cache mounts are added for `/var/cache/apt` and `/var/lib/apt` automatically.

## Project Structure

```
cmd/frontend/    Entry point (main.go)
pkg/parser/      Dockerfile DSL parser
pkg/converter/   Converts parsed AST into BuildKit LLB
pkg/instructions/ Instruction types
```

## Development

```bash
go build ./...
go test ./...
go vet ./...
```

Build the frontend image locally:

```bash
docker build -t docker-frontend .
```

## CI

CI runs on every push via GitHub Actions using [`wow-look-at-my/go-toolchain`](https://github.com/wow-look-at-my/go-toolchain). The Docker image is built and pushed to `ghcr.io/wow-look-at-my/docker-frontend`, tagged by branch name, git tag, and commit SHA.
