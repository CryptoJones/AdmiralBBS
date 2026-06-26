package menu

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"admiralbbs/src/screen"
	"admiralbbs/src/session"
	"admiralbbs/src/store"
	"admiralbbs/src/xfer"
)

// RunFiles drives the file library. Uploads are text/ANSI via paste, binary via
// base64 paste (works on any modern SSH client), or XMODEM; downloads stream the
// decrypted content between markers.
func RunFiles(s *session.Session, st *store.Store, u *store.User, sysopPass string) error {
	handles := newHandleCache(st)
	isSysOp := u.AccessLevel >= CoSysOpLevel
	for {
		cap := s.Cap()
		w := screen.New(s, cap.ANSI, cap.Cols)
		w.Clear()
		w.ColorLine(screen.Cyan, "File Library")
		w.ColorLine(screen.Cyan, "------------")
		areas, err := st.FileAreas().Visible(u.AccessLevel)
		if err != nil {
			return err
		}
		for i, a := range areas {
			w.Color(screen.Yellow)
			w.Printf("  %d) ", i+1)
			w.Color(screen.White)
			w.SafePrint(a.Name)
			w.Reset()
			w.Print("\r\n")
		}
		w.Color(screen.Green)
		if isSysOp {
			w.Print("\r\nArea # ([N]ew area, [Q]uit): ")
		} else {
			w.Print("\r\nArea # (or [Q]uit): ")
		}
		w.Reset()
		in, err := s.ReadLine()
		if err != nil {
			return err
		}
		in = strings.TrimSpace(in)
		if in == "" || strings.EqualFold(in, "q") || strings.EqualFold(in, "x") {
			return nil
		}
		if isSysOp && strings.EqualFold(in, "n") {
			if err := newFileArea(s, st, u, sysopPass); err != nil {
				return err
			}
			continue
		}
		if n, perr := strconv.Atoi(in); perr == nil && n >= 1 && n <= len(areas) {
			if err := browseFileArea(s, st, u, areas[n-1], handles); err != nil {
				return err
			}
		}
	}
}

// newFileArea lets a SysOp create a new File Area from the file menu, mirroring
// the message-board [N]ew board. Password-gated by the SysOp step-up secret when
// one is configured, so an unattended SysOp session can't be used to add areas.
func newFileArea(s *session.Session, st *store.Store, u *store.User, sysopPass string) error {
	cap := s.Cap()
	w := screen.New(s, cap.ANSI, cap.Cols)
	if u.AccessLevel < CoSysOpLevel {
		return nil // defence in depth — the prompt isn't offered to non-SysOps
	}
	if sysopPass != "" {
		w.Color(screen.Green)
		w.Print("\r\nSysOp password: ")
		w.Reset()
		entered, err := s.ReadPassword()
		if err != nil {
			return err
		}
		if !sysopPassOK(entered, sysopPass) {
			s.Activity("filearea-create-denied", "bad sysop password")
			w.ColorLine(screen.Red, "Denied.")
			_, _ = s.ReadKey()
			return nil
		}
	}
	name := prompt(s, w, "New file area name (blank to cancel): ")
	if name == "" {
		return nil
	}
	minLevel := 0
	if v := prompt(s, w, "Min access level [0]: "); v != "" {
		if lvl, e := strconv.Atoi(v); e == nil && lvl >= 0 {
			minLevel = lvl
		}
	}
	if _, err := st.FileAreas().Create(name, minLevel); err != nil {
		w.ColorLine(screen.Red, "could not create area: "+err.Error())
		_, _ = s.ReadKey()
		return nil
	}
	s.Activity("filearea-create", name)
	w.ColorLine(screen.Cyan, "File area created.")
	_, _ = s.ReadKey()
	return nil
}

