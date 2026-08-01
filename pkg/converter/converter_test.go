package converter

import (
	"strings"
	"testing"

	"github.com/wow-look-at-my/testify/assert"
	"github.com/wow-look-at-my/testify/require"
)

func TestPreprocessComprehensive(t *testing.T) {
	// Realistic multi-stage Dockerfile exercising standard instructions + custom APT.
	// Preprocess is pure string transformation — no Docker daemon, no images produced.
	input := `# syntax=wow-look-at-my/docker-frontend
# Build stage
FROM golang:1.22-bookworm AS builder

ARG APP_VERSION=1.0.0
ENV CGO_ENABLED=0

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -ldflags "-X main.version=${APP_VERSION}" -o /app

# Runtime stage
FROM debian:bookworm-slim

LABEL maintainer="team@example.com" version="1.0"

APT install curl ca-certificates

RUN groupadd -r appuser && useradd -r -g appuser appuser

COPY --from=builder /app /usr/local/bin/app

ADD https://github.com/example/config.git#main /etc/app/

WORKDIR /home/appuser
ENV APP_ENV=production \
    LOG_LEVEL=info

EXPOSE 8080/tcp
EXPOSE 9090

VOLUME /data /var/log/app

USER appuser

HEALTHCHECK --interval=30s --timeout=5s CMD curl -f http://localhost:8080/health || exit 1

STOPSIGNAL SIGTERM

ENTRYPOINT ["/usr/local/bin/app"]
CMD ["--config", "/etc/app/config.yaml"]
`
	result, err := Preprocess(input)
	require.Nil(t, err)

	lines := strings.Split(result, "\n")

	// Syntax directive stripped, other comments preserved
	assert.Equal(t, "# Build stage", lines[0])

	// All standard instructions pass through verbatim
	for _, want := range []string{
		"FROM golang:1.22-bookworm AS builder",
		"ARG APP_VERSION=1.0.0",
		"ENV CGO_ENABLED=0",
		"WORKDIR /src",
		"COPY go.mod go.sum ./",
		"RUN go mod download",
		"COPY . .",
		"FROM debian:bookworm-slim",
		`LABEL maintainer="team@example.com" version="1.0"`,
		"RUN groupadd -r appuser && useradd -r -g appuser appuser",
		"COPY --from=builder /app /usr/local/bin/app",
		"ADD https://github.com/example/config.git#main /etc/app/",
		"WORKDIR /home/appuser",
		"EXPOSE 8080/tcp",
		"EXPOSE 9090",
		"VOLUME /data /var/log/app",
		"USER appuser",
		"HEALTHCHECK --interval=30s --timeout=5s CMD curl -f http://localhost:8080/health || exit 1",
		"STOPSIGNAL SIGTERM",
		`ENTRYPOINT ["/usr/local/bin/app"]`,
		`CMD ["--config", "/etc/app/config.yaml"]`,
	} {
		assert.Contains(t, result, want, "expected passthrough of: %s", want)
	}

	// APT expanded correctly
	assert.NotContains(t, result, "APT install")
	assert.Contains(t, result, "RUN --mount=type=cache,target=/var/cache/apt,sharing=shared")
	assert.Contains(t, result, "--mount=type=cache,target=/var/lib/apt,sharing=shared")
	assert.Contains(t, result, "apt-get update && apt-get install -y --no-install-recommends curl ca-certificates")

	// Multi-line ENV with continuation should be joined
	assert.Contains(t, result, "ENV APP_ENV=production LOG_LEVEL=info")

	// RUN with -ldflags (contains build arg reference) passes through
	assert.Contains(t, result, `go build -ldflags "-X main.version=${APP_VERSION}" -o /app`)
}

func TestPreprocessAPTBadSubcommand(t *testing.T) {
	_, err := Preprocess("FROM debian:bookworm-slim\nAPT remove curl\n")
	require.NotNil(t, err)
	assert.Contains(t, err.Error(), "install")
}

func TestPreprocessAPTNoPackages(t *testing.T) {
	_, err := Preprocess("FROM debian:bookworm-slim\nAPT install\n")
	require.NotNil(t, err)
}

func TestPreprocessEmpty(t *testing.T) {
	result, err := Preprocess("")
	require.Nil(t, err)
	assert.Equal(t, "", result)
}

func TestPreprocessLineContinuation(t *testing.T) {
	input := "FROM debian:bookworm-slim\nAPT install curl \\\n    git \\\n    jq\n"
	result, err := Preprocess(input)
	require.Nil(t, err)
	assert.Contains(t, result, "apt-get install -y --no-install-recommends curl git jq")
	assert.NotContains(t, result, "APT")
}
