package profile

import "testing"

func TestProfileArchiveImpliesLocal(t *testing.T) {
	for _, test := range []struct {
		name    string
		options []ProfileDeployOption
		want    bool
	}{
		{name: "default"},
		{name: "explicit local", options: []ProfileDeployOption{WithLocal(true)}, want: true},
		{name: "file", options: []ProfileDeployOption{WithArchive("release.tar.gz")}, want: true},
		{name: "directory", options: []ProfileDeployOption{WithArchive("/tmp/releases")}, want: true},
		{name: "missing path", options: []ProfileDeployOption{WithArchive("/nonexistent/releases")}, want: true},
		{name: "URL", options: []ProfileDeployOption{WithArchive("https://example.com/release.tar.gz")}},
		{name: "URL with explicit local", options: []ProfileDeployOption{WithArchive("https://example.com/release.tar.gz"), WithLocal(true)}, want: true},
		{name: "local before archive", options: []ProfileDeployOption{WithLocal(false), WithArchive("release.tar.gz")}, want: true},
		{name: "local after archive", options: []ProfileDeployOption{WithArchive("release.tar.gz"), WithLocal(false)}, want: true},
		{name: "final archive URL", options: []ProfileDeployOption{WithArchive("release.tar.gz"), WithArchive("https://example.com/release.tar.gz")}},
	} {
		t.Run(test.name, func(t *testing.T) {
			opts := evaluateProfileDeployOptions(test.options...)
			if opts.local != test.want {
				t.Errorf("local mode = %v, want %v", opts.local, test.want)
			}
		})
	}
}
