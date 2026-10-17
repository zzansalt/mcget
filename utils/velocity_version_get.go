package util

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type velocityVersionResponse struct {
	Versions []struct {
		Version struct {
			ID string `json:"id"`
		} `json:"version"`
	} `json:"versions"`
}

type velocityBuildsResponse struct {
	Builds []int `json:"builds"`
}

type velocityBuildDetailsResponse struct {
	Downloads map[string]struct {
		Name      string `json:"name"`
		Size      int64  `json:"size"`
		Checksums struct {
			SHA256 string `json:"sha256"`
		} `json:"checksums"`
		URL string `json:"url"`
	} `json:"downloads"`
}

// GetVelocityVersions fetches the list of PaperMC versions (latest first).
func GetVelocityVersions() ([]string, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest("GET", "https://fill.papermc.io/v3/projects/velocity/versions", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("accept", "application/json")
	req.Header.Set("User-Agent", "mcget/1.0.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status: %d", resp.StatusCode)
	}

	var data paperVersionResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	var versions []string
	for _, v := range data.Versions {
		versions = append(versions, v.Version.ID)
	}

	return versions, nil
}

// GetVelocityBuilds fetches build numbers for a given version.
func GetVelocityBuilds(version string) ([]string, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest("GET", "https://fill.papermc.io/v3/projects/velocity/versions/"+version, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("accept", "application/json")
	req.Header.Set("User-Agent", "mcget/1.0.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status: %d", resp.StatusCode)
	}

	var data paperBuildsResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	var builds []string
	for _, b := range data.Builds {
		builds = append(builds, strconv.Itoa(b))
	}

	return builds, nil
}

// GetVelocityBuildDetails fetches download info for a specific version and build.
func GetVelocityBuildDetails(version string, build string) (*BuildDetails, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest("GET", "https://fill.papermc.io/v3/projects/velocity/versions/"+version+"/builds/"+build, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("accept", "application/json")
	req.Header.Set("User-Agent", "mcget/1.0.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status: %d", resp.StatusCode)
	}

	var data paperBuildDetailsResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	var dlInfo struct {
		Name      string `json:"name"`
		Size      int64  `json:"size"`
		Checksums struct {
			SHA256 string `json:"sha256"`
		} `json:"checksums"`
		URL string `json:"url"`
	}
	found := false

	if defaultDl, ok := data.Downloads["server:default"]; ok {
		dlInfo = defaultDl
		found = true
	} else {
		for k, dl := range data.Downloads {
			if strings.Contains(k, "server") {
				dlInfo = dl
				found = true
				break
			}
		}
	}

	if !found {
		return nil, fmt.Errorf("no server download found in build details")
	}

	return &BuildDetails{
		Name:   dlInfo.Name,
		Size:   dlInfo.Size,
		SHA256: dlInfo.Checksums.SHA256,
		URL:    dlInfo.URL,
	}, nil
}

// DownloadFile downloads a URL to a target destination, notifying progress via progressChan.
func velocityDownloadFile(url string, destPath string, progressChan chan<- int64) error {
	dir := filepath.Dir(destPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()

	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "mcget/1.0.0")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	buf := make([]byte, 32*1024)
	var written int64
	for {
		nr, er := resp.Body.Read(buf)
		if nr > 0 {
			nw, ew := out.Write(buf[0:nr])
			if nw > 0 {
				written += int64(nw)
				if progressChan != nil {
					progressChan <- written
				}
			}
			if ew != nil {
				err = ew
				break
			}
			if nr != nw {
				err = io.ErrShortWrite
				break
			}
		}
		if er != nil {
			if er != io.EOF {
				err = er
			}
			break
		}
	}

	return err
}