func browseFileArea(s *session.Session, st *store.Store, u *store.User, area *store.FileArea, handles *handleCache) error {
	sortBy := "newest"
	page := 0
	var listed []*store.FileEntry // non-nil => active search/filter result
	var header string
	for {
		cap := s.Cap()
		w := screen.New(s, cap.ANSI, cap.Cols)

		files := listed
		if files == nil {
			var err error
			files, err = st.Files().ListSorted(area.ID, sortBy)
			if err != nil {
				return err
			}
		}
		lo, hi, pages := pageWindow(len(files), page)
		page = clampPage(page, pages)

		w.Clear()
		w.Color(screen.Cyan)
		w.Print("Files: ")
		w.SafePrint(area.Name)
		w.Printf("   (%s)\r\n", sortBy)
		w.Reset()
		if header != "" {
			w.ColorLine(screen.Blue, header)
		}
		if len(files) == 0 {
			w.Line("  (no files)")
		}
		for i := lo; i < hi; i++ {
			f := files[i]
			w.Color(screen.Yellow)
			w.Printf("  %d) ", i+1)
			w.Color(screen.White)
			w.SafePrint(f.Filename)
			w.Reset()
			w.Printf("  %s (%dx) by %s  ", humanSize(f.SizeBytes), f.DownloadCount, handles.handle(f.UploaderID))
			w.SafePrint(firstLine(f.Description))
			w.Print("\r\n")
		}
		pageFooter(w, page, pages)
		w.Color(screen.Green)
		w.Print("\r\n[#] download  [U]pload  [S]earch  [B] by user  [R] sort  [K] delete  [C]lear  [Q]uit: ")
		w.Reset()
		in, err := s.ReadLine()
		if err != nil {
			return err
		}
		in = strings.TrimSpace(in)
		switch {
		case in == "" || strings.EqualFold(in, "q") || strings.EqualFold(in, "x"):
			return nil
		case in == ">":
			page++
		case in == "<":
			page--
		case strings.EqualFold(in, "u"):
			listed, header, page = nil, "", 0
			if err := uploadFile(s, st, u, area.ID); err != nil {
				return err
			}
		case strings.EqualFold(in, "c"):
			listed, header, page = nil, "", 0
		case strings.EqualFold(in, "r"):
			sortBy = nextFileSort(sortBy)
			listed, header, page = nil, "", 0
		case strings.EqualFold(in, "s"):
			w.Color(screen.Green)
			w.Print("\r\nSearch text: ")
			w.Reset()
			q, _ := s.ReadLine()
			if q = strings.TrimSpace(q); q != "" {
				res, serr := st.Files().Search(area.ID, q)
				if serr != nil {
					return serr
				}
				listed, header, page = res, fmt.Sprintf("search %q — %d hit(s)", q, len(res)), 0
			}
		case strings.EqualFold(in, "b"):
			w.Color(screen.Green)
			w.Print("\r\nFilter by uploader handle: ")
			w.Reset()
			h, _ := s.ReadLine()
			if target, terr := st.Users().ByHandle(strings.TrimSpace(h)); terr == nil {
				res, ferr := st.Files().ByUploader(area.ID, target.ID)
				if ferr != nil {
					return ferr
				}
				listed, header, page = res, fmt.Sprintf("by %s — %d file(s)", target.Handle, len(res)), 0
			} else {
				w.ColorLine(screen.Red, "no such user")
			}
		case strings.EqualFold(in, "k"):
			w.Color(screen.Green)
			w.Print("\r\nDelete file # (blank to cancel): ")
			w.Reset()
			ks, _ := s.ReadLine()
			if n, perr := strconv.Atoi(strings.TrimSpace(ks)); perr == nil && n >= 1 && n <= len(files) {
				f := files[n-1]
				if f.UploaderID == u.ID || u.AccessLevel >= CoSysOpLevel {
					if derr := st.Files().Delete(f.ID); derr != nil {
						w.ColorLine(screen.Red, "delete failed")
					} else {
						s.Activity("delete-file", f.Filename)
						listed, header, page = nil, "", 0
					}
				} else {
					w.ColorLine(screen.Red, "you can only delete your own files")
					_, _ = s.ReadKey()
				}
			}
		default:
			if n, perr := strconv.Atoi(in); perr == nil && n >= 1 && n <= len(files) {
				if err := downloadFile(s, st, files[n-1]); err != nil {
					return err
				}
			}
		}
	}
}

func nextFileSort(cur string) string {
	switch cur {
	case "newest":
		return "oldest"
	case "oldest":
		return "name"
	case "name":
		return "size"
	case "size":
		return "downloads"
	default:
		return "newest"
	}
}

