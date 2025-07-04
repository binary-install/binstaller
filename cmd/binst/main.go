package main

import (
	"github.com/binary-install/binstaller/cmd"
)

var (
	// Version and Commit are set during build
	version = "dev"
	commit  = "none"
)

func main() {
	// Set version info for the cmd package
	cmd.Version = version
	cmd.Commit = commit
	
	cmd.Execute()
}