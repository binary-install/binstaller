package main

import (
	"context"
	"os"
	"syscall"

	"github.com/binary-install/binstaller/cmd"
	"github.com/charmbracelet/fang"
)

var (
	// Version and Commit are set during build
	version = "dev"
	commit  = "none"
)

func main() {
	// Use fang to execute the command with enhanced features
	if err := fang.Execute(
		context.Background(),
		cmd.RootCmd,
		fang.WithVersion(version),
		fang.WithCommit(commit),
		fang.WithNotifySignal(syscall.SIGINT, syscall.SIGTERM),
	); err != nil {
		os.Exit(1)
	}
}