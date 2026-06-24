package menu

import "testing"

func TestPageWindow(t *testing.T) {
	cases := []struct {
		n, page              int
		wantLo, wantHi, want int // want = pages
	}{
		{0, 0, 0, 0, 1},        // empty -> one (empty) page
		{10, 0, 0, 10, 1},      // fits on one page
		{pageSize, 0, 0, 15, 1},// exactly full single page
		{16, 0, 0, 15, 2},      // spills to a second page
		{16, 1, 15, 16, 2},     // second page has the remainder
		{31, 2, 30, 31, 3},     // last page partial
		{31, 9, 30, 31, 3},     // page past the end clamps to last
		{31, -5, 0, 15, 3},     // negative clamps to first
	}
	for _, c := range cases {
		lo, hi, pages := pageWindow(c.n, c.page)
		if lo != c.wantLo || hi != c.wantHi || pages != c.want {
			t.Errorf("pageWindow(%d,%d) = (%d,%d,%d), want (%d,%d,%d)",
				c.n, c.page, lo, hi, pages, c.wantLo, c.wantHi, c.want)
		}
	}
}

func TestClampPage(t *testing.T) {
	if clampPage(-1, 3) != 0 {
		t.Error("negative should clamp to 0")
	}
	if clampPage(5, 3) != 2 {
		t.Error("over-range should clamp to pages-1")
	}
	if clampPage(1, 3) != 1 {
		t.Error("in-range should pass through")
	}
}
