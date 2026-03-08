package converter

import (
	"testing"

	"github.com/wow-look-at-my/docker-frontend/pkg/instructions"
	"github.com/wow-look-at-my/docker-frontend/pkg/parser"
)

func TestConvertBasicFrom(t *testing.T) {
	insts := []instructions.Instruction{
		{Command: "FROM", Args: []string{"debian:bookworm-slim"}, Flags: map[string]string{}, Line: 1},
	}
	result, err := Convert(insts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestConvertAPT(t *testing.T) {
	insts := []instructions.Instruction{
		{Command: "FROM", Args: []string{"debian:bookworm-slim"}, Flags: map[string]string{}, Line: 1},
		{Command: "APT", Args: []string{"install", "curl", "ca-certificates"}, Flags: map[string]string{}, Line: 2},
	}
	result, err := Convert(insts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestConvertAPTBadSubcommand(t *testing.T) {
	insts := []instructions.Instruction{
		{Command: "FROM", Args: []string{"debian:bookworm-slim"}, Flags: map[string]string{}, Line: 1},
		{Command: "APT", Args: []string{"remove", "curl"}, Flags: map[string]string{}, Line: 2},
	}
	_, err := Convert(insts)
	if err == nil {
		t.Fatal("expected error for unsupported APT subcommand")
	}
}

func TestConvertWorkdirAndEnv(t *testing.T) {
	insts := []instructions.Instruction{
		{Command: "FROM", Args: []string{"alpine:3.19"}, Flags: map[string]string{}, Line: 1},
		{Command: "WORKDIR", Args: []string{"/app"}, Flags: map[string]string{}, Line: 2},
		{Command: "ENV", Args: []string{"MY_VAR", "hello"}, Flags: map[string]string{}, Line: 3},
	}
	result, err := Convert(insts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Image.Config.WorkingDir != "/app" {
		t.Errorf("expected workdir /app, got %s", result.Image.Config.WorkingDir)
	}
	if len(result.Image.Config.Env) != 1 || result.Image.Config.Env[0] != "MY_VAR=hello" {
		t.Errorf("expected ENV MY_VAR=hello, got %v", result.Image.Config.Env)
	}
}

func TestConvertCmd(t *testing.T) {
	insts := []instructions.Instruction{
		{Command: "FROM", Args: []string{"alpine:3.19"}, Flags: map[string]string{}, Line: 1},
		{Command: "CMD", Args: []string{`["/bin/sh", "-c", "echo hello"]`}, Flags: map[string]string{}, Line: 2},
	}
	result, err := Convert(insts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Image.Config.Cmd) != 3 {
		t.Fatalf("expected 3 cmd args, got %d: %v", len(result.Image.Config.Cmd), result.Image.Config.Cmd)
	}
	if result.Image.Config.Cmd[0] != "/bin/sh" {
		t.Errorf("expected /bin/sh, got %s", result.Image.Config.Cmd[0])
	}
}

func TestConvertMultiStage(t *testing.T) {
	insts := []instructions.Instruction{
		{Command: "FROM", Args: []string{"golang:1.22", "AS", "builder"}, Flags: map[string]string{}, Line: 1},
		{Command: "RUN", Args: []string{"go", "build", "-o", "/app"}, Flags: map[string]string{}, Line: 2},
		{Command: "FROM", Args: []string{"alpine:3.19"}, Flags: map[string]string{}, Line: 3},
		{Command: "COPY", Args: []string{"/app", "/app"}, Flags: map[string]string{"from": "builder"}, Line: 4},
	}
	result, err := Convert(insts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, exists := result.Stages["builder"]; !exists {
		t.Error("expected 'builder' stage to exist")
	}
}

func TestConvertExpose(t *testing.T) {
	insts := []instructions.Instruction{
		{Command: "FROM", Args: []string{"alpine:3.19"}, Flags: map[string]string{}, Line: 1},
		{Command: "EXPOSE", Args: []string{"8080", "443"}, Flags: map[string]string{}, Line: 2},
	}
	result, err := Convert(insts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Image.Config.ExposedPorts) != 2 {
		t.Errorf("expected 2 exposed ports, got %d", len(result.Image.Config.ExposedPorts))
	}
}

func TestFullPipelineParseAndConvert(t *testing.T) {
	input := `FROM debian:bookworm-slim
APT install curl ca-certificates
WORKDIR /app
ENV APP_ENV production
EXPOSE 8080
CMD ["./myapp"]
`
	insts, err := parser.Parse(input)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	result, err := Convert(insts)
	if err != nil {
		t.Fatalf("convert error: %v", err)
	}
	if result.Image.Config.WorkingDir != "/app" {
		t.Errorf("expected workdir /app, got %s", result.Image.Config.WorkingDir)
	}
	if len(result.Image.Config.Cmd) != 1 || result.Image.Config.Cmd[0] != "./myapp" {
		t.Errorf("expected cmd [./myapp], got %v", result.Image.Config.Cmd)
	}
}
