package util

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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

type paperVersionResponse struct {
	Versions []struct {
		Version struct {
			ID string `json:"id"`
		} `json:"version"`
	} `json:"versions"`
}

type paperBuildsResponse struct {
	Builds []int `json:"builds"`
}

type paperBuildDetailsResponse struct {
	Downloads map[string]struct {
		Name      string `json:"name"`
		Size      int64  `json:"size"`
		Checksums struct {
			SHA256 string `json:"sha256"`
		} `json:"checksums"`
		URL string `json:"url"`
	} `json:"downloads"`
}

// GetPaperVersions fetches the list of PaperMC versions (latest first).
func GetPaperVersions() ([]string, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest("GET", "https://fill.papermc.io/v3/projects/paper/versions", nil)
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

// GetPaperBuilds fetches build numbers for a given version.
func GetPaperBuilds(version string) ([]string, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest("GET", "https://fill.papermc.io/v3/projects/paper/versions/"+version, nil)
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

// GetPaperBuildDetails fetches download info for a specific version and build.
func GetPaperBuildDetails(version string, build string) (*BuildDetails, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest("GET", "https://fill.papermc.io/v3/projects/paper/versions/"+version+"/builds/"+build, nil)
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
func DownloadFile(ctx context.Context, url string, destPath string, expectedSHA256 string, overwrite bool, progressChan chan<- int64) error {
	dir := filepath.Dir(destPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".mcget-*.part")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	keepTemp := false
	defer func() {
		tmp.Close()
		if !keepTemp {
			os.Remove(tmpPath)
		}
	}()

	client := &http.Client{}
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
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
	hash := sha256.New()
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		nr, er := resp.Body.Read(buf)
		if nr > 0 {
			nw, ew := tmp.Write(buf[0:nr])
			if nw > 0 {
				hash.Write(buf[:nw])
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

	if err != nil {
		return err
	}
	if err := tmp.Sync(); err != nil {
		return err
	}
	if expectedSHA256 != "" {
		actual := hex.EncodeToString(hash.Sum(nil))
		if !strings.EqualFold(actual, expectedSHA256) {
			return fmt.Errorf("SHA-256 verification failed (expected %s, got %s)", expectedSHA256, actual)
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if overwrite {
		backup, backupErr := os.CreateTemp(dir, ".mcget-backup-*")
		if backupErr != nil {
			return backupErr
		}
		backupPath := backup.Name()
		_ = backup.Close()
		_ = os.Remove(backupPath)
		if _, statErr := os.Stat(destPath); statErr == nil {
			if err := os.Rename(destPath, backupPath); err != nil {
				return fmt.Errorf("could not replace existing file: %w", err)
			}
			if err := os.Rename(tmpPath, destPath); err != nil {
				_ = os.Rename(backupPath, destPath)
				return err
			}
			_ = os.Remove(backupPath)
			keepTemp = true
			return nil
		}
	}
	// Prefer an atomic hard link. Some Windows filesystems do not support it,
	// so fall back to an exclusive create and copy of the already verified file.
	if err := os.Link(tmpPath, destPath); err == nil {
		// Windows does not allow removing an open file, so close the temp handle
		// before unlinking its temporary name.
		if err := tmp.Close(); err != nil {
			_ = os.Remove(destPath)
			return err
		}
		if err := os.Remove(tmpPath); err != nil {
			_ = os.Remove(destPath)
			return err
		}
		keepTemp = true
		return nil
	}

	if _, err := tmp.Seek(0, io.SeekStart); err != nil {
		return err
	}
	final, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(final, tmp)
	syncErr := final.Sync()
	closeErr := final.Close()
	if copyErr != nil || syncErr != nil || closeErr != nil {
		_ = os.Remove(destPath)
		if copyErr != nil {
			return copyErr
		}
		if syncErr != nil {
			return syncErr
		}
		return closeErr
	}
	return nil
}
