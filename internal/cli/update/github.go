package update

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/nylas/cli/internal/cli/common"
)

const (
	// GitHub repository information
	repoOwner = "nylas"
	repoName  = "cli"

	// GitHub API endpoints
	releasesAPIURL = "https://api.github.com/repos/%s/%s/releases/latest"

	// HTTP client timeout
	httpTimeout = 30 * time.Second
)

// Release represents a GitHub release.
type Release struct {
	TagName    string  `json:"tag_name"`
	Name       string  `json:"name"`
	HTMLURL    string  `json:"html_url"`
	Body       string  `json:"body"`
	Draft      bool    `json:"draft"`
	Prerelease bool    `json:"prerelease"`
	Assets     []Asset `json:"assets"`
}

// Asset represents a release asset (downloadable file).
type Asset struct {
	Name               string `json:"name"`
	Size               int64  `json:"size"`
	BrowserDownloadURL string `json:"browser_download_url"`
	ContentType        string `json:"content_type"`
}

// getLatestRelease fetches the latest release from GitHub.
func getLatestRelease(ctx context.Context) (*Release, error) {
	url := fmt.Sprintf(releasesAPIURL, repoOwner, repoName)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, common.WrapCreateError("request", err)
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "nylas-cli")

	client := &http.Client{Timeout: httpTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, common.WrapFetchError("release", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("no releases found")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &release, nil
}

// findAsset finds the asset matching the given name.
func findAsset(release *Release, assetName string) *Asset {
	for i := range release.Assets {
		if release.Assets[i].Name == assetName {
			return &release.Assets[i]
		}
	}
	return nil
}

// findChecksumAsset finds the checksums.txt asset.
func findChecksumAsset(release *Release) *Asset {
	return findAsset(release, "checksums.txt")
}
