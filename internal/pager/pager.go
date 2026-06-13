// Package pager turns a single file into a fixed-size array of 4 KB pages and
// is the durability boundary for the engine: every byte that must survive a
// crash passes through here.
package pager

import (
	"errors"
	"fmt"
	"os"
)

// PageSize is the fixed size of every page, in bytes.
const PageSize = 4096

// PageID addresses a page within the file. The byte offset of a page is
// id * PageSize. Page 0 is reserved as the meta/header page (see lesson 01).
type PageID uint32

// Pager is a fixed-size-page view over a single file. It is not safe for
// concurrent use (we stay single-threaded until the MVCC lesson).
type Pager struct {
	file      *os.File
	pageCount PageID
}

// Open opens (or creates) the database file at path and prepares it for paged
// access.
//
// On a fresh file it establishes the reserved meta page (page 0: magic number +
// page size). On an existing file it validates that header and recovers the page
// count. If the file length is not a whole number of pages, Open returns an
// error (we do not truncate/repair — that is the WAL's job later).
func Open(path string) (*Pager, error) {
	// Check if we need to create the file
	info, err := os.Stat(path)

	switch {
	case errors.Is(err, os.ErrNotExist):
		// must create the file
		f, err := os.Create(path)
		if err != nil {
			return nil, err
		}

		// create metadata page
		p := &Pager{file: f, pageCount: 1}
		if err := p.writeMeta(); err != nil {
			return nil, err
		}
		// Ensure that the file is created on a fresh database
		if err := p.Sync(); err != nil {
			return nil, err
		}
		return p, nil
	case err != nil:
		return nil, err
	default:
		// Already exists

		// Assert expected size
		if info.Size()%PageSize != 0 {
			return nil, fmt.Errorf("pager: file size %d not a multiple of %d", info.Size(), PageSize)
		}

		f, err := os.OpenFile(path, os.O_RDWR, 0o644)
		if err != nil {
			return nil, err
		}

		// Extract the page count
		buf := make([]byte, PageSize)
		_, err = f.ReadAt(buf, 0)
		if err != nil {
			return nil, err
		}

		m, err := decodeMeta(buf)
		if err != nil {
			if err := f.Close(); err != nil {
				return nil, err
			}
			return nil, err
		}

		return &Pager{file: f, pageCount: PageID(m.pageCount)}, nil
	}

}

// ReadPage reads the page with the given id into dst. dst must be exactly
// PageSize bytes. Reading an unallocated page (id >= PageCount) returns an
// error; there is no zero-fill-on-read.
func (p *Pager) ReadPage(id PageID, dst []byte) error {
	if len(dst) != PageSize {
		return fmt.Errorf("invalid dst size, expected %d got %d", PageSize, len(dst))
	}
	if id >= p.pageCount {
		return fmt.Errorf("invalid page id %d, page count is %d", id, p.pageCount)
	}

	_, err := p.file.ReadAt(dst, int64(id)*PageSize)
	return err
}

// WritePage writes the PageSize bytes in src to the page with the given id.
// src must be exactly PageSize bytes. WritePage does NOT fsync; call Sync to
// make writes durable.
func (p *Pager) WritePage(id PageID, src []byte) error {
	if len(src) != PageSize {
		return fmt.Errorf("invalid page size, expected %d got %d", PageSize, len(src))
	}

	_, err := p.file.WriteAt(src, int64(id)*PageSize)
	if err != nil {
		return err
	}

	return nil
}

// AllocatePage grows the file by one zero-filled page and returns the new
// page's id, keeping file length == PageCount * PageSize.
func (p *Pager) AllocatePage() (PageID, error) {
	buf := make([]byte, PageSize)
	id := p.pageCount
	if _, err := p.file.WriteAt(buf, int64(id)*PageSize); err != nil {
		return 0, err
	}
	p.pageCount++
	// Commit the new count: the meta page is the commit point, written after
	// the data page so a crash in between leaves the extra page orphaned, never
	// a meta that claims a page the file lacks.
	if err := p.writeMeta(); err != nil {
		return 0, err
	}
	return id, nil
}

// writeMeta encodes the pager's current state into page 0. It is the single
// source of truth for persisting bootstrap state; callers that change
// pageCount must call it to keep disk in sync.
func (p *Pager) writeMeta() error {
	buf := make([]byte, PageSize)
	meta{pageSize: PageSize, pageCount: uint32(p.pageCount)}.encode(buf)
	_, err := p.file.WriteAt(buf, 0)
	return err
}

// PageCount returns the number of pages currently in the file, including the
// reserved meta page.
func (p *Pager) PageCount() PageID {
	return p.pageCount
}

// Sync flushes buffered writes durably to disk (fsync).
func (p *Pager) Sync() error {
	return p.file.Sync()
}

// Close syncs and closes the underlying file. After Close the Pager must not
// be used again.
func (p *Pager) Close() error {
	if err := p.Sync(); err != nil {
		return err
	}

	return p.file.Close()
}
