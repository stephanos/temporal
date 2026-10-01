package sentry_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/getsentry/sentry-go"
)

var releaseEnvironment = []string{
	"SENTRY_RELEASE", "HEROKU_BUILD_COMMIT", "HEROKU_SLUG_COMMIT", "SOURCE_VERSION",
	"CODEBUILD_RESOLVED_SOURCE_VERSION", "CIRCLE_SHA1", "GAE_DEPLOYMENT_ID", "GITHUB_SHA",
	"COMMIT_REF", "VERCEL_GIT_COMMIT_SHA", "ZEIT_GITHUB_COMMIT_SHA", "ZEIT_GITLAB_COMMIT_SHA",
	"ZEIT_BITBUCKET_COMMIT_SHA",
}

func TestReleaseWithoutGit(t *testing.T) {
	for _, name := range releaseEnvironment {
		t.Setenv(name, "")
	}
	t.Setenv("SENTRY_DSN", "")
	bin := t.TempDir()
	marker := filepath.Join(bin, "git-invoked")
	t.Setenv("GOMAD_SENTRY_GIT_MARKER", marker)
	t.Setenv("PATH", bin)
	script := "#!/bin/sh\nprintf invoked > \"$GOMAD_SENTRY_GIT_MARKER\"\nprintf 'host-git-release\\n'\n"
	if err := os.WriteFile(filepath.Join(bin, "git"), []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	var logs bytes.Buffer
	client, err := sentry.NewClient(sentry.ClientOptions{Debug: true, DebugWriter: &logs})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(client.Close)
	_, markerErr := os.Stat(marker)
	if client.Options().Release != "" || !os.IsNotExist(markerErr) {
		t.Fatalf("optional release = %q; fake Git executed = %v; want unknown release without subprocess", client.Options().Release, markerErr == nil)
	}
	if !strings.Contains(logs.String(), "gomad: Git release discovery is unavailable") {
		t.Fatalf("missing explicit skipped-discovery diagnostic: %s", logs.String())
	}
}

func TestReleaseSelectionSurvives(t *testing.T) {
	t.Setenv("SENTRY_DSN", "")
	for _, name := range releaseEnvironment {
		t.Setenv(name, "")
	}
	for _, test := range []struct {
		name, explicit, sentryRelease, buildCommit, slugCommit, want string
	}{
		{name: "explicit", explicit: "explicit-release", sentryRelease: "environment-release", want: "explicit-release"},
		{name: "sentry-environment", sentryRelease: "environment-release", buildCommit: "build-release", want: "environment-release"},
		{name: "build-environment", buildCommit: "build-release", slugCommit: "old-release", want: "build-release"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("SENTRY_RELEASE", test.sentryRelease)
			t.Setenv("HEROKU_BUILD_COMMIT", test.buildCommit)
			t.Setenv("HEROKU_SLUG_COMMIT", test.slugCommit)
			client, err := sentry.NewClient(sentry.ClientOptions{Release: test.explicit})
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(client.Close)
			if got := client.Options().Release; got != test.want {
				t.Fatalf("release = %q, want %q", got, test.want)
			}
		})
	}
}
