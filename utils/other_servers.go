package util

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

var providerHTTP = &http.Client{Timeout: 20 * time.Second}

func getJSON(endpoint string, target any) error {
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "mcget/1.0.0")
	resp, err := providerHTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API request failed with status: %s", resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(target)
}

func getText(endpoint string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "mcget/1.0.0")
	resp, err := providerHTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("request failed with status: %s", resp.Status)
	}
	b, err := io.ReadAll(resp.Body)
	return strings.TrimSpace(string(b)), err
}

func headDetails(downloadURL, fileName, checksumURL string) (*BuildDetails, error) {
	req, err := http.NewRequest(http.MethodHead, downloadURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "mcget/1.0.0")
	resp, err := providerHTTP.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusMethodNotAllowed || resp.StatusCode == http.StatusNotImplemented {
		resp.Body.Close()
		req, err = http.NewRequest(http.MethodGet, downloadURL, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", "mcget/1.0.0")
		resp, err = providerHTTP.Do(req)
		if err != nil {
			return nil, err
		}
	}
	resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return nil, fmt.Errorf("download metadata request failed with status: %s", resp.Status)
	}
	name := ""
	if disposition := resp.Header.Get("Content-Disposition"); disposition != "" {
		_, params, parseErr := mime.ParseMediaType(disposition)
		if parseErr == nil {
			name = params["filename"]
		}
	}
	if name == "" && resp.Request != nil {
		name = path.Base(resp.Request.URL.Path)
	}
	if name == "." || name == "/" || name == "" || name == "download" || name == "jar" {
		name = fileName
	}
	details := &BuildDetails{Name: name, Size: resp.ContentLength, URL: downloadURL}
	for _, header := range []string{"X-Checksum-Sha256", "X-Checksum-SHA256"} {
		if sum := strings.TrimSpace(resp.Header.Get(header)); len(sum) == 64 {
			details.SHA256 = sum
			break
		}
	}
	if checksumURL != "" {
		if checksum, err := getText(checksumURL); err == nil {
			fields := strings.Fields(checksum)
			if len(fields) > 0 && len(fields[0]) == 64 {
				details.SHA256 = fields[0]
			}
		}
	}
	return details, nil
}

type purpurVersions struct {
	Versions []string `json:"versions"`
}
type purpurBuildList struct {
	Builds struct {
		All []json.RawMessage `json:"all"`
	} `json:"builds"`
}

func GetPurpurVersions() ([]string, error) {
	var result purpurVersions
	if err := getJSON("https://api.purpurmc.org/v2/purpur/", &result); err != nil {
		return nil, err
	}
	sortMinecraftVersions(result.Versions)
	return result.Versions, nil
}

func sortMinecraftVersions(versions []string) {
	sort.SliceStable(versions, func(i, j int) bool {
		a, b := strings.Split(versions[i], "."), strings.Split(versions[j], ".")
		for n := 0; n < len(a) && n < len(b); n++ {
			ai, aerr := strconv.Atoi(a[n])
			bi, berr := strconv.Atoi(b[n])
			if aerr == nil && berr == nil && ai != bi {
				return ai > bi
			}
			if a[n] != b[n] {
				return a[n] > b[n]
			}
		}
		return len(a) > len(b)
	})
}

func GetPurpurBuilds(version string) ([]string, error) {
	var result purpurBuildList
	endpoint := "https://api.purpurmc.org/v2/purpur/" + url.PathEscape(version)
	if err := getJSON(endpoint, &result); err != nil {
		return nil, err
	}
	builds := make([]string, 0, len(result.Builds.All))
	for _, raw := range result.Builds.All {
		var s string
		if json.Unmarshal(raw, &s) != nil {
			s = strings.Trim(string(raw), `"`)
		}
		builds = append(builds, s)
	}
	return builds, nil
}

