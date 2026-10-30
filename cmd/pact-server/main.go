package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	engine := gin.Default()
	engine.Handle("GET", "/", func(c *gin.Context) {
		c.String(http.StatusOK, "Hello, World!")
	})
	engine.Handle("GET", "/search", search)
	engine.Handle("GET", "/download", download)
	server := &http.Server{
		Addr:           ":8082",
		Handler:        engine,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("failed to run server: %v", err)
	}
}

func getExecutable() string {
	execFile, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	return filepath.Dir(execFile)
}

var execDir = getExecutable()
var pactDirName = "pact"
var packageDirName = "packages"
var pactDir = filepath.Join(execDir, pactDirName)
var packageDir = filepath.Join(pactDir, packageDirName)
var packageFileExt = ".json"

func search(ctx *gin.Context) {
	query, exists := ctx.Params.Get("query")
	if !exists {
		_ = ctx.AbortWithError(http.StatusBadRequest, errors.New("query parameter is required"))
		return
	}
	// This process unmarshals and then remarshals the package series file... for now
	pkg, err := searchForPackageSeries(query)
	if err != nil {
		if err.Error() == "package not found" {
			ctx.AbortWithStatus(404)
		} else {
			ctx.AbortWithStatus(500)
		}
		return
	}
	ctx.JSON(200, pkg)
}

func download(ctx *gin.Context) {
	name, exists := ctx.Params.Get("name")
	if !exists {
		_ = ctx.AbortWithError(http.StatusBadRequest, errors.New("'name' parameter is required"))
		return
	}
	version, _ := ctx.Params.Get("version")
	pkg, err := searchForPackage(name, version)
	if err != nil {
		if err.Error() == fmt.Sprintf("package '%s' not found", name) {
			ctx.AbortWithStatus(404)
		} else {
			ctx.AbortWithStatus(500)
		}
		return
	}
	if pkg == nil {
		ctx.AbortWithStatus(404)
		return
	}
	ctx.FileAttachment(pkg.Uuid, pkg.Title)
}

func searchForPackage(name string, version string) (*Package, error) {
	dir, err := os.ReadDir(packageDir)
	if err != nil {
		return nil, err
	}
	needle := name + "_" + version
	for _, file := range dir {
		if file.Name() == needle {
			filename := filepath.Join(packageDir, file.Name()+packageFileExt)
			return readPackageFile(filename)
		}
	}
	return nil, errors.New(fmt.Sprintf("package '%s' not found", name))
}

func searchForPackageSeries(name string) (*PackageSeries, error) {
	dir, err := os.ReadDir(packageDir)
	if err != nil {
		return nil, err
	}
	key := name + packageFileExt
	for _, file := range dir {
		if file.Name() == key {
			filename := filepath.Join(packageDir, key)
			return readPackageSeriesFile(filename)
		}
	}
	return nil, errors.New(fmt.Sprintf("package '%s' not found", name))
}

func readPackageFile(path string) (*Package, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var output *Package
	err = json.Unmarshal(data, output)
	return output, err
}

func readPackageSeriesFile(path string) (*PackageSeries, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var output *PackageSeries
	err = json.Unmarshal(data, output)
	return output, err
}

type Package struct {
	Uuid         string   `json:"uuid"`
	Checksum     string   `json:"checksum"`
	Version      string   `json:"version"`
	Title        string   `json:"title"`
	Synopsis     string   `json:"synopsis"`
	Description  string   `json:"description"`
	UploadDate   int64    `json:"upload_date"`
	UpdateDate   int64    `json:"update_date"`
	Authors      []string `json:"authors"`
	Links        []string `json:"links"`
	Dependencies []string `json:"dependencies"`
}

type PackageSeries struct {
	Uuid          string            `json:"uuid"`
	Title         string            `json:"title"`
	LatestVersion string            `json:"latest_version"`
	Versions      map[string]string `json:"versions"` // version -> pkg uuid
}
