package main

import "github.com/precedent-cli/precedent/cmd"

var version = "dev"

func main() {
	cmd.SetVersion(version)
	cmd.Execute()
}
