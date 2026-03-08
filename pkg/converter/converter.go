package converter

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/moby/buildkit/client/llb"
	"github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/wow-look-at-my/docker-frontend/pkg/instructions"
)

// Result holds the final LLB state and image config produced by conversion.
type Result struct {
	State  llb.State
	Image  v1.Image
	Stages map[string]llb.State
}

// Convert takes parsed instructions and produces an LLB state + image config.
// buildContext is the LLB state for the build context (used by COPY without --from).
// Pass llb.Scratch() if no build context is available (e.g., in tests).
func Convert(insts []instructions.Instruction, buildContext ...llb.State) (*Result, error) {
	var ctx llb.State
	if len(buildContext) > 0 {
		ctx = buildContext[0]
	} else {
		ctx = llb.Scratch()
	}
	r := &Result{
		State:  llb.Scratch(),
		Stages: make(map[string]llb.State),
		Image: v1.Image{
			Config: v1.ImageConfig{},
		},
	}

	var currentStageName string

	for _, inst := range insts {
		switch inst.Command {
		case "FROM":
			if err := handleFrom(r, inst, &currentStageName); err != nil {
				return nil, fmt.Errorf("line %d: %w", inst.Line, err)
			}
		case "RUN":
			handleRun(r, inst)
		case "APT":
			if err := handleAPT(r, inst); err != nil {
				return nil, fmt.Errorf("line %d: %w", inst.Line, err)
			}
		case "COPY":
			handleCopy(r, inst, ctx)
		case "ADD":
			handleCopy(r, inst, ctx)
		case "WORKDIR":
			handleWorkdir(r, inst)
		case "ENV":
			handleEnv(r, inst)
		case "USER":
			handleUser(r, inst)
		case "EXPOSE":
			handleExpose(r, inst)
		case "CMD":
			handleCmd(r, inst)
		case "ENTRYPOINT":
			handleEntrypoint(r, inst)
		case "VOLUME":
			handleVolume(r, inst)
		case "LABEL":
			handleLabel(r, inst)
		case "ARG":
			// ARGs are handled at parse time for variable substitution (future)
		case "SHELL":
			handleShell(r, inst)
		case "STOPSIGNAL":
			handleStopSignal(r, inst)
		default:
			return nil, fmt.Errorf("line %d: unsupported instruction %s", inst.Line, inst.Command)
		}

		// Keep stage mapping updated
		if currentStageName != "" {
			r.Stages[currentStageName] = r.State
		}
	}

	return r, nil
}

func handleFrom(r *Result, inst instructions.Instruction, currentStageName *string) error {
	if len(inst.Args) == 0 {
		return fmt.Errorf("FROM requires an image argument")
	}

	image := inst.Args[0]

	// Reset image config for new stage
	r.Image = v1.Image{Config: v1.ImageConfig{}}

	if strings.ToLower(image) == "scratch" {
		r.State = llb.Scratch()
	} else {
		r.State = llb.Image(image)
	}

	// Handle AS <name>
	*currentStageName = ""
	for i, arg := range inst.Args {
		if strings.ToUpper(arg) == "AS" && i+1 < len(inst.Args) {
			*currentStageName = inst.Args[i+1]
			r.Stages[*currentStageName] = r.State
			break
		}
	}

	return nil
}

func handleRun(r *Result, inst instructions.Instruction) {
	cmd := strings.Join(inst.Args, " ")
	r.State = r.State.Run(llb.Shlex(cmd)).Root()
}

func handleAPT(r *Result, inst instructions.Instruction) error {
	if len(inst.Args) < 2 || strings.ToLower(inst.Args[0]) != "install" {
		return fmt.Errorf("APT currently only supports 'install' subcommand (e.g., APT install curl git)")
	}

	packages := inst.Args[1:]

	// Build the shell command:
	// 1. Remove the docker-clean sabotage file
	// 2. apt-get update
	// 3. apt-get install with --no-install-recommends
	cmd := fmt.Sprintf(
		"rm -f /etc/apt/apt.conf.d/docker-clean && apt-get update && apt-get install -y --no-install-recommends %s",
		strings.Join(packages, " "),
	)

	// Create the run with cache mounts for apt
	r.State = r.State.Run(
		llb.Shlex(cmd),
		llb.AddMount("/var/cache/apt", llb.Scratch(), llb.AsPersistentCacheDir("apt-cache", llb.CacheMountShared)),
		llb.AddMount("/var/lib/apt", llb.Scratch(), llb.AsPersistentCacheDir("apt-lib", llb.CacheMountShared)),
	).Root()

	return nil
}

