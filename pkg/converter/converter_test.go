package converter

import (
	"testing"

	"github.com/wow-look-at-my/docker-frontend/pkg/instructions"
	"github.com/wow-look-at-my/testify/assert"
	"github.com/wow-look-at-my/testify/require"
	"github.com/wow-look-at-my/docker-frontend/pkg/parser"
)

func TestConvertBasicFrom(t *testing.T) {
	insts := []instructions.Instruction{
		{Command: "FROM", Args: []string{"debian:bookworm-slim"}, Flags: map[string]string{}, Line: 1},
	}
	result, err := Convert(insts)
	require.Nil(t, err)

	require.NotNil(t, result)

}

func TestConvertAPT(t *testing.T) {
	insts := []instructions.Instruction{
		{Command: "FROM", Args: []string{"debian:bookworm-slim"}, Flags: map[string]string{}, Line: 1},
		{Command: "APT", Args: []string{"install", "curl", "ca-certificates"}, Flags: map[string]string{}, Line: 2},
	}
	result, err := Convert(insts)
	require.Nil(t, err)

	require.NotNil(t, result)

}

func TestConvertAPTBadSubcommand(t *testing.T) {
	insts := []instructions.Instruction{
		{Command: "FROM", Args: []string{"debian:bookworm-slim"}, Flags: map[string]string{}, Line: 1},
		{Command: "APT", Args: []string{"remove", "curl"}, Flags: map[string]string{}, Line: 2},
	}
	_, err := Convert(insts)
	require.NotNil(t, err)

}

func TestConvertWorkdirAndEnv(t *testing.T) {
	insts := []instructions.Instruction{
		{Command: "FROM", Args: []string{"alpine:3.19"}, Flags: map[string]string{}, Line: 1},
		{Command: "WORKDIR", Args: []string{"/app"}, Flags: map[string]string{}, Line: 2},
		{Command: "ENV", Args: []string{"MY_VAR", "hello"}, Flags: map[string]string{}, Line: 3},
	}
	result, err := Convert(insts)
	require.Nil(t, err)

	assert.Equal(t, "/app", result.Image.Config.WorkingDir)

	assert.False(t, len(result.Image.Config.Env) != 1 || result.Image.Config.Env[0] != "MY_VAR=hello")

}

func TestConvertCmd(t *testing.T) {
	insts := []instructions.Instruction{
		{Command: "FROM", Args: []string{"alpine:3.19"}, Flags: map[string]string{}, Line: 1},
		{Command: "CMD", Args: []string{`["/bin/sh", "-c", "echo hello"]`}, Flags: map[string]string{}, Line: 2},
	}
	result, err := Convert(insts)
	require.Nil(t, err)

	require.Equal(t, 3, len(result.Image.Config.Cmd))

	assert.Equal(t, "/bin/sh", result.Image.Config.Cmd[0])

}

func TestConvertMultiStage(t *testing.T) {
	insts := []instructions.Instruction{
		{Command: "FROM", Args: []string{"golang:1.22", "AS", "builder"}, Flags: map[string]string{}, Line: 1},
		{Command: "RUN", Args: []string{"go", "build", "-o", "/app"}, Flags: map[string]string{}, Line: 2},
		{Command: "FROM", Args: []string{"alpine:3.19"}, Flags: map[string]string{}, Line: 3},
		{Command: "COPY", Args: []string{"/app", "/app"}, Flags: map[string]string{"from": "builder"}, Line: 4},
	}
	result, err := Convert(insts)
	require.Nil(t, err)

	_, exists := result.Stages["builder"]
	assert.True(t, exists)

}

func TestConvertExpose(t *testing.T) {
	insts := []instructions.Instruction{
		{Command: "FROM", Args: []string{"alpine:3.19"}, Flags: map[string]string{}, Line: 1},
		{Command: "EXPOSE", Args: []string{"8080", "443"}, Flags: map[string]string{}, Line: 2},
	}
	result, err := Convert(insts)
	require.Nil(t, err)

	assert.Equal(t, 2, len(result.Image.Config.ExposedPorts))

}

func TestConvertUser(t *testing.T) {
	insts := []instructions.Instruction{
		{Command: "FROM", Args: []string{"alpine:3.19"}, Flags: map[string]string{}, Line: 1},
		{Command: "USER", Args: []string{"nobody"}, Flags: map[string]string{}, Line: 2},
	}
	result, err := Convert(insts)
	require.Nil(t, err)
	assert.Equal(t, "nobody", result.Image.Config.User)
}

func TestConvertVolume(t *testing.T) {
	insts := []instructions.Instruction{
		{Command: "FROM", Args: []string{"alpine:3.19"}, Flags: map[string]string{}, Line: 1},
		{Command: "VOLUME", Args: []string{"/data", "/logs"}, Flags: map[string]string{}, Line: 2},
	}
	result, err := Convert(insts)
	require.Nil(t, err)
	assert.Equal(t, 2, len(result.Image.Config.Volumes))
	_, hasData := result.Image.Config.Volumes["/data"]
	assert.True(t, hasData)
	_, hasLogs := result.Image.Config.Volumes["/logs"]
	assert.True(t, hasLogs)
}

func TestConvertLabel(t *testing.T) {
	insts := []instructions.Instruction{
		{Command: "FROM", Args: []string{"alpine:3.19"}, Flags: map[string]string{}, Line: 1},
		{Command: "LABEL", Args: []string{"maintainer=test@example.com", "version=1.0"}, Flags: map[string]string{}, Line: 2},
	}
	result, err := Convert(insts)
	require.Nil(t, err)
	assert.Equal(t, "test@example.com", result.Image.Config.Labels["maintainer"])
	assert.Equal(t, "1.0", result.Image.Config.Labels["version"])
}

