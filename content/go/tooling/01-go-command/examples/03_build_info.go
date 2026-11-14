package main

import (
	"fmt"
	"runtime/debug"
)

func main() {
	// go build stamps module and build metadata into every binary;
	// `go version -m <binary>` prints the same record from the outside.
	info, ok := debug.ReadBuildInfo()
	if !ok {
		fmt.Println("no build info: binary built outside module mode")
		return
	}
	fmt.Println("go version:  ", info.GoVersion)
	fmt.Println("main package:", info.Path)
	fmt.Println("main module: ", info.Main.Path, info.Main.Version)
	fmt.Println("dependencies:", len(info.Deps))

	settings := map[string]string{}
	for _, s := range info.Settings {
		settings[s.Key] = s.Value
	}
	fmt.Println("compiler:    ", settings["-compiler"])
	fmt.Println("target:      ", settings["GOOS"]+"/"+settings["GOARCH"])
	_, hasVCS := settings["vcs.revision"]
	fmt.Println("vcs stamped: ", hasVCS)
}