func GetPurpurBuildDetails(version, build string) (*BuildDetails, error) {
	v, b := url.PathEscape(version), url.PathEscape(build)
	u := fmt.Sprintf("https://api.purpurmc.org/v2/purpur/%s/%s/download", v, b)
	return headDetails(u, "purpur-"+version+"-"+build+".jar", "")
}

type fabricGameVersion struct {
	Version string `json:"version"`
	Stable  bool   `json:"stable"`
}
type fabricLoaderEntry struct {
	Loader struct {
		Version string `json:"version"`
		Stable  bool   `json:"stable"`
	} `json:"loader"`
}
type fabricInstaller struct {
	Version string `json:"version"`
	Stable  bool   `json:"stable"`
	URL     string `json:"url"`
}

func GetFabricVersions() ([]string, error) {
	var data []fabricGameVersion
	if err := getJSON("https://meta.fabricmc.net/v2/versions/game", &data); err != nil {
		return nil, err
	}
	out := make([]string, 0, len(data))
	for _, item := range data {
		if item.Stable {
			out = append(out, item.Version)
		}
	}
	return out, nil
}

func GetFabricBuilds(version string) ([]string, error) {
	var data []fabricLoaderEntry
	endpoint := "https://meta.fabricmc.net/v2/versions/loader/" + url.PathEscape(version)
	if err := getJSON(endpoint, &data); err != nil {
		return nil, err
	}
	out := make([]string, 0, len(data))
	for _, item := range data {
		if item.Loader.Stable {
			out = append(out, item.Loader.Version)
		}
	}
	return out, nil
}

func GetFabricBuildDetails(version, build string) (*BuildDetails, error) {
	var installers []fabricInstaller
	if err := getJSON("https://meta.fabricmc.net/v2/versions/installer", &installers); err != nil {
		return nil, err
	}
	installer := ""
	for _, item := range installers {
		if item.Stable {
			installer = item.Version
			break
		}
	}
	if installer == "" {
		return nil, fmt.Errorf("no stable Fabric installer is available")
	}
	u := "https://maven.fabricmc.net/net/fabricmc/fabric-installer/" + url.PathEscape(installer) + "/fabric-installer-" + url.PathEscape(installer) + ".jar"
	name := "fabric-installer-" + installer + ".jar"
	details, err := headDetails(u, name, "")
	if err != nil {
		return nil, err
	}
	details.InstallArgs = []string{"server", "-mcversion", version, "-loader", build, "-downloadMinecraft"}
	return details, nil
}

type forgePromotions struct {
	Promos map[string]string `json:"promos"`
}

func getForgePromotions() (map[string]string, error) {
	var data forgePromotions
	if err := getJSON("https://files.minecraftforge.net/net/minecraftforge/forge/promotions_slim.json", &data); err != nil {
		return nil, err
	}
	return data.Promos, nil
}

func GetForgeVersions() ([]string, error) {
	promos, err := getForgePromotions()
	if err != nil {
		return nil, err
	}
	set := map[string]bool{}
	for key := range promos {
		if i := strings.LastIndexByte(key, '-'); i > 0 {
			set[key[:i]] = true
		}
	}
	versions := make([]string, 0, len(set))
	for v := range set {
		versions = append(versions, v)
	}
	sort.Slice(versions, func(i, j int) bool { return versions[i] > versions[j] })
	return versions, nil
}

func GetForgeBuilds(version string) ([]string, error) {
	promos, err := getForgePromotions()
	if err != nil {
		return nil, err
	}
	set := map[string]bool{}
	for _, kind := range []string{"recommended", "latest"} {
		if build := promos[version+"-"+kind]; build != "" {
			set[build+" ("+kind+")"] = true
		}
	}
	builds := make([]string, 0, len(set))
	for b := range set {
		builds = append(builds, b)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(builds)))
	return builds, nil
}

