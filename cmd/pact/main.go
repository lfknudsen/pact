package main

import (
	"bytes"
	"encoding/json/v2"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	. "pact/pkg"
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
			"If you would prefer for Pact to download/manage its binaries somewhere"+
			"else, you can change this by editing the 'path' field in\n"+
			"%s.\n",
			binDir,
			filepath.Join(configDir, configFilename),
		)
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
		download(os.Args[2], config)
	case "uninstall":
		uninstall(os.Args[2], config)
	case "upgrade", "update":
		update(os.Args[2], config)
	case "search":
		search(os.Args[2], config)

	}
}

func download(pkg string, config *Config) {

}

func uninstall(pkg string, config *Config) {

}

func update(pkg string, config *Config) {

}

func search(name string, config *Config) {
	client := http.DefaultClient
	url := fmt.Sprintf("%s?name=%s", config.URL, name)
	resp, err := client.Get(url)
	if err != nil {
		panic(err)
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)
	if resp.StatusCode != http.StatusOK {
		fmt.Print(resp.Body)
		os.Exit(1)
	}
	var buf bytes.Buffer
	_, err = io.Copy(&buf, resp.Body)
	if err != nil {
		panic(err)
	}
	var pkg *Package
	err = json.Unmarshal(buf.Bytes(), pkg)
	if err != nil {
		panic(err)
	}
	fmt.Println(pkg)
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
	err = ensureDirsExist()
	if err != nil {
		return nil, err
	}

	return &config, os.WriteFile(filepath.Join(configDir, configFilename), jsoned, 0600)
}

func ensureDirsExist() error {
	err := os.MkdirAll(binDir, 0755)
	if err != nil {
		return err
	}
	err = os.MkdirAll(pkgDir, 0755)
	if err != nil {
		return err
	}
}
