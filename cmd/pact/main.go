package main

import (
	"encoding/json/v2"
	"fmt"
	"os"
	"path/filepath"
)

func getExecDir() string {
	execFile, err := os.Executable()
	if err != nil {
		panic(err)
	}
	return filepath.Dir(execFile)
}

var execDir = getExecDir()
var managedDirName = ".pact"
var binDirName = "bin"
var pkgDirName = "pkg"
var managedDir = filepath.Join(execDir, managedDirName)
var binDir = filepath.Join(managedDir, binDirName)
var pkgDir = filepath.Join(managedDir, pkgDirName)
var configDir = execDir
var configFilename = ".pact-config.json"

func main() {
	config, err := readConfig()
	if err != nil {
		config, err = initConfig()
		if err != nil {
			panic(err)
		}
		fmt.Printf("Pact has finished setting up.\n"+
			"Please add the following location to your PATH:\n%s\n"+
			"You will find a configuration file named '%s'\n"+
			"in the directory %s.\n"+
			"Change the value of the 'path' field inside if you'd like for Pact"+
			"to download binaries to a different location than\n%s\n.",
			binDir,
			configFilename,
			configDir,
			managedDirName)
		os.Exit(0)
	}
	if config == nil {
		fmt.Printf("An error occurred parsing the config file. Aborting.\n")
		os.Exit(1)
	}

	if len(os.Args) != 3 {
		fmt.Printf("Usage:\npact <name> [<version>]\n")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "install":
		download(os.Args[2], config.Path)
	case "uninstall":
		uninstall(os.Args[2], config.Path)
	case "upgrade", "update":
		update(os.Args[2], config.Path)
	case "search":
		search(os.Args[2], config.Path)

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

type Config struct {
	Path string `json:"path"`
	URL  string `json:"url"`
}

func readConfig() (*Config, error) {
	contents, err := os.ReadFile(filepath.Join(configDir, configFilename))
	if err != nil {
		return nil, err
	}
	var config *Config
	err = json.Unmarshal(contents, config)
	return config, err
}

func initConfig() (*Config, error) {
	var config = Config{
		Path: managedDir,
		URL:  "localhost:8082",
	}
	jsoned, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}
	err = os.MkdirAll(binDir, 0755)
	if err != nil {
		return nil, err
	}
	return &config, os.WriteFile(filepath.Join(configDir, configFilename), jsoned, 0600)
}
