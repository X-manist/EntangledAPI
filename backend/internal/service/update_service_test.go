//go:build unit

package service

import (
	"context"
	"errors"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type updateServiceCacheStub struct {
	data string
}

func (s *updateServiceCacheStub) GetUpdateInfo(context.Context) (string, error) {
	if s.data == "" {
		return "", errors.New("cache miss")
	}
	return s.data, nil
}

func (s *updateServiceCacheStub) SetUpdateInfo(_ context.Context, data string, _ time.Duration) error {
	s.data = data
	return nil
}

type updateServiceGitHubClientStub struct {
	release        *GitHubRelease
	recentReleases []*GitHubRelease
	recentErr      error
	latestRepo     string
	recentRepo     string
}

func (s *updateServiceGitHubClientStub) FetchLatestRelease(_ context.Context, repo string) (*GitHubRelease, error) {
	s.latestRepo = repo
	return s.release, nil
}

func (s *updateServiceGitHubClientStub) FetchRecentReleases(_ context.Context, repo string, _ int) ([]*GitHubRelease, error) {
	s.recentRepo = repo
	return s.recentReleases, s.recentErr
}

func (s *updateServiceGitHubClientStub) DownloadFile(context.Context, string, string, int64) error {
	panic("DownloadFile should not be called when no update is available")
}

func (s *updateServiceGitHubClientStub) FetchChecksumFile(context.Context, string) ([]byte, error) {
	panic("FetchChecksumFile should not be called when no update is available")
}

func TestUpdateServicePerformUpdateNoUpdateReturnsSentinel(t *testing.T) {
	svc := NewUpdateService(
		&updateServiceCacheStub{},
		&updateServiceGitHubClientStub{
			release: &GitHubRelease{
				TagName: "v0.1.132",
				Name:    "v0.1.132",
			},
		},
		"0.1.132",
		"release",
		"",
		"",
	)

	err := svc.PerformUpdate(context.Background())

	require.Error(t, err)
	require.True(t, errors.Is(err, ErrNoUpdateAvailable))
	require.ErrorIs(t, err, ErrNoUpdateAvailable)
}

func TestUpdateServiceDefaultsToDownstreamReleaseSource(t *testing.T) {
	client := &updateServiceGitHubClientStub{
		release: &GitHubRelease{TagName: "v0.1.153-entangled.1"},
	}
	svc := NewUpdateService(
		&updateServiceCacheStub{},
		client,
		"0.1.153-entangled.1",
		"release",
		"",
		"",
	)

	info, err := svc.CheckUpdate(context.Background(), true)

	require.NoError(t, err)
	require.Equal(t, "X-manist/EntangledAPI", client.latestRepo)
	require.Equal(t, "X-manist/EntangledAPI", info.Repository)
	require.Equal(t, "ghcr.io/x-manist/sub2api", info.DockerImage)
}

func newRollbackTestService(current string, releases []*GitHubRelease) *UpdateService {
	return NewUpdateService(
		&updateServiceCacheStub{},
		&updateServiceGitHubClientStub{recentReleases: releases},
		current,
		"release",
		"",
		"",
	)
}

func TestUpdateServiceListRollbackVersionsFiltersAndCaps(t *testing.T) {
	releases := []*GitHubRelease{
		{TagName: "v0.1.148", PublishedAt: "2026-07-09T00:00:00Z"},                       // newer than current: excluded
		{TagName: "v0.1.147", PublishedAt: "2026-07-08T00:00:00Z"},                       // current: excluded
		{TagName: "v0.1.146-rc1", PublishedAt: "2026-07-07T12:00:00Z", Prerelease: true}, // prerelease: excluded
		{TagName: "v0.1.146", PublishedAt: "2026-07-07T00:00:00Z"},
		{TagName: "v0.1.145", PublishedAt: "2026-07-06T00:00:00Z", Draft: true}, // draft: excluded
		{TagName: "v0.1.144", PublishedAt: "2026-07-05T00:00:00Z"},
		{TagName: "v0.1.144", PublishedAt: "2026-07-05T00:00:00Z"}, // duplicate: excluded
		{TagName: "v0.1.143", PublishedAt: "2026-07-04T00:00:00Z"},
		{TagName: "v0.1.142", PublishedAt: "2026-07-03T00:00:00Z"}, // beyond cap of 3: excluded
	}
	svc := newRollbackTestService("0.1.147", releases)

	versions, err := svc.ListRollbackVersions(context.Background())

	require.NoError(t, err)
	require.Len(t, versions, 3)
	require.Equal(t, "0.1.146", versions[0].Version)
	require.Equal(t, "0.1.144", versions[1].Version)
	require.Equal(t, "0.1.143", versions[2].Version)
}

func TestUpdateServiceListRollbackVersionsSortsUnorderedInput(t *testing.T) {
	releases := []*GitHubRelease{
		{TagName: "v0.1.144"},
		{TagName: "v0.1.146"},
		{TagName: "v0.1.145"},
	}
	svc := newRollbackTestService("0.1.147", releases)

	versions, err := svc.ListRollbackVersions(context.Background())

	require.NoError(t, err)
	require.Len(t, versions, 3)
	require.Equal(t, "0.1.146", versions[0].Version)
	require.Equal(t, "0.1.145", versions[1].Version)
	require.Equal(t, "0.1.144", versions[2].Version)
}

func TestUpdateServiceListRollbackVersionsEmptyWhenNoneOlder(t *testing.T) {
	releases := []*GitHubRelease{
		{TagName: "v0.1.147"},
		{TagName: "v0.1.148"},
	}
	svc := newRollbackTestService("0.1.147", releases)

	versions, err := svc.ListRollbackVersions(context.Background())

	require.NoError(t, err)
	require.Empty(t, versions)
}

func TestUpdateServiceListRollbackVersionsPropagatesFetchError(t *testing.T) {
	svc := NewUpdateService(
		&updateServiceCacheStub{},
		&updateServiceGitHubClientStub{recentErr: errors.New("github unavailable")},
		"0.1.147",
		"release",
		"",
		"",
	)

	_, err := svc.ListRollbackVersions(context.Background())

	require.Error(t, err)
	require.Contains(t, err.Error(), "github unavailable")
}

func TestUpdateServiceRollbackToVersionRejectsDisallowedTargets(t *testing.T) {
	releases := []*GitHubRelease{
		{TagName: "v0.1.148"},
		{TagName: "v0.1.147"},
		{TagName: "v0.1.146"},
		{TagName: "v0.1.145"},
		{TagName: "v0.1.144"},
		{TagName: "v0.1.143"},
		{TagName: "v0.1.142"},
	}
	svc := newRollbackTestService("0.1.147", releases)

	for _, target := range []string{
		"",         // empty
		"0.1.147",  // current version
		"v0.1.147", // current version with prefix
		"0.1.148",  // newer than current
		"0.1.142",  // older than the 3 most recent
		"9.9.9",    // nonexistent
	} {
		err := svc.RollbackToVersion(context.Background(), target)
		require.ErrorIs(t, err, ErrRollbackVersionNotAllowed, "target %q should be rejected", target)
	}
}

func TestUpdateServiceRollbackToVersionAcceptsVPrefix(t *testing.T) {
	// No platform asset in the release: the target passes the allowlist check
	// and fails later at asset lookup, proving the version itself was accepted.
	releases := []*GitHubRelease{
		{TagName: "v0.1.147"},
		{TagName: "v0.1.146"},
	}
	svc := newRollbackTestService("0.1.147", releases)

	err := svc.RollbackToVersion(context.Background(), "v0.1.146")

	require.Error(t, err)
	require.NotErrorIs(t, err, ErrRollbackVersionNotAllowed)
	require.Contains(t, err.Error(), "no compatible release found")
}

func TestUpdateServiceUsesConfiguredPrivateRepositoryAndAssetAPIURL(t *testing.T) {
	client := &updateServiceGitHubClientStub{
		release: &GitHubRelease{
			TagName: "v0.1.151-entangled.2",
			Name:    "Private release",
			Assets: []GitHubAsset{
				{
					Name:               "sub2api_0.1.151-entangled.2_linux_amd64.tar.gz",
					APIURL:             "https://api.github.com/repos/X-manist/EntangledAPI/releases/assets/123",
					BrowserDownloadURL: "https://github.com/X-manist/EntangledAPI/releases/download/v0.1.151-entangled.2/sub2api.tar.gz",
				},
			},
		},
	}
	svc := NewUpdateService(
		&updateServiceCacheStub{},
		client,
		"0.1.151-entangled.1",
		"release",
		"X-manist/EntangledAPI",
		"ghcr.io/x-manist/sub2api",
	)

	info, err := svc.CheckUpdate(context.Background(), true)

	require.NoError(t, err)
	require.True(t, info.HasUpdate)
	require.Equal(t, "X-manist/EntangledAPI", client.latestRepo)
	require.Equal(t, "X-manist/EntangledAPI", info.Repository)
	require.Equal(t, "ghcr.io/x-manist/sub2api", info.DockerImage)
	require.Len(t, info.ReleaseInfo.Assets, 1)
	require.Equal(t, "https://api.github.com/repos/X-manist/EntangledAPI/releases/assets/123", info.ReleaseInfo.Assets[0].DownloadURL)
}

func TestCompareVersionsSupportsCustomReleaseSequence(t *testing.T) {
	tests := []struct {
		current string
		latest  string
		want    int
	}{
		{current: "0.1.151-entangled.1", latest: "0.1.151-entangled.2", want: -1},
		{current: "v0.1.151-entangled.2", latest: "0.1.151-entangled.2", want: 0},
		{current: "0.1.151-entangled.2", latest: "0.1.152-entangled.1", want: -1},
		{current: "0.1.152-entangled.1", latest: "0.1.151-entangled.9", want: 1},
		{current: "0.1.153", latest: "0.1.153-entangled.1", want: -1},
		{current: "0.1.153-entangled.2", latest: "0.1.153", want: 1},
		{current: "0.1.153-rc.1", latest: "0.1.153", want: -1},
		{current: "0.1.153", latest: "0.1.153-entangled.2", want: -1},
		{current: "0.1.153-zzz.1", latest: "0.1.153-entangled.1", want: -1},
		{current: "0.1.149-green.1", latest: "0.1.151-entangled.1", want: -1},
	}

	for _, tt := range tests {
		got := compareVersions(tt.current, tt.latest)
		switch {
		case tt.want < 0:
			require.Less(t, got, 0, "%s vs %s", tt.current, tt.latest)
		case tt.want > 0:
			require.Greater(t, got, 0, "%s vs %s", tt.current, tt.latest)
		default:
			require.Zero(t, got, "%s vs %s", tt.current, tt.latest)
		}
	}
}

func TestCompareVersionsDefinesTransitiveDownstreamOrder(t *testing.T) {
	ordered := []string{
		"1.0.0-alpha.1",
		"1.0.0-rc.1",
		"1.0.0",
		"1.0.0-entangled.1",
		"1.0.0-entangled.2",
		"1.0.1-alpha.1",
	}

	for i := range ordered {
		require.Zero(t, compareVersions(ordered[i], ordered[i]))
		for j := i + 1; j < len(ordered); j++ {
			require.Less(t, compareVersions(ordered[i], ordered[j]), 0, "%s should precede %s", ordered[i], ordered[j])
			require.Greater(t, compareVersions(ordered[j], ordered[i]), 0, "%s should follow %s", ordered[j], ordered[i])
		}
	}
}

func TestUpdateServiceRejectsLegacyCacheWithoutRepository(t *testing.T) {
	cache := &updateServiceCacheStub{data: `{
		"latest":"0.1.153",
		"release_info":null,
		"timestamp":9999999999
	}`}
	svc := NewUpdateService(
		cache,
		&updateServiceGitHubClientStub{},
		"0.1.152-entangled.1",
		"release",
		"",
		"",
	)

	_, err := svc.getFromCache(context.Background())

	require.ErrorContains(t, err, "legacy update cache is missing its repository")
}

func TestUpdateServiceRejectsReleaseWithoutChecksumAsset(t *testing.T) {
	svc := NewUpdateService(
		&updateServiceCacheStub{},
		&updateServiceGitHubClientStub{},
		"0.1.153-entangled.1",
		"release",
		"",
		"",
	)

	err := svc.applyReleaseAssets(context.Background(), []Asset{{
		Name:        "sub2api_0.1.153-entangled.2_" + runtime.GOOS + "_" + runtime.GOARCH + ".tar.gz",
		DownloadURL: "https://github.com/X-manist/EntangledAPI/releases/download/v0.1.153-entangled.2/sub2api.tar.gz",
	}})

	require.Error(t, err)
	require.Contains(t, err.Error(), "missing required checksums.txt")
}

func TestUpdateAssetFileNameUsesReleaseAssetNameForPrivateAPIURL(t *testing.T) {
	asset := Asset{
		Name:        "sub2api_0.1.151-entangled.2_linux_amd64.tar.gz",
		DownloadURL: "https://api.github.com/repos/X-manist/EntangledAPI/releases/assets/123",
	}

	require.Equal(t, "sub2api_0.1.151-entangled.2_linux_amd64.tar.gz", updateAssetFileName(asset))
}
