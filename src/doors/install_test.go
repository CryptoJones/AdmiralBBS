package doors

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestFetchRelease(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"tag_name":"v2.3.4","assets":[
			{"name":"door-linux-amd64","browser_download_url":"http://x/linux"},
			{"name":"door-windows-amd64.exe","browser_download_url":"http://x/win"}]}`)
	}))
	defer srv.Close()

	rel, err := FetchRelease(srv.URL, time.Second)
	if err != nil {
		t.Fatalf("FetchRelease: %v", err)
	}
	if rel.TagName != "v2.3.4" || len(rel.Assets) != 2 {
		t.Fatalf("parsed wrong: tag=%q assets=%d", rel.TagName, len(rel.Assets))
	}
}

// TestPickAssetMatchesHost proves the installer treats every OS/arch equally: it
// selects the asset built for whatever platform the test runs on.
func TestPickAssetMatchesHost(t *testing.T) {
	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}
	want := fmt.Sprintf("mydoor-%s-%s%s", runtime.GOOS, runtime.GOARCH, ext)
	rel := &Release{TagName: "v1", Assets: []ReleaseAsset{
		{Name: "mydoor-plan9-sparc", URL: "http://x/nope"},  // never the host
		{Name: want, URL: "http://x/yes"},                   // the host's binary
		{Name: "mydoor-source.tar.gz", URL: "http://x/src"}, // source, not a binary
	}}
	got, err := PickAsset(rel)
	if err != nil {
		t.Fatalf("PickAsset(%s/%s): %v", runtime.GOOS, runtime.GOARCH, err)
	}
	if got.Name != want {
		t.Fatalf("picked %q, want %q (host %s/%s)", got.Name, want, runtime.GOOS, runtime.GOARCH)
	}
}

func TestPickAssetNoMatch(t *testing.T) {
	rel := &Release{TagName: "v1", Assets: []ReleaseAsset{
		{Name: "mydoor-plan9-sparc", URL: "http://x/nope"},
	}}
	if _, err := PickAsset(rel); err == nil {
		t.Fatal("expected an error when no asset matches the host platform")
	}
}

func TestDownloadBinary(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "BINARY-PAYLOAD")
	}))
	defer srv.Close()

	dest := filepath.Join(t.TempDir(), "sub", "door-bin")
	if err := DownloadBinary(srv.URL, dest, time.Second); err != nil {
		t.Fatalf("DownloadBinary: %v", err)
	}
	b, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if string(b) != "BINARY-PAYLOAD" {
		t.Fatalf("content = %q", b)
	}
	if runtime.GOOS != "windows" { // Windows ignores the exec bit
		if fi, _ := os.Stat(dest); fi.Mode()&0o100 == 0 {
			t.Fatalf("downloaded file is not executable: %v", fi.Mode())
		}
	}
}

func TestFreeLocalPort(t *testing.T) {
	p, err := FreeLocalPort()
	if err != nil || p <= 0 {
		t.Fatalf("FreeLocalPort: port=%d err=%v", p, err)
	}
}

// TestSupervisorLifecycle exercises Launch/Running/Stop without depending on a
// real door binary: a missing path makes the child fail to start fast, so we
// test the supervision bookkeeping (idempotent launch, stop) cross-platform.
func TestSupervisorLifecycle(t *testing.T) {
	sup := NewSupervisor()
	missing := filepath.Join(t.TempDir(), "no-such-door")
	if err := sup.Launch("d", missing, t.TempDir(), "127.0.0.1:65500"); err != nil {
		t.Fatalf("Launch: %v", err)
	}
	if !sup.Running("d") {
		t.Fatal("door should be supervised after Launch")
	}
	if err := sup.Launch("d", missing, t.TempDir(), "127.0.0.1:65500"); err != nil {
		t.Fatalf("idempotent Launch: %v", err)
	}
	sup.Stop("d")
	if sup.Running("d") {
		t.Fatal("door should be gone after Stop")
	}
	sup.StopAll() // no-op, must not panic
}