func downloadFile(s *session.Session, st *store.Store, f *store.FileEntry) error {
	content, err := st.Files().Content(f.ID) // decrypts + bumps the counter
	if err != nil {
		return nil
	}
	cap := s.Cap()
	w := screen.New(s, cap.ANSI, cap.Cols)
	w.Clear()
	w.Color(screen.Green)
	w.Print("[X]MODEM transfer or [V]iew inline? ")
	w.Reset()
	mode, err := s.ReadKey()
	if err != nil {
		return err
	}
	s.Activity("download-file", f.Filename)
	if toLower(mode) == 'x' {
		w.Line("\r\nStart your XMODEM receive now...")
		if xerr := xfer.Send(s.Raw(), content); xerr != nil {
			w.ColorLine(screen.Red, "\r\ntransfer failed: "+xerr.Error())
		} else {
			w.ColorLine(screen.Cyan, "\r\ntransfer complete.")
		}
	} else {
		w.ColorLine(screen.Cyan, "----- BEGIN "+f.Filename+" -----")
		s.Write(content)
		w.Print("\r\n")
		w.ColorLine(screen.Cyan, "----- END "+f.Filename+" -----")
	}
	w.Color(screen.Green)
	w.Print("\r\nPress any key to continue...")
	w.Reset()
	_, err = s.ReadKey()
	return err
}

func uploadFile(s *session.Session, st *store.Store, u *store.User, areaID int64) error {
	cap := s.Cap()
	w := screen.New(s, cap.ANSI, cap.Cols)
	w.Print("\r\n")
	w.Color(screen.Green)
	w.Print("Filename: ")
	w.Reset()
	filename, err := s.ReadLine()
	if err != nil {
		return err
	}
	if strings.TrimSpace(filename) == "" {
		return nil
	}
	w.Color(screen.Green)
	w.Print("Description: ")
	w.Reset()
	desc, err := s.ReadLine()
	if err != nil {
		return err
	}
	w.Color(screen.Green)
	w.Print("[B]ase64 (binary paste), [P]aste text, or [X]MODEM? ")
	w.Reset()
	mode, err := s.ReadKey()
	if err != nil {
		return err
	}
	var content []byte
	switch toLower(mode) {
	case 'x':
		w.Line("\r\nStart your XMODEM send now...")
		data, xerr := xfer.Receive(s.Raw(), store.MaxFileBytes)
		if xerr != nil {
			w.ColorLine(screen.Red, "\r\nupload failed: "+xerr.Error())
			return nil
		}
		content = data
	case 'b':
		// Base64 binary paste — works on any modern SSH client (no sz/rz). Paste
		// `base64 <file>` output; we strip whitespace and decode.
		w.Line("\r\nPaste the file's base64 (e.g. `base64 file`). End with a single '.' on its own line.")
		var sb strings.Builder
		for {
			line, err := s.ReadLine()
			if err != nil {
				return err
			}
			if line == "." {
				break
			}
			sb.WriteString(strings.TrimSpace(line))
		}
		data, derr := base64.StdEncoding.DecodeString(sb.String())
		if derr != nil {
			w.ColorLine(screen.Red, "that isn't valid base64 — try again: "+derr.Error())
			return nil
		}
		content = data
	default:
		w.Line("\r\nPaste the file contents (text/ANSI). End with a single '.' on its own line.")
		var lines []string
		for {
			line, err := s.ReadLine()
			if err != nil {
				return err
			}
			if line == "." {
				break
			}
			lines = append(lines, line)
		}
		content = []byte(strings.Join(lines, "\n"))
	}
	if _, err := st.Files().Add(areaID, u.ID, strings.TrimSpace(filename), desc, content); err != nil {
		if err == store.ErrTooLarge {
			w.ColorLine(screen.Red, "too large — limit is 10 MiB")
		} else if err == store.ErrQuotaExceeded {
			w.ColorLine(screen.Red, "over your storage quota (100 MiB) — delete some files first")
		} else if err == store.ErrDuplicateName {
			w.ColorLine(screen.Red, "a file with that name already exists here — delete it first to replace")
		} else {
			w.ColorLine(screen.Red, "upload failed: "+err.Error())
		}
		return nil
	}
	s.Activity("upload-file", strings.TrimSpace(filename))
	w.ColorLine(screen.Cyan, "Uploaded.")
	return nil
}

func humanSize(b int64) string {
	switch {
	case b >= 1<<20:
		return fmt.Sprintf("%.1fMB", float64(b)/(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%.1fKB", float64(b)/(1<<10))
	default:
		return fmt.Sprintf("%dB", b)
	}
}
