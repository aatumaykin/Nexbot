package version

import "fmt"

var (
	Version   = "0.1.0-dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
	GoVersion = "unknown"
)

func SetInfo(v, bt, gc, gv string) {
	if v != "" {
		Version = v
	}
	if bt != "" {
		BuildTime = bt
	}
	if gc != "" {
		GitCommit = gc
	}
	if gv != "" {
		GoVersion = gv
	}
}

func FormatStartupMessage() string {
	return fmt.Sprintf("📱 Nexbot запущен\nВерсия: %s\nСборка: %s", Version, BuildTime)
}
