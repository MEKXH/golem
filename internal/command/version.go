package command

import (
	"context"
	"fmt"
	"runtime"

	"github.com/MEKXH/golem/internal/version"
)

// VersionCommand 实现 /version 命令 — 用于显示当前 Golem 二进制文件的版本及构建环境。
type VersionCommand struct{}

// Name 返回命令名称。
func (c *VersionCommand) Name() string        { return "version" }

// Description 返回命令描述。
func (c *VersionCommand) Description() string { return "Show version information" }

// Execute 执行显示版本信息的逻辑。
func (c *VersionCommand) Execute(_ context.Context, _ string, _ Env) Result {
	return Result{Content: fmt.Sprintf("golem %s %s/%s", version.Version, runtime.GOOS, runtime.GOARCH)}
}
