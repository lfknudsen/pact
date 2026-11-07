package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"maps"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
	"uuid"

	"github.com/gin-gonic/gin"
)

func main() {
	err := os.MkdirAll(packageDir, 0755)
	if err != nil {
		log.Fatal(err)
	}
	err = os.MkdirAll(fileDir, 0755)
	if err != nil {
		log.Fatal(err)
	}

	engine := gin.Default()
	engine.Handle("GET", "/", func(c *gin.Context) {
		c.String(http.StatusOK, "Hello, World!")
	})
	engine.Handle("GET", "/search", search)
	engine.Handle("GET", "/versions", versions)
	engine.Handle("GET", "/download", download)
	engine.Handle("PUT", "/push", receivePush)
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
var fileDirName = "files"
var pactDir = filepath.Join(execDir, pactDirName)

// packageDir contains the .json files for each package series.
var packageDir = filepath.Join(pactDir, packageDirName)

// fileDir contains the directories within which the actual files are located.
var fileDir = filepath.Join(pactDir, fileDirName)
var packageFileExt = ".json"

func search(ctx *gin.Context) {
	params := ctx.Request.URL.Query()
	name := params.Get("name")
	if name == "" {
		_ = ctx.AbortWithError(http.StatusBadRequest, errors.New("'name' parameter is required"))
		return
	}
	version := params.Get("version")
	// If no version parameter was given, we grab the latest version.
	if version == "" {
		series, err := searchForPackageSeries(name)
		if err != nil {
			log.Println(err)
			if err.Error() == fmt.Sprintf("package '%s' not found", name) {
				_ = ctx.AbortWithError(http.StatusNotFound, err)
			} else {
				_ = ctx.AbortWithError(http.StatusInternalServerError, err)
			}
			return
		}
		version = series.LatestVersion
	}

	// Retrieve the specific package with the latest version.
	filename := fmt.Sprintf("%s_%s%s", name, version, packageFileExt)
	filename = filepath.Join(packageDir, filename)
	pkg, err := readPackageFile(filename)
	if err != nil {
		log.Println(err)
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
	ctx.JSON(200, pkg)
}

// download is the handler for the `/download` endpoint.
// It retrieves a file which matches the given name (and optionally version),
// and attaches it to the response.
func download(ctx *gin.Context) {
	params := ctx.Request.URL.Query()
	name := params.Get("name")
	if name == "" {
		_ = ctx.AbortWithError(http.StatusBadRequest, errors.New("'name' parameter is required"))
		return
	}
	version := params.Get("version")
	if version == "" {
		_ = ctx.AbortWithError(http.StatusBadRequest, errors.New("'version' parameter is required"))
		return
	}

	filename := fmt.Sprintf("%s_%s%s", name, version, packageFileExt)
	filename = filepath.Join(packageDir, filename)
	pkg, err := readPackageFile(filename)
	if err != nil {
		log.Println(err)
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

	ctx.FileAttachment(pkg.Path(), pkg.Title)
}

func receivePush(c *gin.Context) {
	log.Println("Received request to upload one file from", c.Request.RemoteAddr)
	err := c.Request.ParseMultipartForm(32 << 20)
	if err != nil {
		log.Println(err)
		c.String(http.StatusBadRequest, "parse form err: %s", err.Error())
		return
	}
	pkgStr, found := c.GetPostForm("package")
	if !found {
		c.String(http.StatusBadRequest, "missing 'package' form field")
		return
	}
	pkg := &Package{}
	err = json.Unmarshal([]byte(pkgStr), pkg)
	if err != nil {
		log.Println(err)
		c.String(http.StatusBadRequest, "parse form err: %s", err.Error())
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		log.Println(err)
		c.String(http.StatusBadRequest, "get form err: %s", err.Error())
		return
	}
	pkg.Uuid = uuid.New().String()
	pkg.Filename = sanitiseFilename(pkg.Filename)
	file.Filename = pkg.AssembleFilename()
	err = c.SaveUploadedFile(file, pkg.Path())
	if err != nil {
		log.Println(err)
		c.String(http.StatusBadRequest, "save err: %s", err.Error())
		return
	}
	c.String(http.StatusOK, "upload success")
}

func push(ctx *gin.Context) {
	pkg := new(Package)
	if err := ctx.BindJSON(pkg); err != nil {
		err = ctx.AbortWithError(http.StatusBadRequest, err)
		if err != nil {
			log.Println(err)
		}
		return
	}
	filename := fmt.Sprintf("%s_%s%s", pkg.Title, pkg.Version, packageFileExt)
	filename = filepath.Join(packageDir, filename)
	_, err := readPackageFile(filename)
	if err == nil {
		ctx.AbortWithStatus(200)
		return
	}
	encoded, err := json.Marshal(pkg)
	if err != nil {
		err = ctx.AbortWithError(http.StatusInternalServerError, err)
		if err != nil {
			log.Println(err)
		}
		return
	}
	err = os.WriteFile(pkg.Path(), encoded, 0644)
	if err != nil {
		err = ctx.AbortWithError(http.StatusInternalServerError, err)
		if err != nil {
			log.Println(err)
		}
		return
	}
	return
}

func versions(ctx *gin.Context) {
	params := ctx.Request.URL.Query()
	name := params.Get("name")
	if name == "" {
		_ = ctx.AbortWithError(http.StatusBadRequest, errors.New("'name' parameter is required"))
		return
	}
	series, err := searchForPackageSeries(name)
	if err != nil {
		log.Println(err)
		if err.Error() == fmt.Sprintf("package '%s' not found", name) {
			_ = ctx.AbortWithError(http.StatusNotFound, err)
		} else {
			_ = ctx.AbortWithError(http.StatusInternalServerError, err)
		}
		return
	}
	if series == nil {
		log.Println("no series found")
		_ = ctx.AbortWithError(http.StatusNotFound, err)
		return
	}
	listVer := slices.Collect(maps.Keys(series.Versions))
	slices.SortFunc[[]string](listVer, cmpVersions)
	ctx.JSON(200, &listVer)
}

func cmpVersions(a string, b string) int {
	aSplit := strings.Split(a, ".")
	bSplit := strings.Split(b, ".")
	if len(aSplit) != len(bSplit) {
		return 0
	}
	for i := range aSplit {
		aInt, err := strconv.ParseInt(aSplit[i], 10, 64)
		if err != nil {
			return 0
		}
		bInt, err := strconv.ParseInt(bSplit[i], 10, 64)
		if err != nil {
			return 0
		}
		if aInt > bInt {
			return -1
		}
		if aInt < bInt {
			return 1
		}
	}
	return 0
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
	output := new(Package)
	err = json.Unmarshal(data, output)
	return output, err
}

func readPackageSeriesFile(path string) (*PackageSeries, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	output := new(PackageSeries)
	err = json.Unmarshal(data, output)
	return output, err
}

type Package struct {
	Uuid         string   `json:"uuid"`
	Filename     string   `json:"filename"`
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
	SeriesUuid   string   `json:"series_uuid"`
}

func (pkg Package) AssembleFilename() string {
	stem, ext := splitExt(pkg.Filename)
	return fmt.Sprintf("%s_%s%s", stem, pkg.Version, ext)
}

// Path returns the path to the file.
func (pkg Package) Path() string {
	return filepath.Join(pkg.Dir(), pkg.AssembleFilename())
}

func (pkg Package) Dir() string {
	return filepath.Join(fileDir, pkg.SeriesUuid)
}

type PackageSeries struct {
	Uuid          string            `json:"uuid"`
	Title         string            `json:"title"`
	LatestVersion string            `json:"latest_version"`
	Versions      map[string]string `json:"versions"` // version -> pkg uuid
}

// As per https://github.com/gin-gonic/gin/issues/1693
func sanitiseFilename(filename string) string {
	filename = filepath.Clean(filename)
	filename = filepath.Base(filename)
	return filename
}

// splitExt returns the stem and extension of the given filename.
func splitExt(filename string) (stem string, ext string) {
	extStart := strings.LastIndex(filename, ".")
	if extStart == -1 {
		return filename, ""
	}
	return filename[:extStart], filename[extStart:]
}

// generateUniqueFilename appends the filename with a (<number>) if a file with
// the given name already exists in PlumeDir.
func generateUniqueFilename(dir string, filename string) string {
	counter := 1
	newFilename := filename
	stem, ext := splitExt(filename)
	_, err := os.Stat(filepath.Join(dir, filename))
	for err == nil {
		newFilename = fmt.Sprintf("%s (%d)%s", stem, counter, ext)
		_, err = os.Stat(filepath.Join(dir, newFilename))
		counter++
	}
	return newFilename
}
