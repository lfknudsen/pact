package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	path := os.Getenv("PACT_PATH")
	if path == "" {
		execFile, err := os.Executable()
		if err != nil {
			panic(err)
		}
		execDir := filepath.Dir(execFile)
		managedDir := filepath.Join(execDir, "bin")
		err = os.Setenv("PACT_PATH", managedDir)
		if err != nil {
			fmt.Printf("Pact failed to set the environment variable PACT_PATH:\n%s\n", err)
			os.Exit(1)
		}
		fmt.Printf("Pact has finished setting up.\n"+
			"Please add the following location to your PATH:\n%s\n"+
			"If you'd like to change where Pact installs and manages programmes,\n"+
			"please change the value of the PACT_PACT environment variable,\n"+
			"and add the new location to your PATH.", managedDir)
		os.Exit(0)
	}

	if len(os.Args) == 3 {
		switch os.Args[1] {
		case "install":
			download(os.Args[2], path)
		case "uninstall":
			uninstall(os.Args[2], path)
		case "upgrade", "update":
			update(os.Args[2], path)
		case "search":
			search(os.Args[2], path)
		}
	}
}

func download(pkg string, path string) {

}

func uninstall(pkg string, path string) {

}

func update(pkg string, path string) {

}

func search(pkg string, path string) {
}
