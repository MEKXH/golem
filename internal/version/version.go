package version

import "runtime/debug"

var (
	// Version is the current version of the application.
	// It is intended to be set at build time using -ldflags.
	// Falls back to the module version embedded by go install.
	Version = "dev"
)

func init() {
	if Version != "dev" {
		return
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		Version = info.Main.Version
	}
}