func TestConvertEntrypoint(t *testing.T) {
	insts := []instructions.Instruction{
		{Command: "FROM", Args: []string{"alpine:3.19"}, Flags: map[string]string{}, Line: 1},
		{Command: "ENTRYPOINT", Args: []string{`["/usr/bin/app"]`}, Flags: map[string]string{}, Line: 2},
	}
	result, err := Convert(insts)
	require.Nil(t, err)
	require.Equal(t, 1, len(result.Image.Config.Entrypoint))
	assert.Equal(t, "/usr/bin/app", result.Image.Config.Entrypoint[0])
}

func TestConvertStopSignal(t *testing.T) {
	insts := []instructions.Instruction{
		{Command: "FROM", Args: []string{"alpine:3.19"}, Flags: map[string]string{}, Line: 1},
		{Command: "STOPSIGNAL", Args: []string{"SIGTERM"}, Flags: map[string]string{}, Line: 2},
	}
	result, err := Convert(insts)
	require.Nil(t, err)
	assert.Equal(t, "SIGTERM", result.Image.Config.StopSignal)
}

func TestConvertFromScratch(t *testing.T) {
	insts := []instructions.Instruction{
		{Command: "FROM", Args: []string{"scratch"}, Flags: map[string]string{}, Line: 1},
	}
	result, err := Convert(insts)
	require.Nil(t, err)
	require.NotNil(t, result)
}

func TestConvertFromNoArgs(t *testing.T) {
	insts := []instructions.Instruction{
		{Command: "FROM", Args: []string{}, Flags: map[string]string{}, Line: 1},
	}
	_, err := Convert(insts)
	require.NotNil(t, err)
}

func TestConvertUnsupportedInstruction(t *testing.T) {
	insts := []instructions.Instruction{
		{Command: "FROM", Args: []string{"alpine:3.19"}, Flags: map[string]string{}, Line: 1},
		{Command: "HEALTHCHECK", Args: []string{"CMD", "curl", "-f", "http://localhost/"}, Flags: map[string]string{}, Line: 2},
	}
	_, err := Convert(insts)
	require.NotNil(t, err)
}

func TestConvertCopyFromImage(t *testing.T) {
	insts := []instructions.Instruction{
		{Command: "FROM", Args: []string{"alpine:3.19"}, Flags: map[string]string{}, Line: 1},
		{Command: "COPY", Args: []string{"/bin/busybox", "/usr/bin/busybox"}, Flags: map[string]string{"from": "busybox:latest"}, Line: 2},
	}
	result, err := Convert(insts)
	require.Nil(t, err)
	require.NotNil(t, result)
}

func TestConvertCopyTooFewArgs(t *testing.T) {
	insts := []instructions.Instruction{
		{Command: "FROM", Args: []string{"alpine:3.19"}, Flags: map[string]string{}, Line: 1},
		{Command: "COPY", Args: []string{"onlyone"}, Flags: map[string]string{}, Line: 2},
	}
	result, err := Convert(insts)
	require.Nil(t, err)
	require.NotNil(t, result)
}

func TestConvertEnvKeyValueForm(t *testing.T) {
	insts := []instructions.Instruction{
		{Command: "FROM", Args: []string{"alpine:3.19"}, Flags: map[string]string{}, Line: 1},
		{Command: "ENV", Args: []string{"KEY=value", ""}, Flags: map[string]string{}, Line: 2},
	}
	result, err := Convert(insts)
	require.Nil(t, err)
	require.Equal(t, 1, len(result.Image.Config.Env))
	assert.Equal(t, "KEY=value", result.Image.Config.Env[0])
}

func TestConvertShell(t *testing.T) {
	insts := []instructions.Instruction{
		{Command: "FROM", Args: []string{"alpine:3.19"}, Flags: map[string]string{}, Line: 1},
		{Command: "SHELL", Args: []string{`["/bin/bash", "-c"]`}, Flags: map[string]string{}, Line: 2},
	}
	result, err := Convert(insts)
	require.Nil(t, err)
	require.NotNil(t, result)
}

func TestConvertArg(t *testing.T) {
	insts := []instructions.Instruction{
		{Command: "FROM", Args: []string{"alpine:3.19"}, Flags: map[string]string{}, Line: 1},
		{Command: "ARG", Args: []string{"VERSION=1.0"}, Flags: map[string]string{}, Line: 2},
	}
	result, err := Convert(insts)
	require.Nil(t, err)
	require.NotNil(t, result)
}

func TestParseExecFormTokenized(t *testing.T) {
	// Test the branch where JSON array is split across multiple args
	args := []string{`["echo",`, `"hello"]`}
	result := parseExecForm(args)
	assert.Equal(t, 2, len(result))
	assert.Equal(t, "echo", result[0])
	assert.Equal(t, "hello", result[1])
}

func TestParseExecFormShellForm(t *testing.T) {
	args := []string{"echo", "hello"}
	result := parseExecForm(args)
	assert.Equal(t, 2, len(result))
	assert.Equal(t, "echo", result[0])
}

func TestConvertWithBuildContext(t *testing.T) {
	insts := []instructions.Instruction{
		{Command: "FROM", Args: []string{"alpine:3.19"}, Flags: map[string]string{}, Line: 1},
		{Command: "COPY", Args: []string{".", "/app"}, Flags: map[string]string{}, Line: 2},
	}
	result, err := Convert(insts)
	require.Nil(t, err)
	require.NotNil(t, result)
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
	require.Nil(t, err)

	result, err := Convert(insts)
	require.Nil(t, err)

	assert.Equal(t, "/app", result.Image.Config.WorkingDir)

	assert.False(t, len(result.Image.Config.Cmd) != 1 || result.Image.Config.Cmd[0] != "./myapp")

}
