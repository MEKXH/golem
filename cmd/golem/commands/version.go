package commands

import (
	"fmt"
	"runtime"

	"github.com/MEKXH/golem/internal/version"
	"github.com/spf13/cobra"
)

// NewVersionCmd creates the version command
func NewVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version of Golem",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("golem %s %s/%s\n", version.Version, runtime.GOOS, runtime.GOARCH)
		},
	}
}
