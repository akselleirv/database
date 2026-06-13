// Package pager turns a single file into a fixed-size array of 4 KB pages and
// is the durability boundary for the engine: every byte that must survive a
// crash passes through here.
package pager

import "os"

// PageSize is the fixed size of every page, in bytes.
const PageSize = 4096

// PageID addresses a page within the file. The byte offset of a page is
// id * PageSize. Page 0 is reserved as the meta/header page (see lesson 01).
type PageID uint32

// Pager is a fixed-size-page view over a single file. It is not safe for
// concurrent use (we stay single-threaded until the MVCC lesson).
type Pager struct {
	file *os.File
	// TODO: what else must you track? At minimum you need the page count.
	// Where does it come from on Open, and when does it change?
}

// Open opens (or creates) the database file at path and prepares it for paged
// access.
//
// On a fresh file it establishes the reserved meta page (page 0: magic number +
// page size). On an existing file it validates that header and recovers the page
// count. If the file length is not a whole number of pages, Open returns an
// error (we do not truncate/repair — that is the WAL's job later).
func Open(path string) (*Pager, error) {
	// TODO
	return nil, nil
}

// ReadPage reads the page with the given id into dst. dst must be exactly
// PageSize bytes. Reading an unallocated page (id >= PageCount) returns an
// error; there is no zero-fill-on-read.
func (p *Pager) ReadPage(id PageID, dst []byte) error {
	// TODO
	return nil
}

// WritePage writes the PageSize bytes in src to the page with the given id.
// src must be exactly PageSize bytes. WritePage does NOT fsync; call Sync to
// make writes durable.
func (p *Pager) WritePage(id PageID, src []byte) error {
	// TODO
	return nil
}

// AllocatePage grows the file by one zero-filled page and returns the new
// page's id, keeping file length == PageCount * PageSize.
func (p *Pager) AllocatePage() (PageID, error) {
	// TODO
	return 0, nil
}

// PageCount returns the number of pages currently in the file, including the
// reserved meta page.
func (p *Pager) PageCount() PageID {
	// TODO
	return 0
}

// Sync flushes buffered writes durably to disk (fsync).
func (p *Pager) Sync() error {
	// TODO
	return nil
}

// Close syncs and closes the underlying file. After Close the Pager must not
// be used again.
func (p *Pager) Close() error {
	// TODO
	return nil
}
