package parser

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestParseAPT(t *testing.T) {
	input := `FROM debian:bookworm-slim
APT install curl ca-certificates
`
	insts, err := Parse(input)
	require.Nil(t, err)

	require.Equal(t, 2, len(insts))

	apt := insts[1]
	assert.Equal(t, "APT", apt.Command)

	require.Equal(t, 3, len(apt.Args))

	assert.Equal(t, "install", apt.Args[0])

	assert.Equal(t, "curl", apt.Args[1])

}

func TestParseLineContinuation(t *testing.T) {
	input := `FROM debian:bookworm-slim
RUN echo hello \
    world
`
	insts, err := Parse(input)
	require.Nil(t, err)

	require.Equal(t, 2, len(insts))

	run := insts[1]
	assert.Equal(t, "RUN", run.Command)

	// After continuation joining, args should contain both parts
	joined := joinArgs(run.Args)
	assert.Equal(t, "echo hello world", joined)

}

func TestParseSkipsComments(t *testing.T) {
	input := `# this is a comment
#syntax=something
FROM alpine:3.19
# another comment
RUN echo hi
`
	insts, err := Parse(input)
	require.Nil(t, err)

	require.Equal(t, 2, len(insts))

}

func TestParseEmptyInput(t *testing.T) {
	insts, err := Parse("")
	require.Nil(t, err)

	require.Equal(t, 0, len(insts))

}

func TestParseMultiStage(t *testing.T) {
	input := `FROM golang:1.22 AS builder
RUN go build -o /app
FROM alpine:3.19
COPY --from=builder /app /app
CMD ["/app"]
`
	insts, err := Parse(input)
	require.Nil(t, err)

	require.Equal(t, 5, len(insts))

	// Check COPY --from flag
	copyInst := insts[3]
	assert.Equal(t, "builder", copyInst.Flags["from"])

}

func TestParseUnknownInstruction(t *testing.T) {
	input := `FROM alpine
FOOBAR something
`
	_, err := Parse(input)
	require.NotNil(t, err)

}

func TestTokenizeQuotedStrings(t *testing.T) {
	input := `FROM alpine:3.19
LABEL "maintainer"="John Doe" 'version'='1.0'
`
	insts, err := Parse(input)
	require.Nil(t, err)
	require.Equal(t, 2, len(insts))
	assert.Equal(t, "LABEL", insts[1].Command)
}

func TestTokenizeEscapedQuote(t *testing.T) {
	input := `FROM alpine:3.19
RUN echo "hello \"world\""
`
	insts, err := Parse(input)
	require.Nil(t, err)
	require.Equal(t, 2, len(insts))
}

func TestTokenizeBrackets(t *testing.T) {
	input := `FROM alpine:3.19
CMD ["echo", "hello world"]
`
	insts, err := Parse(input)
	require.Nil(t, err)
	require.Equal(t, 2, len(insts))
	// The JSON array should be kept as a single token
	assert.Equal(t, 1, len(insts[1].Args))
}

func TestTokenizeTabSeparated(t *testing.T) {
	tokens := tokenize("hello\tworld")
	assert.Equal(t, 2, len(tokens))
}

func joinArgs(args []string) string {
	result := ""
	for i, a := range args {
		if i > 0 {
			result += " "
		}
		result += a
	}
	return result
}
