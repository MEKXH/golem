package command

import (
	"context"
	"fmt"
	"runtime"

	"github.com/MEKXH/golem/internal/version"
)

// VersionCommand implements /version — shows build version info.
type VersionCommand struct{}

func (c *VersionCommand) Name() string        { return "version" }
func (c *VersionCommand) Description() string { return "Show version information" }

func (c *VersionCommand) Execute(_ context.Context, _ string, _ Env) Result {
	return Result{Content: fmt.Sprintf("golem %s %s/%s", version.Version, runtime.GOOS, runtime.GOARCH)}
}
