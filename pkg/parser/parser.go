package parser

import (
	"fmt"
	"strings"

	"github.com/wow-look-at-my/docker-frontend/pkg/instructions"
)

var knownCommands = map[string]bool{
	"FROM": true, "RUN": true, "COPY": true, "ADD": true,
	"WORKDIR": true, "ENV": true, "ARG": true, "EXPOSE": true,
	"CMD": true, "ENTRYPOINT": true, "USER": true, "VOLUME": true,
	"LABEL": true, "SHELL": true, "STOPSIGNAL": true, "HEALTHCHECK": true,
	"ONBUILD": true, "MAINTAINER": true,
	// Custom instructions
	"APT": true,
}

// Parse parses the Dockerfile DSL content into a list of instructions.
func Parse(content string) ([]instructions.Instruction, error) {
	lines := strings.Split(content, "\n")
	var result []instructions.Instruction

	i := 0
	for i < len(lines) {
		line := strings.TrimSpace(lines[i])

		// Skip empty lines, comments, and syntax directive
		if line == "" || strings.HasPrefix(line, "#") {
			i++
			continue
		}

		// Handle line continuations
		fullLine := line
		lineNum := i + 1
		for strings.HasSuffix(fullLine, "\\") {
			fullLine = strings.TrimSuffix(fullLine, "\\")
			fullLine = strings.TrimRight(fullLine, " \t")
			i++
			if i < len(lines) {
				fullLine += " " + strings.TrimSpace(lines[i])
			}
		}
		i++

		inst, err := parseLine(fullLine, lineNum)
		if err != nil {
			return nil, err
		}
		if inst != nil {
			result = append(result, *inst)
		}
	}

	return result, nil
}

func parseLine(line string, lineNum int) (*instructions.Instruction, error) {
	// Split into command and rest
	parts := strings.SplitN(line, " ", 2)
	command := strings.ToUpper(parts[0])

	if !knownCommands[command] {
		return nil, fmt.Errorf("line %d: unknown instruction %q", lineNum, parts[0])
	}

	rest := ""
	if len(parts) > 1 {
		rest = parts[1]
	}

	flags := make(map[string]string)
	var args []string

	// Parse flags and args from rest
	tokens := tokenize(rest)
	for _, tok := range tokens {
		if strings.HasPrefix(tok, "--") && strings.Contains(tok, "=") {
			kv := strings.SplitN(tok[2:], "=", 2)
			flags[kv[0]] = kv[1]
		} else {
			args = append(args, tok)
		}
	}

	return &instructions.Instruction{
		Command: command,
		Args:    args,
		Flags:   flags,
		Line:    lineNum,
	}, nil
}

// tokenize splits a string by whitespace, respecting quoted strings and JSON arrays.
func tokenize(s string) []string {
	var tokens []string
	var current strings.Builder
	inQuote := false
	quoteChar := byte(0)
	bracketDepth := 0

	for i := 0; i < len(s); i++ {
		ch := s[i]

		if inQuote {
			current.WriteByte(ch)
			if ch == quoteChar && (i == 0 || s[i-1] != '\\') {
				inQuote = false
			}
			continue
		}

		if ch == '[' {
			bracketDepth++
			current.WriteByte(ch)
			continue
		}
		if ch == ']' {
			bracketDepth--
			current.WriteByte(ch)
			continue
		}

		if bracketDepth > 0 {
			current.WriteByte(ch)
			continue
		}

		if ch == '"' || ch == '\'' {
			inQuote = true
			quoteChar = ch
			current.WriteByte(ch)
			continue
		}

		if ch == ' ' || ch == '\t' {
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
			continue
		}

		current.WriteByte(ch)
	}

	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}

	return tokens
}
