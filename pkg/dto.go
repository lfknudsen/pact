package pkg

import (
	"fmt"
	"strings"
)

type PackageSeries struct {
	Title         string `json:"title"`
	LatestVersion string `json:"latest_version"`
}

func (pkg *PackageSeries) String() string {
	return fmt.Sprintf("%s v%s", pkg.Title, pkg.LatestVersion)
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

func (pkg *Package) String() string {
	if len(pkg.Authors) == 0 {
		return fmt.Sprintf("%s v%s by unknown author.\n%s\n",
			pkg.Title, pkg.Version, pkg.Synopsis)
	}
	if len(pkg.Authors) == 1 {
		return fmt.Sprintf("%s v%s by %s\n%s\n",
			pkg.Title, pkg.Version, pkg.Authors[0], pkg.Synopsis)
	}
	return fmt.Sprintf("%s v%s\n%s\nAuthors:\n%s\n",
		pkg.Title, pkg.Version, pkg.Synopsis, strings.Join(pkg.Authors, ", "))
}
