package includes

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"strings"
	"testing"
	"testing/fstest"
	"testing/iotest"
)

func TestCompareContents(t *testing.T) {
	long := strings.Repeat("0123456789", 1000)
	for _, test := range []struct {
		name string
		a, b string
		want bool
	}{
		{"empty", "", "", true},
		{"same", "abc", "abc", true},
		{"same long", long, long, true},
		{"exact chunk", strings.Repeat("x", chunkSize), strings.Repeat("x", chunkSize), true},
		{"different", "abc", "abd", false},
		{"different long", long, long[:len(long)-1] + "x", false},
		{"shorter", long, long[:len(long)-1], false},
		{"longer", "abc", "abcd", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			// readers that return short reads, as files over SSH can
			if got := compareContents(iotest.OneByteReader(strings.NewReader(test.a)), strings.NewReader(test.b)); got != test.want {
				t.Errorf("one byte reader: compareContents() = %t, want %t", got, test.want)
			}
			if got := compareContents(iotest.HalfReader(strings.NewReader(test.a)), iotest.DataErrReader(strings.NewReader(test.b))); got != test.want {
				t.Errorf("half reader: compareContents() = %t, want %t", got, test.want)
			}
			if got := compareContents(strings.NewReader(test.a), strings.NewReader(test.b)); got != test.want {
				t.Errorf("compareContents() = %t, want %t", got, test.want)
			}
		})
	}

	if compareContents(strings.NewReader("abc"), iotest.ErrReader(io.ErrClosedPipe)) {
		t.Error("compareContents() ignored a read error")
	}
}

func TestExtractIncludesWithoutIncludesIsAnError(t *testing.T) {
	h := newTestGatewayHost(t)
	source := fstest.MapFS{"files/other.xml": &fstest.MapFile{Data: []byte("x")}}
	if err := extractIncludes(h, source, "v1.2.3", true); err == nil {
		t.Fatal("extractIncludes() with no include files succeeded")
	}
	if _, err := h.Stat(h.PathTo("gateway", "includes", "v1.2.3")); err == nil {
		t.Error("an empty includes directory was created")
	}
}

// includeArchive builds an include archive with a single include file
// of the given declared size.
func includeArchive(t *testing.T, size int64) *bytes.Reader {
	t.Helper()
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(zw)
	write := func(h *tar.Header, data []byte) {
		t.Helper()
		if err := tw.WriteHeader(h); err != nil {
			t.Fatal(err)
		}
		if data != nil {
			if _, err := tw.Write(data); err != nil {
				t.Fatal(err)
			}
		}
	}
	write(&tar.Header{Name: "geneos-includes-v1.0.0/", Typeflag: tar.TypeDir, Mode: 0755}, nil)
	write(&tar.Header{Name: "geneos-includes-v1.0.0/v1.0.0/", Typeflag: tar.TypeDir, Mode: 0755}, nil)
	data := bytes.Repeat([]byte("x"), int(size))
	write(&tar.Header{Name: "geneos-includes-v1.0.0/v1.0.0/itrs.test.xml", Typeflag: tar.TypeReg, Mode: 0644, Size: size}, data)
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return bytes.NewReader(buf.Bytes())
}

func TestStageIncludeArchiveLimitsFileSize(t *testing.T) {
	version, err := stageIncludeArchive(includeArchive(t, 1024), t.TempDir())
	if err != nil || version != "v1.0.0" {
		t.Fatalf("stageIncludeArchive() = %q, %v", version, err)
	}

	// a header claiming a huge file is rejected before anything is written
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(zw)
	for _, h := range []*tar.Header{
		{Name: "geneos-includes-v1.0.0/", Typeflag: tar.TypeDir, Mode: 0755},
		{Name: "geneos-includes-v1.0.0/v1.0.0/", Typeflag: tar.TypeDir, Mode: 0755},
		{Name: "geneos-includes-v1.0.0/v1.0.0/itrs.test.xml", Typeflag: tar.TypeReg, Mode: 0644, Size: maxIncludeFileSize + 1},
	} {
		if err := tw.WriteHeader(h); err != nil {
			t.Fatal(err)
		}
	}
	_ = zw.Close()
	staging := t.TempDir()
	if _, err := stageIncludeArchive(bytes.NewReader(buf.Bytes()), staging); err == nil || !strings.Contains(err.Error(), "too large") {
		t.Fatalf("stageIncludeArchive() with a huge file = %v, want a too large error", err)
	}
}
