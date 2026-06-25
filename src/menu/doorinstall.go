package menu

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"admiralbbs/src/doors"
	"admiralbbs/src/store"
)

// DoorInstaller installs and supervises resident doors fetched from a forge
// release URL (a SysOp-gated action). It downloads the binary matching the BBS
// host's OWN OS/arch, runs it under supervision on a localhost port, registers
// the resident-door bridge, and persists the record so it relaunches on restart.
// Every OS is first-class — the only host-specific bit is which release asset is
// chosen (see doors.PickAsset).
type DoorInstaller struct {
	db      *store.Store
	sup     *doors.Supervisor
	binDir  string // downloaded door binaries
	dataDir string // base dir for per-door working data
}

// NewDoorInstaller wires an installer; binBase is typically the doors-data dir.
func NewDoorInstaller(db *store.Store, sup *doors.Supervisor, binBase string) *DoorInstaller {
	return &DoorInstaller{
		db:      db,
		sup:     sup,
		binDir:  filepath.Join(binBase, "installed-bin"),
		dataDir: filepath.Join(binBase, "installed-data"),
	}
}

func binExt() string {
	if runtime.GOOS == "windows" {
		return ".exe"
	}
	return ""
}

// Install fetches the release at sourceURL, downloads the host-matched binary,
// assigns a localhost port, launches it supervised, registers the resident-door
// bridge, and persists the record. Returns the installed version tag.
func (di *DoorInstaller) Install(name, sourceURL string, minLevel int) (string, error) {
	rel, err := doors.FetchRelease(sourceURL, 0)
	if err != nil {
		return "", fmt.Errorf("fetch release: %w", err)
	}
	asset, err := doors.PickAsset(rel)
	if err != nil {
		return "", err
	}
	slug := slugify(name)
	if slug == "" {
		return "", fmt.Errorf("door name %q has no usable slug", name)
	}
	port, err := doors.FreeLocalPort()
	if err != nil {
		return "", err
	}
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	binPath := filepath.Join(di.binDir, slug+binExt())
	if err := doors.DownloadBinary(asset.URL, binPath, 0); err != nil {
		return "", fmt.Errorf("download %s: %w", asset.Name, err)
	}
	if err := di.db.InstalledDoors().Upsert(store.InstalledDoor{
		Name: name, SourceURL: sourceURL, Version: rel.TagName,
		BinPath: binPath, Address: addr, MinAccessLevel: minLevel,
	}, time.Now()); err != nil {
		return "", fmt.Errorf("persist: %w", err)
	}
	if err := di.sup.Launch(name, binPath, filepath.Join(di.dataDir, slug), addr); err != nil {
		return "", fmt.Errorf("launch: %w", err)
	}
	if err := di.db.Doors().EnsureResidentDoor(name, "tcp", addr, minLevel); err != nil {
		return "", fmt.Errorf("register: %w", err)
	}
	return rel.TagName, nil
}

// RelaunchAll (re)starts every persisted installed door and re-registers its
// bridge — called on BBS boot. A missing binary is re-downloaded first.
func (di *DoorInstaller) RelaunchAll() {
	list, err := di.db.InstalledDoors().List()
	if err != nil {
		log.Printf("installed doors: list failed: %v", err)
		return
	}
	for _, d := range list {
		if _, statErr := os.Stat(d.BinPath); statErr != nil {
			if rel, e := doors.FetchRelease(d.SourceURL, 0); e == nil {
				if a, e2 := doors.PickAsset(rel); e2 == nil {
					if e3 := doors.DownloadBinary(a.URL, d.BinPath, 0); e3 != nil {
						log.Printf("installed door %q: re-download failed: %v", d.Name, e3)
						continue
					}
				}
			}
		}
		if err := di.sup.Launch(d.Name, d.BinPath, filepath.Join(di.dataDir, slugify(d.Name)), d.Address); err != nil {
			log.Printf("installed door %q: launch failed: %v", d.Name, err)
			continue
		}
		if err := di.db.Doors().EnsureResidentDoor(d.Name, "tcp", d.Address, d.MinAccessLevel); err != nil {
			log.Printf("installed door %q: register failed: %v", d.Name, err)
			continue
		}
		log.Printf("installed door %q (%s) relaunched at %s", d.Name, d.Version, d.Address)
	}
}