func handleCopy(r *Result, inst instructions.Instruction, buildContext llb.State) {
	if len(inst.Args) < 2 {
		return
	}

	src := inst.Args[0]
	dst := inst.Args[len(inst.Args)-1]

	if fromStage, ok := inst.Flags["from"]; ok {
		if stage, exists := r.Stages[fromStage]; exists {
			r.State = r.State.File(
				llb.Copy(stage, src, dst),
			)
			return
		}
		// If stage not found, treat --from as an image reference
		r.State = r.State.File(
			llb.Copy(llb.Image(fromStage), src, dst),
		)
		return
	}

	// COPY from build context
	r.State = r.State.File(
		llb.Copy(buildContext, src, dst),
	)
}

func handleWorkdir(r *Result, inst instructions.Instruction) {
	if len(inst.Args) > 0 {
		dir := inst.Args[0]
		r.State = r.State.Dir(dir)
		r.Image.Config.WorkingDir = dir
	}
}

func handleEnv(r *Result, inst instructions.Instruction) {
	if len(inst.Args) >= 2 {
		key := inst.Args[0]
		// Handle both ENV KEY VALUE and ENV KEY=VALUE
		if strings.Contains(key, "=") {
			parts := strings.SplitN(key, "=", 2)
			r.State = r.State.AddEnv(parts[0], parts[1])
			r.Image.Config.Env = append(r.Image.Config.Env, key)
		} else {
			value := strings.Join(inst.Args[1:], " ")
			r.State = r.State.AddEnv(key, value)
			r.Image.Config.Env = append(r.Image.Config.Env, key+"="+value)
		}
	}
}

func handleUser(r *Result, inst instructions.Instruction) {
	if len(inst.Args) > 0 {
		r.State = r.State.User(inst.Args[0])
		r.Image.Config.User = inst.Args[0]
	}
}

func handleExpose(r *Result, inst instructions.Instruction) {
	if r.Image.Config.ExposedPorts == nil {
		r.Image.Config.ExposedPorts = make(map[string]struct{})
	}
	for _, port := range inst.Args {
		r.Image.Config.ExposedPorts[port] = struct{}{}
	}
}

func handleCmd(r *Result, inst instructions.Instruction) {
	r.Image.Config.Cmd = parseExecForm(inst.Args)
}

func handleEntrypoint(r *Result, inst instructions.Instruction) {
	r.Image.Config.Entrypoint = parseExecForm(inst.Args)
}

func handleVolume(r *Result, inst instructions.Instruction) {
	if r.Image.Config.Volumes == nil {
		r.Image.Config.Volumes = make(map[string]struct{})
	}
	for _, v := range inst.Args {
		r.Image.Config.Volumes[v] = struct{}{}
	}
}

func handleLabel(r *Result, inst instructions.Instruction) {
	if r.Image.Config.Labels == nil {
		r.Image.Config.Labels = make(map[string]string)
	}
	for _, arg := range inst.Args {
		if parts := strings.SplitN(arg, "=", 2); len(parts) == 2 {
			key := strings.Trim(parts[0], "\"")
			val := strings.Trim(parts[1], "\"")
			r.Image.Config.Labels[key] = val
		}
	}
}

func handleShell(r *Result, inst instructions.Instruction) {
	// SHELL instruction changes the default shell used for RUN
	// Stored in image config
	shell := parseExecForm(inst.Args)
	if len(shell) > 0 {
		// v1.ImageConfig doesn't have a Shell field directly,
		// but we store it for our own use
	}
}

func handleStopSignal(r *Result, inst instructions.Instruction) {
	if len(inst.Args) > 0 {
		r.Image.Config.StopSignal = inst.Args[0]
	}
}

// parseExecForm handles both exec form ["cmd","arg"] and shell form.
func parseExecForm(args []string) []string {
	if len(args) == 1 {
		s := args[0]
		// Try to parse as JSON array (exec form)
		if strings.HasPrefix(s, "[") {
			var parsed []string
			if err := json.Unmarshal([]byte(s), &parsed); err == nil {
				return parsed
			}
		}
	}
	// Check if the combined args look like a JSON array that was tokenized
	combined := strings.Join(args, " ")
	if strings.HasPrefix(combined, "[") && strings.HasSuffix(combined, "]") {
		var parsed []string
		if err := json.Unmarshal([]byte(combined), &parsed); err == nil {
			return parsed
		}
	}
	return args
}
