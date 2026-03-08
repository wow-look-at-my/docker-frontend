package parser

import (
	"testing"
)

func TestParseAPT(t *testing.T) {
	input := `FROM debian:bookworm-slim
APT install curl ca-certificates
`
	insts, err := Parse(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(insts) != 2 {
		t.Fatalf("expected 2 instructions, got %d", len(insts))
	}

	apt := insts[1]
	if apt.Command != "APT" {
		t.Errorf("expected command APT, got %s", apt.Command)
	}
	if len(apt.Args) != 3 {
		t.Fatalf("expected 3 args (install, curl, ca-certificates), got %d: %v", len(apt.Args), apt.Args)
	}
	if apt.Args[0] != "install" {
		t.Errorf("expected first arg 'install', got %s", apt.Args[0])
	}
	if apt.Args[1] != "curl" {
		t.Errorf("expected second arg 'curl', got %s", apt.Args[1])
	}
}

func TestParseLineContinuation(t *testing.T) {
	input := `FROM debian:bookworm-slim
RUN echo hello \
    world
`
	insts, err := Parse(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(insts) != 2 {
		t.Fatalf("expected 2 instructions, got %d", len(insts))
	}
	run := insts[1]
	if run.Command != "RUN" {
		t.Errorf("expected RUN, got %s", run.Command)
	}
	// After continuation joining, args should contain both parts
	joined := joinArgs(run.Args)
	if joined != "echo hello world" {
		t.Errorf("expected 'echo hello world', got %q", joined)
	}
}

func TestParseSkipsComments(t *testing.T) {
	input := `# this is a comment
#syntax=something
FROM alpine:3.19
# another comment
RUN echo hi
`
	insts, err := Parse(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(insts) != 2 {
		t.Fatalf("expected 2 instructions, got %d", len(insts))
	}
}

func TestParseEmptyInput(t *testing.T) {
	insts, err := Parse("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(insts) != 0 {
		t.Fatalf("expected 0 instructions, got %d", len(insts))
	}
}

func TestParseMultiStage(t *testing.T) {
	input := `FROM golang:1.22 AS builder
RUN go build -o /app
FROM alpine:3.19
COPY --from=builder /app /app
CMD ["/app"]
`
	insts, err := Parse(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(insts) != 5 {
		t.Fatalf("expected 5 instructions, got %d", len(insts))
	}
	// Check COPY --from flag
	copyInst := insts[3]
	if copyInst.Flags["from"] != "builder" {
		t.Errorf("expected --from=builder flag, got %v", copyInst.Flags)
	}
}

func TestParseUnknownInstruction(t *testing.T) {
	input := `FROM alpine
FOOBAR something
`
	_, err := Parse(input)
	if err == nil {
		t.Fatal("expected error for unknown instruction, got nil")
	}
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
