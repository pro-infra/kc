package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
)

const (
	GITHUB_OWNER = "pro-infra"
	GITHUB_REPO  = "kc"
)

type versiont struct {
	maj, min, pat int
}

type GitHubTagResponse struct {
	Ref string `json:"ref"` // "ref": "refs/tags/v0.2.0",
	Url string `json:"url"` //  "url": "https://api.github.com/repos/schreibe72/kc/git/refs/tags/v0.2.0",
}

var versionExp = regexp.MustCompile(`^refs/tags/v[0-9]+.[0-9]+(.[0-9]+)?$`)
var versionNumExp = regexp.MustCompile(`[0-9]+`)

func VersionFromString(str string) versiont {
	if !versionExp.MatchString(str) {
		return versiont{}
	}

	var ver versiont
	s := versionNumExp.FindAllString(str, -1)
	if i, err := strconv.Atoi(s[0]); err != nil {
		return versiont{}
	} else {
		ver.maj = i
	}
	if i, err := strconv.Atoi(s[1]); err != nil {
		return versiont{}
	} else {
		ver.min = i
	}
	if len(s) == 3 {
		if i, err := strconv.Atoi(s[2]); err != nil {
			return versiont{}
		} else {
			ver.pat = i
		}
	}
	return ver
}

func (v versiont) eq(v2 versiont) bool {
	return v.maj == v2.maj && v.min == v2.min && v.pat == v2.pat
}

func (v versiont) gt(v2 versiont) bool {
	if v.maj != v2.maj {
		return v.maj > v2.maj
	}
	if v.min != v2.min {
		return v.min > v2.min
	}
	return v.pat > v2.pat
}

func (v versiont) ge(v2 versiont) bool {
	return v.eq(v2) || v.gt(v2)
}

func (v versiont) String() string {
	return fmt.Sprintf("v%d.%d.%d", v.maj, v.min, v.pat)
}

func updatekc(dryRun bool) {
	var err error
	var versions []versiont
	if versions, err = getAvailableVersions(GITHUB_OWNER, GITHUB_REPO); err != nil {
		log.Fatalln(err)
	}

	if len(versions) <= 1 {
		log.Println("No newer version found")
		return
	}

	max := versions[0]
	for _, v := range versions {
		if v.gt(max) {
			max = v
		}
	}
	current := VersionFromString(version)
	if current.ge(max) {
		log.Println("Newest version is already installed")
		return
	}
	log.Println("Update needed")

	filename, err := os.Executable()
	if err != nil {
		panic(err)
	}
	log.Printf("Update executable: %s\n", filename)

	if err = checkWriteProtection(filename); err != nil {
		log.Fatalln(err)
	}

	url := fmt.Sprintf("https://github.com/%s/%s/releases/download/%s/kc.%s_%s", GITHUB_OWNER, GITHUB_REPO, max.String(), GOOS, GOARCH)
	if dryRun {
		log.Println("Would download", url, "to", filename)
	} else {
		if err = downloadFile(url, filename); err != nil {
			log.Fatalln(err)
		}
	}
	log.Println("success")
}

func getAvailableVersions(owner, repo string) ([]versiont, error) {
	client := &http.Client{}
	header := map[string][]string{
		"Accept": {"application/json"},
	}
	var err error
	var req *http.Request
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/git/refs/tags", owner, repo)
	if req, err = http.NewRequest("GET", url, nil); err != nil {
		return nil, err
	}
	req.Header = header

	var resp *http.Response
	if resp, err = client.Do(req); err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var body []byte
	if body, err = io.ReadAll(resp.Body); err != nil {
		return nil, err
	}

	var tags []GitHubTagResponse
	if err = json.Unmarshal(body, &tags); err != nil {
		return nil, err
	}

	return parseVersions(tags), nil
}

func parseVersions(tags []GitHubTagResponse) []versiont {
	versions := make([]versiont, len(tags))
	for i, tag := range tags {
		if versionExp.MatchString(tag.Ref) {
			log.Println("Found version", tag.Ref)
			versions[i] = VersionFromString(tag.Ref)
		}
	}
	return versions
}

func checkWriteProtection(filename string) error {
	var err error
	var info os.FileInfo
	if info, err = os.Lstat(filename); err != nil {
		return err
	}
	if info.Mode().Perm()&0200 == 0 {
		return errors.New("can not update - file is write protected")
	}
	return nil
}

func downloadFile(url, filename string) error {
	var err error
	var resp *http.Response
	if resp, err = http.Get(url); err != nil {
		return err
	}
	defer resp.Body.Close()

	var file *os.File
	if file, err = os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0755); err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	return err
}
