package service

import (
	"encoding/json"
	"testing"
)

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		name  string
		left  string
		right string
		want  int
	}{
		{name: "date newer", left: "2026.06.16-1254", right: "2026.06.16-1253", want: 1},
		{name: "date equal with tag", left: "v2026.06.16-1253", right: "2026.06.16-1253", want: 0},
		{name: "numeric newer", left: "1.2.4", right: "1.2.3", want: 1},
		{name: "numeric padding equal", left: "1.2", right: "1.2.0", want: 0},
		{name: "older", left: "1.9.9", right: "2.0.0", want: -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := compareVersions(tt.left, tt.right)
			if got != tt.want {
				t.Fatalf("compareVersions(%q, %q) = %d, want %d", tt.left, tt.right, got, tt.want)
			}
		})
	}
}

func TestParseReleasePayloadFindsConfiguredAsset(t *testing.T) {
	info := VersionInfo{
		Version:       "2026.06.16-1253",
		Repo:          "Zilvren/Learning-Assitant",
		AssetName:     "Tracker.zip",
		AppExe:        "Tracker.exe",
		CanAutoUpdate: true,
	}
	payload := decodeReleaseFixture(t, `{
		"tag_name": "v2026.06.16-1254",
		"published_at": "2026-06-16T12:54:00Z",
		"html_url": "https://example.test/release",
		"body": "notes",
		"assets": [
			{"name": "other.zip", "size": 1, "browser_download_url": "https://example.test/other.zip"},
			{"name": "Tracker.zip", "size": 42, "browser_download_url": "https://example.test/Tracker.zip"}
		]
	}`)

	release := parseReleasePayload(info, payload)
	if !release.HasUpdate {
		t.Fatal("expected release to be newer")
	}
	if !release.AssetFound {
		t.Fatal("expected configured asset to be found")
	}
	if release.AssetSize != 42 || release.DownloadURL != "https://example.test/Tracker.zip" {
		t.Fatalf("unexpected asset data: size=%d url=%q", release.AssetSize, release.DownloadURL)
	}
}

func TestParseReleasePayloadMissingAsset(t *testing.T) {
	info := VersionInfo{
		Version:       "2026.06.16-1253",
		Repo:          "Zilvren/Learning-Assitant",
		AssetName:     "Tracker.zip",
		AppExe:        "Tracker.exe",
		CanAutoUpdate: true,
	}
	payload := decodeReleaseFixture(t, `{
		"tag_name": "v2026.06.16-1254",
		"assets": [{"name": "Other.zip", "size": 1, "browser_download_url": "https://example.test/Other.zip"}]
	}`)

	release := parseReleasePayload(info, payload)
	if !release.HasUpdate {
		t.Fatal("expected release to be newer")
	}
	if release.AssetFound {
		t.Fatal("expected configured asset to be missing")
	}
	if release.DownloadURL != "" {
		t.Fatalf("expected no download URL, got %q", release.DownloadURL)
	}
}

func decodeReleaseFixture(t *testing.T, text string) githubReleaseResponse {
	t.Helper()
	var payload githubReleaseResponse
	if err := json.Unmarshal([]byte(text), &payload); err != nil {
		t.Fatal(err)
	}
	return payload
}
