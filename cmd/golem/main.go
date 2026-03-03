// Package main 是 Golem 的主程序入口。
package main

import (
	"os"

	"github.com/MEKXH/golem/cmd/golem/commands"
)

// main 是程序的启动入口，负责解析命令行参数并执行根命令。
func main() {
	if err := commands.NewRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}
