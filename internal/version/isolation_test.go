package version

import (
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fixtureTransport func(*http.Request) (*http.Response, error)

func (f fixtureTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// Existing cache-miss tests expect a failed request. Make that deterministic
// without contacting GitHub, and keep every default cache lookup in a fixture.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "td-version-test-")
	if err != nil {
		panic(err)
	}
	if err := os.Setenv("TD_CONFIG_DIR", dir); err != nil {
		panic(err)
	}
	http.DefaultTransport = fixtureTransport(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("fixture network unavailable")
	})
	code := m.Run()
	if err := os.RemoveAll(dir); err != nil {
		panic(err)
	}
	os.Exit(code)
}

func TestCachePathDefaultAndProfile(t *testing.T) {
	fixture := t.TempDir()
	if got, want := resolveCachePath("", func() (string, error) { return fixture, nil }), filepath.Join(fixture, ".config", "td", cacheFile); got != want {
		t.Fatalf("default cache path = %q, want %q", got, want)
	}
	for _, tc := range []struct {
		profile string
		want    string
	}{
		{fixture, filepath.Join(fixture, cacheFile)},
		{"relative-profile", ""},
	} {
		called := false
		got := resolveCachePath(tc.profile, func() (string, error) {
			called = true
			return "", errors.New("must not look up home for an explicit profile")
		})
		if got != tc.want || called {
			t.Fatalf("profile %q: path = %q, want %q; home called = %v", tc.profile, got, tc.want, called)
		}
	}
}

func TestCheckUsesReleaseResponse(t *testing.T) {
	previous := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = previous })
	for _, tc := range []struct {
		name   string
		status int
		body   string
		bad    bool
	}{
		{"new release", http.StatusOK, `{"tag_name":"v2.0.0","html_url":"https://github.com/marcus/td/releases/tag/v2.0.0"}`, false},
		{"invalid response", http.StatusOK, `{invalid}`, true},
		{"service error", http.StatusInternalServerError, `{}`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			http.DefaultTransport = fixtureTransport(func(request *http.Request) (*http.Response, error) {
				calls++
				if request.Method != http.MethodGet || request.URL.String() != "https://api.github.com/repos/marcus/td/releases/latest" {
					t.Fatalf("unexpected update request: %s %s", request.Method, request.URL)
				}
				return &http.Response{StatusCode: tc.status, Status: http.StatusText(tc.status), Body: io.NopCloser(strings.NewReader(tc.body)), Header: make(http.Header)}, nil
			})
			result := Check("v1.0.0")
			if calls != 1 || result.CurrentVersion != "v1.0.0" || (result.Error != nil) != tc.bad {
				t.Fatalf("request count = %d, result = %+v", calls, result)
			}
			if !tc.bad && (!result.HasUpdate || result.LatestVersion != "v2.0.0" || result.UpdateURL != "https://github.com/marcus/td/releases/tag/v2.0.0") {
				t.Fatalf("release response lost: %+v", result)
			}
		})
	}
}
