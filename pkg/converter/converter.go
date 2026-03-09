package converter

import (
	"fmt"
	"strings"
)

// Preprocess takes raw Dockerfile content and expands custom instructions
// (like APT) into standard Dockerfile syntax. All standard Dockerfile
// instructions are passed through unchanged so that BuildKit's built-in
// dockerfile frontend handles them.
func Preprocess(content string) (string, error) {
	lines := strings.Split(content, "\n")
	var out []string

	i := 0
	for i < len(lines) {
		line := strings.TrimSpace(lines[i])

		// Pass through empty lines and comments unchanged
		if line == "" || strings.HasPrefix(line, "#") {
			out = append(out, lines[i])
			i++
			continue
		}

		// Join continuation lines
		fullLine := line
		firstIdx := i
		for strings.HasSuffix(fullLine, "\\") {
			fullLine = strings.TrimSuffix(fullLine, "\\")
			fullLine = strings.TrimRight(fullLine, " \t")
			i++
			if i < len(lines) {
				fullLine += " " + strings.TrimSpace(lines[i])
			}
		}
		i++

		// Check if this is a custom instruction
		parts := strings.SplitN(fullLine, " ", 2)
		command := strings.ToUpper(parts[0])

		switch command {
		case "APT":
			rest := ""
			if len(parts) > 1 {
				rest = parts[1]
			}
			expanded, err := expandAPT(rest, firstIdx+1)
			if err != nil {
				return "", err
			}
			out = append(out, expanded)
		default:
			// Pass through unchanged — let BuildKit handle it
			out = append(out, fullLine)
		}
	}

	return strings.Join(out, "\n"), nil
}

// expandAPT converts "APT install pkg1 pkg2 ..." into a standard RUN instruction
// with cache mounts for apt.
func expandAPT(args string, lineNum int) (string, error) {
	tokens := strings.Fields(args)
	if len(tokens) < 2 || strings.ToLower(tokens[0]) != "install" {
		return "", fmt.Errorf("line %d: APT currently only supports 'install' subcommand (e.g., APT install curl git)", lineNum)
	}

	packages := tokens[1:]

	return fmt.Sprintf(
		"RUN --mount=type=cache,target=/var/cache/apt,sharing=shared --mount=type=cache,target=/var/lib/apt,sharing=shared rm -f /etc/apt/apt.conf.d/docker-clean && apt-get update && apt-get install -y --no-install-recommends %s",
		strings.Join(packages, " "),
	), nil
}