func GetForgeBuildDetails(version, build string) (*BuildDetails, error) {
	forgeVersion := strings.TrimSuffix(strings.TrimSuffix(build, " (latest)"), " (recommended)")
	artifact := version + "-" + forgeVersion
	base := "https://maven.minecraftforge.net/net/minecraftforge/forge/" + url.PathEscape(artifact) + "/forge-" + url.PathEscape(artifact) + "-installer.jar"
	details, err := headDetails(base, "forge-"+artifact+"-installer.jar", base+".sha256")
	if err != nil {
		return nil, err
	}
	details.InstallArgs = []string{"--installServer"}
	return details, nil
}

type mavenMetadata struct {
	Versioning struct {
		Versions struct {
			Version []string `xml:"version"`
		} `xml:"versions"`
	} `xml:"versioning"`
}

func getNeoForgeVersions() ([]string, error) {
	body, err := getText("https://maven.neoforged.net/releases/net/neoforged/neoforge/maven-metadata.xml")
	if err != nil {
		return nil, err
	}
	var metadata mavenMetadata
	if err := xml.Unmarshal([]byte(body), &metadata); err != nil {
		return nil, err
	}
	return metadata.Versioning.Versions.Version, nil
}

func neoForgePrefix(mcVersion string) string {
	parts := strings.Split(mcVersion, ".")
	if len(parts) >= 3 && parts[0] == "1" {
		return strings.Join(parts[1:3], ".") + "."
	}
	if len(parts) >= 2 {
		return strings.Join(parts[:2], ".") + "."
	}
	return mcVersion + "."
}

func minecraftFromNeoForge(version string) string {
	parts := strings.Split(version, ".")
	if len(parts) < 2 {
		return ""
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return ""
	}
	if major < 26 {
		return "1." + parts[0] + "." + parts[1]
	}
	return parts[0] + "." + parts[1]
}

func GetNeoForgeVersions() ([]string, error) {
	all, err := getNeoForgeVersions()
	if err != nil {
		return nil, err
	}
	set := map[string]bool{}
	for _, build := range all {
		if v := minecraftFromNeoForge(build); v != "" {
			set[v] = true
		}
	}
	versions := make([]string, 0, len(set))
	for v := range set {
		versions = append(versions, v)
	}
	sort.Slice(versions, func(i, j int) bool { return versions[i] > versions[j] })
	return versions, nil
}

func GetNeoForgeBuilds(version string) ([]string, error) {
	all, err := getNeoForgeVersions()
	if err != nil {
		return nil, err
	}
	prefix := neoForgePrefix(version)
	var builds []string
	for _, build := range all {
		if strings.HasPrefix(build, prefix) {
			builds = append(builds, build)
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(builds)))
	return builds, nil
}

func GetNeoForgeBuildDetails(version, build string) (*BuildDetails, error) {
	name := "neoforge-" + build + "-installer.jar"
	u := "https://maven.neoforged.net/releases/net/neoforged/neoforge/" + url.PathEscape(build) + "/" + name
	details, err := headDetails(u, name, u+".sha256")
	if err != nil {
		return nil, err
	}
	details.InstallArgs = []string{"--installServer"}
	return details, nil
}

func GetSpigotVersions() ([]string, error) {
	var payload any
	if err := getJSON("https://getbukkit.org/api/jars/spigot", &payload); err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var versions []string
	var visit func(any)
	visit = func(value any) {
		switch item := value.(type) {
		case []any:
			for _, child := range item {
				visit(child)
			}
		case map[string]any:
			for key, child := range item {
				k := strings.ToLower(key)
				if k == "version" || k == "minecraftversion" || k == "mcversion" {
					if v, ok := child.(string); ok && isMinecraftVersion(v) && !seen[v] {
						seen[v] = true
						versions = append(versions, v)
					}
				} else {
					visit(child)
				}
			}
		case string:
			// Some API responses return the version list as plain strings.
			if isMinecraftVersion(item) && !seen[item] {
				seen[item] = true
				versions = append(versions, item)
			}
		}
	}
	visit(payload)
	if len(versions) == 0 {
		return nil, fmt.Errorf("GetBukkit returned no Spigot versions")
	}
	return versions, nil
}

