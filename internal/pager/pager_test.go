package pager

import (
	"bytes"
	"path/filepath"
	"testing"
)

// tmpDB returns a path to a fresh database file inside the test's temp dir.
func tmpDB(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "test.db")
}

// makePage returns a PageSize buffer filled with byte b, for distinguishable
// page contents.
func makePage(b byte) []byte {
	p := make([]byte, PageSize)
	for i := range p {
		p[i] = b
	}
	return p
}

// --- Invariant behavior: any correct implementation must pass these. ---

func TestOpenFreshReservesMetaPage(t *testing.T) {
	p, err := Open(tmpDB(t))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer p.Close()

	// A fresh database is not empty: page 0 is the reserved meta page, so the
	// file must already account for at least one page.
	if got := p.PageCount(); got < 1 {
		t.Fatalf("PageCount on fresh db = %d, want >= 1 (meta page)", got)
	}
}

func TestAllocateReturnsDistinctIncreasingIDs(t *testing.T) {
	p, err := Open(tmpDB(t))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer p.Close()

	a, err := p.AllocatePage()
	if err != nil {
		t.Fatalf("AllocatePage: %v", err)
	}
	b, err := p.AllocatePage()
	if err != nil {
		t.Fatalf("AllocatePage: %v", err)
	}
	if a == 0 {
		t.Errorf("AllocatePage returned reserved id 0")
	}
	if b <= a {
		t.Errorf("AllocatePage ids not increasing: got %d then %d", a, b)
	}
}

func TestAllocateGrowsPageCount(t *testing.T) {
	p, err := Open(tmpDB(t))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer p.Close()

	before := p.PageCount()
	if _, err := p.AllocatePage(); err != nil {
		t.Fatalf("AllocatePage: %v", err)
	}
	if got := p.PageCount(); got != before+1 {
		t.Errorf("PageCount after one alloc = %d, want %d", got, before+1)
	}
}

func TestWriteReadRoundTrip(t *testing.T) {
	p, err := Open(tmpDB(t))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer p.Close()

	id, err := p.AllocatePage()
	if err != nil {
		t.Fatalf("AllocatePage: %v", err)
	}
	want := makePage(0xAB)
	if err := p.WritePage(id, want); err != nil {
		t.Fatalf("WritePage: %v", err)
	}

	got := make([]byte, PageSize)
	if err := p.ReadPage(id, got); err != nil {
		t.Fatalf("ReadPage: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("round-trip mismatch")
	}
}

func TestPersistenceAcrossReopen(t *testing.T) {
	path := tmpDB(t)

	p, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	id, err := p.AllocatePage()
	if err != nil {
		t.Fatalf("AllocatePage: %v", err)
	}
	want := makePage(0xCD)
	if err := p.WritePage(id, want); err != nil {
		t.Fatalf("WritePage: %v", err)
	}
	countBefore := p.PageCount()
	if err := p.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Reopen: page count and contents must survive.
	p2, err := Open(path)
	if err != nil {
		t.Fatalf("reopen Open: %v", err)
	}
	defer p2.Close()

	if got := p2.PageCount(); got != countBefore {
		t.Errorf("PageCount after reopen = %d, want %d", got, countBefore)
	}
	got := make([]byte, PageSize)
	if err := p2.ReadPage(id, got); err != nil {
		t.Fatalf("ReadPage after reopen: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("contents not durable across reopen")
	}
}

func TestWriteReadRejectWrongBufferSize(t *testing.T) {
	p, err := Open(tmpDB(t))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer p.Close()

	id, err := p.AllocatePage()
	if err != nil {
		t.Fatalf("AllocatePage: %v", err)
	}

	// A buffer that isn't exactly PageSize is a programming error and must be
	// rejected, not silently truncated or padded.
	if err := p.WritePage(id, make([]byte, PageSize-1)); err == nil {
		t.Errorf("WritePage with short src: want error, got nil")
	}
	if err := p.ReadPage(id, make([]byte, PageSize+1)); err == nil {
		t.Errorf("ReadPage with long dst: want error, got nil")
	}
}

// --- Design-choice placeholders: YOU decide the contract, then write these. ---
//
// The lesson leaves these open on purpose. Once you've made the call, replace
// the t.Skip with assertions that pin your chosen behavior.

func TestReadPastEOF(t *testing.T) {
	t.Skip("DESIGN CHOICE (Q3): decide whether reading an unallocated/past-EOF " +
		"page is an error or returns a zero-filled page, then assert it here.")
}

func TestOpenRejectsOrRepairsTornFile(t *testing.T) {
	t.Skip("DESIGN CHOICE (Q2): decide what Open does when the file length is " +
		"not a whole number of pages (error / truncate / round down). Create " +
		"such a file with os.WriteFile of a non-multiple length, then assert it.")
}

func TestDurabilityContract(t *testing.T) {
	t.Skip("DESIGN CHOICE (Q5): document whether WritePage fsyncs or defers to " +
		"Sync. Hard to assert portably, but at minimum assert Sync() returns nil " +
		"after writes and the reopen test above covers the persisted path.")
}
