package main

import (
	"archive/zip"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

var (
	branch string
	token  string
	dest   string
)

func init() {
	flag.StringVar(&branch, "branch", "", "branch to download the artifacts from it's latest workflow run")
	flag.StringVar(&token, "token", "", "github token for better api privileges")
	flag.StringVar(&dest, "dest", "", "destination folder")
	flag.Parse()
}

type archivesDownloadUrl struct {
	linux   string
	macOS   string
	windows string
}

func main() {
	log.Printf("%v; branch: %v, token: %v, dest: %v\n", filepath.Base(os.Args[0]), branch, token, dest)

	runUrl, err := runUrl()
	noError(err)

	artifactsUrl, err := artifactsUrl(runUrl)
	noError(err)

	archivesUrl, err := archivesUrl(artifactsUrl)
	noError(err)

	noError(fetch(archivesUrl.linux, dest))
	noError(fetch(archivesUrl.macOS, dest))
	noError(fetch(archivesUrl.windows, dest))
}

func runUrl() (string, error) {
	url := "https://api.github.com/repos/spacemeshos/gpu-post/actions/runs"
	if branch != "" {
		url += fmt.Sprintf("?branch=%v", branch)
	}
	log.Printf("GET %v", url)
	res, err := http.DefaultClient.Do(req("GET", url))
	if err != nil {
		return "", err
	}

	defer res.Body.Close()
	rawBody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	var body map[string]interface{}
	if err := json.Unmarshal(rawBody, &body); err != nil {
		return "", err
	}

	totalCount, ok := body["total_count"].(float64)
	if !ok {
		return "", fmt.Errorf("unexpected response: %s", rawBody)
	}
	if totalCount == 0 {
		return "", fmt.Errorf("no workflow runs found for branch %v", branch)
	}

	workflowRuns := body["workflow_runs"].([]interface{})
	workflowRun := workflowRuns[0].(map[string]interface{})

	return workflowRun["url"].(string), nil
}

func artifactsUrl(runUrl string) (string, error) {
	log.Printf("GET %v", runUrl)
	res, err := http.DefaultClient.Do(req("GET", runUrl))
	if err != nil {
		return "", err
	}

	defer res.Body.Close()
	rawBody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	var body map[string]interface{}
	if err := json.Unmarshal(rawBody, &body); err != nil {
		return "", err
	}

	url, ok := body["artifacts_url"].(string)
	if !ok {
		return "", fmt.Errorf("unexpected response: %s", rawBody)
	}

	return url, nil
}

func archivesUrl(artifactsUrl string) (archivesDownloadUrl, error) {
	log.Printf("GET %v", artifactsUrl)
	res, err := http.DefaultClient.Do(req("GET", artifactsUrl))
	if err != nil {
		return archivesDownloadUrl{}, err
	}

	defer res.Body.Close()
	rawBody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return archivesDownloadUrl{}, err
	}

	var body map[string]interface{}
	if err := json.Unmarshal(rawBody, &body); err != nil {
		return archivesDownloadUrl{}, err
	}

	totalCount, ok := body["total_count"].(float64)
	if !ok {
		return archivesDownloadUrl{}, fmt.Errorf("unexpected response: %s", rawBody)
	}
	if totalCount != 3 {
		return archivesDownloadUrl{}, fmt.Errorf("found artifacts listing for %v platforms, expected 3", totalCount)
	}

	ret := archivesDownloadUrl{}
	items := body["artifacts"].([]interface{})
	for _, item := range items {
		artifact := item.(map[string]interface{})
		url := artifact["archive_download_url"].(string)
		switch artifact["name"] {
		case "linux":
			ret.linux = url
		case "macos":
			ret.macOS = url
		case "windows":
			ret.windows = url
		default:
			return archivesDownloadUrl{}, fmt.Errorf("invalid artifact tag: %v", artifact["name"])
		}
	}

	return ret, nil
}

func fetch(url, name string) error {
	name, err := download(url)
	if err != nil {
		return err
	}

	if err := unzip(name); err != nil {
		return err
	}

	if err := os.Remove(name); err != nil {
		return err
	}

	return nil
}

func download(url string) (string, error) {
	log.Printf("GET %v", url)
	res, err := http.DefaultClient.Do(req("GET", url))
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	name := filepath.Join(dest, "temp.zip")
	file, err := os.Create(name)
	if err != nil {
		return "", err
	}
	defer file.Close()

	written, err := io.Copy(file, res.Body)
	if err != nil {
		return "", err
	}

	log.Printf("downloaded %v bytes to %v\n", written, name)

	return name, nil
}

func unzip(path string) error {
	r, err := zip.OpenReader(path)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		name := filepath.Join(dest, f.Name)
		file, err := os.Create(name)
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			return err
		}

		written, err := io.Copy(file, rc)

		file.Close()
		rc.Close()

		if err != nil {
			return err
		}

		log.Printf("unzipped %v bytes to %v\n", written, name)
	}

	return nil
}

func req(method, url string) *http.Request {
	req, _ := http.NewRequest(method, url, nil)
	req.Header.Set("Authorization", fmt.Sprintf("token %v", token))
	return req
}

func noError(err error) {
	if err != nil {
		log.Print(err)
		os.Exit(1)
	}
}
