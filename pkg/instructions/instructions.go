package instructions

// Instruction represents a single parsed instruction from the Dockerfile DSL.
type Instruction struct {
	Command string
	Args    []string
	Flags   map[string]string
	Line    int
}