var minecraftVersionPattern = regexp.MustCompile(`^\d+(?:\.\d+){1,3}$`)

func isMinecraftVersion(version string) bool {
	if !minecraftVersionPattern.MatchString(version) {
		return false
	}
	for _, part := range strings.Split(version, ".") {
		if _, err := strconv.Atoi(part); err != nil {
			return false
		}
	}
	return true
}

func GetSpigotBuilds(version string) ([]string, error) {
	_ = version
	return []string{"GetBukkit"}, nil
}

func GetSpigotBuildDetails(version, _ string) (*BuildDetails, error) {
	apiURL := "https://getbukkit.org/api/jars/spigot/" + url.PathEscape(version)
	var payload any
	if err := getJSON(apiURL, &payload); err != nil {
		return nil, err
	}
	downloadURL := apiURL + "/download"
	details, err := headDetails(downloadURL, "spigot-"+version+".jar", "")
	if err != nil {
		return nil, err
	}
	details.Name = "spigot-" + version + ".jar"
	var findHash func(any) string
	findHash = func(value any) string {
		switch item := value.(type) {
		case map[string]any:
			for key, child := range item {
				key = strings.ToLower(strings.ReplaceAll(key, "_", ""))
				if key == "sha256" || key == "sha256checksum" {
					if s, ok := child.(string); ok && len(strings.TrimSpace(s)) == 64 {
						return strings.TrimSpace(s)
					}
				}
				if result := findHash(child); result != "" {
					return result
				}
			}
		case []any:
			for _, child := range item {
				if result := findHash(child); result != "" {
					return result
				}
			}
		}
		return ""
	}
	details.SHA256 = findHash(payload)
	if details.SHA256 == "" {
		checksumURL := fmt.Sprintf("https://getbukkit.org/spigot/spigot-%s.jar.sha256", url.PathEscape(version))
		if checksum, checksumErr := getText(checksumURL); checksumErr == nil {
			if fields := strings.Fields(checksum); len(fields) > 0 && len(fields[0]) == 64 {
				details.SHA256 = fields[0]
			}
		}
	}
	return details, nil
}

func GetServerVersions(server string) ([]string, error) {
	switch server {
	case "paper":
		return GetPaperVersions()
	case "velocity":
		return GetVelocityVersions()
	case "purpur":
		return GetPurpurVersions()
	case "fabric":
		return GetFabricVersions()
	case "forge":
		return GetForgeVersions()
	case "neoforge":
		return GetNeoForgeVersions()
	case "spigot":
		return GetSpigotVersions()
	default:
		return nil, fmt.Errorf("server type %q is not implemented", server)
	}
}

func GetServerBuilds(server, version string) ([]string, error) {
	switch server {
	case "paper":
		return GetPaperBuilds(version)
	case "velocity":
		return GetVelocityBuilds(version)
	case "purpur":
		return GetPurpurBuilds(version)
	case "fabric":
		return GetFabricBuilds(version)
	case "forge":
		return GetForgeBuilds(version)
	case "neoforge":
		return GetNeoForgeBuilds(version)
	case "spigot":
		return GetSpigotBuilds(version)
	default:
		return nil, fmt.Errorf("server type %q is not implemented", server)
	}
}

func GetServerBuildDetails(server, version, build string) (*BuildDetails, error) {
	switch server {
	case "paper":
		return GetPaperBuildDetails(version, build)
	case "velocity":
		return GetVelocityBuildDetails(version, build)
	case "purpur":
		return GetPurpurBuildDetails(version, build)
	case "fabric":
		return GetFabricBuildDetails(version, build)
	case "forge":
		return GetForgeBuildDetails(version, build)
	case "neoforge":
		return GetNeoForgeBuildDetails(version, build)
	case "spigot":
		return GetSpigotBuildDetails(version, build)
	default:
		return nil, fmt.Errorf("server type %q is not implemented", server)
	}
}
