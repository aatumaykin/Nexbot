package workspace

import (
	_ "embed"
)

//go:embed embeddefaults/main/IDENTITY.md
var defaultIdentity []byte

//go:embed embeddefaults/main/AGENTS.md
var defaultAgents []byte

//go:embed embeddefaults/main/TOOLS.md
var defaultTools []byte

//go:embed embeddefaults/main/USER.md
var defaultUser []byte

//go:embed embeddefaults/subagent/AGENTS.md
var defaultSubagentAgents []byte

// DefaultBootstrapContent provides fallback content for bootstrap files
// when they are not found in the workspace directory.
type DefaultBootstrapContent struct {
	Identity       string
	Agents         string
	Tools          string
	User           string
	SubagentAgents string
}

// GetDefaultBootstrap returns the default bootstrap content.
func GetDefaultBootstrap() DefaultBootstrapContent {
	return DefaultBootstrapContent{
		Identity:       string(defaultIdentity),
		Agents:         string(defaultAgents),
		Tools:          string(defaultTools),
		User:           string(defaultUser),
		SubagentAgents: string(defaultSubagentAgents),
	}
}

// GetDefaultFile returns the default content for a specific bootstrap file.
// Returns empty string if the file is not recognized.
func GetDefaultFile(filename string) string {
	bootstrap := GetDefaultBootstrap()
	switch filename {
	case BootstrapIdentity:
		return bootstrap.Identity
	case BootstrapAgents:
		return bootstrap.Agents
	case BootstrapTools:
		return bootstrap.Tools
	case BootstrapUser:
		return bootstrap.User
	default:
		return ""
	}
}

// GetDefaultSubagentFile returns the default content for a subagent bootstrap file.
// Returns empty string if the file is not recognized.
func GetDefaultSubagentFile(filename string) string {
	bootstrap := GetDefaultBootstrap()
	switch filename {
	case "AGENTS.md":
		return bootstrap.SubagentAgents
	default:
		return ""
	}
}
