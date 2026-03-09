package converter

import (
	"strings"
	"testing"

	"github.com/wow-look-at-my/testify/assert"
	"github.com/wow-look-at-my/testify/require"
)

func TestPreprocessPassthroughStandard(t *testing.T) {
	input := `FROM debian:bookworm-slim
RUN echo hello
COPY . /app
WORKDIR /app
ENV APP_ENV=production
EXPOSE 8080
CMD ["./myapp"]
`
	result, err := Preprocess(input)
	require.Nil(t, err)
	// Standard instructions should pass through unchanged
	assert.Equal(t, input, result)
}

func TestPreprocessAPT(t *testing.T) {
	input := `FROM debian:bookworm-slim
APT install curl ca-certificates
`
	result, err := Preprocess(input)
	require.Nil(t, err)

	assert.Contains(t, result, "FROM debian:bookworm-slim")
	assert.Contains(t, result, "RUN --mount=type=cache,target=/var/cache/apt,sharing=shared")
	assert.Contains(t, result, "--mount=type=cache,target=/var/lib/apt,sharing=shared")
	assert.Contains(t, result, "apt-get install -y --no-install-recommends curl ca-certificates")
	assert.NotContains(t, result, "APT")
}

func TestPreprocessAPTBadSubcommand(t *testing.T) {
	input := `FROM debian:bookworm-slim
APT remove curl
`
	_, err := Preprocess(input)
	require.NotNil(t, err)
}

func TestPreprocessComments(t *testing.T) {
	input := `# this is a comment
#syntax=something
FROM alpine:3.19
# another comment
RUN echo hi
`
	result, err := Preprocess(input)
	require.Nil(t, err)
	assert.Contains(t, result, "# this is a comment")
	assert.Contains(t, result, "#syntax=something")
	assert.Contains(t, result, "# another comment")
}

func TestPreprocessEmpty(t *testing.T) {
	result, err := Preprocess("")
	require.Nil(t, err)
	assert.Equal(t, "", result)
}

func TestPreprocessMultiStage(t *testing.T) {
	input := `FROM golang:1.22 AS builder
RUN go build -o /app
FROM alpine:3.19
COPY --from=builder /app /app
CMD ["/app"]
`
	result, err := Preprocess(input)
	require.Nil(t, err)
	// Multi-stage instructions pass through unchanged
	assert.Contains(t, result, "FROM golang:1.22 AS builder")
	assert.Contains(t, result, "COPY --from=builder /app /app")
}

func TestPreprocessADDGitURL(t *testing.T) {
	input := `FROM debian:bookworm-slim
ADD https://github.com/example/repo.git /src
`
	result, err := Preprocess(input)
	require.Nil(t, err)
	// ADD with git URL should pass through to the built-in frontend
	assert.Contains(t, result, "ADD https://github.com/example/repo.git /src")
}

func TestPreprocessHEALTHCHECK(t *testing.T) {
	input := `FROM alpine:3.19
HEALTHCHECK CMD curl -f http://localhost/
`
	result, err := Preprocess(input)
	require.Nil(t, err)
	// HEALTHCHECK should pass through to the built-in frontend
	assert.Contains(t, result, "HEALTHCHECK CMD curl -f http://localhost/")
}

func TestPreprocessLineContinuation(t *testing.T) {
	input := "FROM debian:bookworm-slim\nAPT install curl \\\n    git \\\n    jq\n"
	result, err := Preprocess(input)
	require.Nil(t, err)
	assert.Contains(t, result, "apt-get install -y --no-install-recommends curl git jq")
	assert.NotContains(t, result, "APT")
}

func TestPreprocessFullPipeline(t *testing.T) {
	input := `FROM debian:bookworm-slim
APT install curl ca-certificates
WORKDIR /app
ENV APP_ENV production
EXPOSE 8080
CMD ["./myapp"]
`
	result, err := Preprocess(input)
	require.Nil(t, err)

	lines := strings.Split(result, "\n")
	assert.Equal(t, "FROM debian:bookworm-slim", lines[0])
	assert.Contains(t, lines[1], "RUN --mount=type=cache")
	assert.Equal(t, "WORKDIR /app", lines[2])
}
