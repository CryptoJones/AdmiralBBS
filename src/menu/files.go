package menu

import (
	"fmt"
	"strconv"
	"strings"

	"admiralbbs/src/screen"
	"admiralbbs/src/session"
	"admiralbbs/src/store"
)

// RunFiles drives the file library. Uploads are text/ANSI via paste (binary
// X/Y/Zmodem transfer is a planned follow-on); downloads stream the decrypted
// content between markers.
func RunFiles(s *session.Session, st *store.Store, u *store.User) error {
	handles := newHandleCache(st)
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
		w.Print("\r\nArea # (or [Q]uit): ")
		w.Reset()
		in, err := s.ReadLine()
		if err != nil {
			return err
		}
		in = strings.TrimSpace(in)
		if in == "" || strings.EqualFold(in, "q") {
			return nil
		}
		if n, perr := strconv.Atoi(in); perr == nil && n >= 1 && n <= len(areas) {
			if err := browseFileArea(s, st, u, areas[n-1], handles); err != nil {
				return err
			}
		}
	}
}

func browseFileArea(s *session.Session, st *store.Store, u *store.User, area *store.FileArea, handles *handleCache) error {
	for {
		cap := s.Cap()
		w := screen.New(s, cap.ANSI, cap.Cols)
		w.Clear()
		w.Color(screen.Cyan)
		w.Print("Files: ")
		w.SafePrint(area.Name)
		w.Print("\r\n")
		w.Reset()
		files, err := st.Files().ListByArea(area.ID)
		if err != nil {
			return err
		}
		if len(files) == 0 {
			w.Line("  (no files yet — be the first to upload)")
		}
		for i, f := range files {
			w.Color(screen.Yellow)
			w.Printf("  %d) ", i+1)
			w.Color(screen.White)
			w.SafePrint(f.Filename)
			w.Reset()
			w.Printf("  %s  (%dx)  ", humanSize(f.SizeBytes), f.DownloadCount)
			w.SafePrint(firstLine(f.Description))
			w.Print("\r\n")
		}
		w.Color(screen.Green)
		w.Print("\r\n[#] download  [U]pload  [Q]uit: ")
		w.Reset()
		in, err := s.ReadLine()
		if err != nil {
			return err
		}
		in = strings.TrimSpace(in)
		switch {
		case in == "" || strings.EqualFold(in, "q"):
			return nil
		case strings.EqualFold(in, "u"):
			if err := uploadFile(s, st, u, area.ID); err != nil {
				return err
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

func downloadFile(s *session.Session, st *store.Store, f *store.FileEntry) error {
	content, err := st.Files().Content(f.ID) // decrypts + bumps the counter
	if err != nil {
		return nil
	}
	cap := s.Cap()
	w := screen.New(s, cap.ANSI, cap.Cols)
	w.Clear()
	w.ColorLine(screen.Cyan, "----- BEGIN "+f.Filename+" -----")
	s.Write(content)
	w.Print("\r\n")
	w.ColorLine(screen.Cyan, "----- END "+f.Filename+" -----")
	s.Activity("download-file", f.Filename)
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
	w.Line("Paste the file contents (text/ANSI). End with a single '.' on its own line.")
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
	content := []byte(strings.Join(lines, "\n"))
	if _, err := st.Files().Add(areaID, u.ID, strings.TrimSpace(filename), desc, content); err != nil {
		if err == store.ErrTooLarge {
			w.ColorLine(screen.Red, "too large — limit is 10 MiB")
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
