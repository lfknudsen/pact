package main

import (
	"bytes"
	"encoding/json/v2"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

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
var config *Config

func main() {
	var err error
	config, err = readConfig()
	if err != nil {
		fmt.Println(err)
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

	if len(os.Args) < 2 {
		fmt.Printf("Usage:\npact <name> [<version>]\n")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "download", "install":
		download()
	case "uninstall":
		uninstall(os.Args[2])
	case "upgrade", "update":
		update(os.Args[2])
	case "search":
		search(os.Args[2])
	case "versions":
		versions()
	case "push":
		push()
	default:
		fmt.Printf("Unrecognized command '%s'.\n", os.Args[1])
		os.Exit(1)
	}
}

func download() {
	name := os.Args[2]
	version := ""
	if len(os.Args) > 3 {
		version = os.Args[3]
	}

	var pkg *Package
	var err error
	pkg, err = getSpecificPackage(name, version)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if pkg == nil {
		fmt.Printf("Could not find package '%s'.\n", name)
		os.Exit(1)
	}

	client := http.DefaultClient
	url := fmt.Sprintf("%s/%s?name=%s&version=%s", config.URL, "download", name, pkg.Version)
	url = strings.TrimPrefix(url, "https://")
	if !strings.HasPrefix(url, "http://") {
		url = "http://" + url
	}
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
	file, err := os.OpenFile(filepath.Join(binDir, pkg.Filename), os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		panic(err)
	}
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		_ = file.Close()
		panic(err)
	}
	_ = file.Close()
}

func uninstall(pkg string) {

}

func update(pkg string) {

}

func getSpecificPackage(name string, version string) (*Package, error) {
	client := http.DefaultClient
	url := fmt.Sprintf("%s/%s?name=%s", config.URL, "search", name)
	if version != "" {
		url = fmt.Sprintf("%s&version=%s", url, version)
	}
	url = strings.TrimPrefix(url, "https://")
	if !strings.HasPrefix(url, "http://") {
		url = "http://" + url
	}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
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
		return nil, err
	}
	pkg := new(Package)
	err = json.Unmarshal(buf.Bytes(), pkg)
	return pkg, err
}

func getLatestPackage(name string) (*Package, error) {
	return getSpecificPackage(name, "")
}

func search(name string) {
	pkg, err := getLatestPackage(name)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println(pkg)
}

func versions() {
	name := os.Args[2]
	client := http.DefaultClient
	url := fmt.Sprintf("%s/%s?name=%s", config.URL, "versions", name)
	url = strings.TrimPrefix(url, "https://")
	if !strings.HasPrefix(url, "http://") {
		url = "http://" + url
	}
	resp, err := client.Get(url)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
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
		fmt.Println(err)
		os.Exit(1)
	}
	list := make([]string, 0, 10)
	err = json.Unmarshal(buf.Bytes(), &list)
	for _, ver := range list {
		fmt.Println(ver)
	}
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
	config = new(Config)
	err = json.Unmarshal(contents, config)
	return config, err
}

func initConfig() (*Config, error) {
	config = &Config{
		Path: managedDir,
		URL:  "http://localhost:8082",
	}
	jsoned, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}
	err = ensureDirsExist()
	if err != nil {
		return nil, err
	}

	return config, os.WriteFile(filepath.Join(configDir, configFilename), jsoned, 0600)
}

func ensureDirsExist() error {
	err := os.MkdirAll(binDir, 0755)
	if err != nil {
		return err
	}
	return os.MkdirAll(pkgDir, 0755)
}

var rxTitleDescription = regexp.MustCompile(`\s*#([^\r\n]+)\s*([^#]+)`)
var rxUsage = regexp.MustCompile(`(?s)## Usage:?\s+(.*)(:?\n##\s*[^#])`)

