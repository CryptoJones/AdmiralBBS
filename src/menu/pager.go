package menu

import "admiralbbs/src/screen"

// pageSize is how many list rows fit on a classic 80x25 screen with room for
// the header and the command footer.
const pageSize = 15

// pageWindow returns the [lo, hi) slice bounds for the given 0-based page over a
// list of n items, plus the total page count. page is clamped into range, so a
// caller can blindly increment/decrement and pass it back in.
func pageWindow(n, page int) (lo, hi, pages int) {
	pages = (n + pageSize - 1) / pageSize
	if pages < 1 {
		pages = 1
	}
	if page < 0 {
		page = 0
	}
	if page >= pages {
		page = pages - 1
	}
	lo = page * pageSize
	hi = lo + pageSize
	if hi > n {
		hi = n
	}
	return lo, hi, pages
}

// clampPage keeps a page index in [0, pages-1].
func clampPage(page, pages int) int {
	if page < 0 {
		return 0
	}
	if page >= pages {
		return pages - 1
	}
	return page
}

// pageFooter renders a "Page X/Y" line when there's more than one page. The
// [>] next / [<] prev keys are used (rather than letters) so they never collide
// with a view's own commands like [P]ost or [B]lock.
func pageFooter(w *screen.Writer, page, pages int) {
	if pages <= 1 {
		return
	}
	w.Color(screen.Blue)
	w.Printf("  -- page %d/%d --  [>] next  [<] prev\r\n", page+1, pages)
	w.Reset()
}
