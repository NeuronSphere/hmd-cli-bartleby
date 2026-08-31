// Command bartleby renders reStructuredText documentation using Sphinx inside a
// Docker container.
package main

import "github.com/neuronsphere/hmd-cli-bartleby/cmd"

// version is set at build time with -ldflags "-X main.version=<version>".
var version = "dev"

func main() {
	cmd.Execute(version)
}