func push() {
	pwd, err := filepath.Abs(".")
	if err != nil {
		pwd = "."
	}
	filename1 := filepath.Join(pwd, "README.md")
	content, err := os.ReadFile(filename1)
	if err != nil {
		filename2 := filepath.Join(pwd, "README.MD")
		content, err = os.ReadFile(filename2)
		if err != nil {
			ERROR("failed to open '" + filename1 + "'.")
			os.Exit(1)
		}
	}
	match := rxTitleDescription.FindSubmatch(content)
	if match == nil {
		ERROR("failed to find a title and description in '" + filename1 + "'.")
		os.Exit(1)
	}
	title := match[1]
	description := match[2]

	match = rxUsage.FindSubmatch(content)
	if match == nil {
		ERROR("failed to find a usage section (with a ## section beneath) in '" + filename1 + "'.")
		os.Exit(1)
	}
	usage := match[1]
	release := NewRelease(title, description, usage)
	dir, _ := os.Getwd()
	projectName := filepath.Base(dir)
	projectInfoFilename := projectName + ".md"
	fmt.Println("Output packet filename:", projectInfoFilename)
	err = os.WriteFile(projectInfoFilename, release.Markdown(), 0600)
	if err != nil {
		ERROR("failed to write '"+projectInfoFilename+"':", err.Error())
	}

	binFilename := projectName
	if len(os.Args) > 2 {
		binFilename = os.Args[2]
	} else {
		if _, err = os.Stat(binFilename); os.IsNotExist(err) {
			binFilename = binFilename + ".exe"
			if _, err = os.Stat(binFilename); os.IsNotExist(err) {
				ERROR("failed to find '"+binFilename+"':", err.Error())
			}
		}
	}
	upload(projectInfoFilename, binFilename)
}

type Release struct {
	title       string
	description string
	usage       string
}

func NewRelease(title, description, usage []byte) *Release {
	return &Release{
		title:       string(title),
		description: string(description),
		usage:       string(usage),
	}
}

func (r *Release) Markdown() []byte {
	return []byte(fmt.Sprintf("# %s\n\n%s\n\n## Usage\n\n%s",
		strings.TrimSpace(r.title),
		strings.TrimSpace(r.description),
		strings.TrimSpace(r.usage)))
}

func upload(packageFilename, binaryFilename string) {
	var b bytes.Buffer
	mpWriter := multipart.NewWriter(&b)
	wr, err := mpWriter.CreateFormField("package")
	if err != nil {
		panic(err)
	}
	contents := Open(packageFilename)
	defer closeFile(contents)
	_, err = io.Copy(wr, contents)
	if err != nil {
		panic(err)
	}

	file := Open(binaryFilename)
	filename := filepath.Base(file.Name())
	fw, err := mpWriter.CreateFormFile("file", filename)
	if err != nil {
		_ = mpWriter.Close()
		closeFile(file)
		log.Fatal(err)
	}

	n, err := io.Copy(fw, file)
	if err != nil {
		_ = mpWriter.Close()
		closeFile(file)
		log.Fatal(err)
	}
	fmt.Printf("Wrote %d bytes from %s\n", n, filename)
	closeFile(file)
	closeWriter(mpWriter)
	url := strings.TrimPrefix(config.URL, "https://")
	if !strings.HasPrefix(url, "http://") {
		url = "http://" + url
	}
	url += "/push"
	fmt.Println("URL: " + url)
	req, err := http.NewRequest("PUT", url, &b)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Add("Content-Type", mpWriter.FormDataContentType())
	client := http.DefaultClient

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error occurred:")
		log.Fatal(err)
	}
	fmt.Printf("%d: %s\n", resp.StatusCode, resp.Status)
	if resp.StatusCode != 200 {
		Close(resp.Body)
		log.Fatalf("Status code was not 200, but %d\n", resp.StatusCode)
	}
	Close(resp.Body)
}

func Close(body io.ReadCloser) {
	if body != nil {
		_ = body.Close()
	}
}

func Open(filename string) *os.File {
	file, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	return file
}

func closeWriter(w *multipart.Writer) {
	if w != nil {
		_ = w.Close()
	}
}

func closeFile(f *os.File) {
	if f != nil {
		_ = f.Close()
	}
}

func Copy(dst io.Writer, src io.Reader) {
	_, err := io.Copy(dst, src)
	if err != nil {
		log.Fatal(err)
	}
}

func ERROR(msg ...any) {
	_, _ = fmt.Fprintln(os.Stderr, msg...)
}
