package pager

import (
	"encoding/binary"
	"fmt"
)

// The meta page (page 0) holds the engine's bootstrap state: the few values you
// must know before you can interpret the rest of the file. It occupies a full
// page but currently uses only its first bytes; the remainder is reserved for
// later (root PageID, free-list head).

// magic identifies a file as ours. Open rejects any file whose page 0 does not
// start with this value. (Pick any distinctive constant; treat it as the file
// format's signature.)
const magic uint32 = 0xD1B5DA7A // "DB" + nonsense — change if you like

// Field offsets within the meta page.
const (
	metaMagicOffset     = 0 // uint32
	metaPageSizeOffset  = 4 // uint32
	metaPageCountOffset = 8 // uint32
)

// meta is the in-memory form of page 0.
type meta struct {
	pageSize  uint32
	pageCount uint32
}

// encode writes m into dst, which must be PageSize bytes. It writes the magic,
// page size, and page count at their fixed offsets and leaves the rest zeroed.
// Use encoding/binary.BigEndian.
func (m meta) encode(dst []byte) {
	binary.BigEndian.PutUint32(dst[metaMagicOffset:], magic)
	binary.BigEndian.PutUint32(dst[metaPageSizeOffset:], m.pageSize)
	binary.BigEndian.PutUint32(dst[metaPageCountOffset:], m.pageCount)
}

// decodeMeta parses a meta page from src (PageSize bytes). It returns an error
// if the magic does not match (not our file / corrupt) or the stored page size
// differs from the compiled PageSize (incompatible layout).
func decodeMeta(src []byte) (meta, error) {
	if got := binary.BigEndian.Uint32(src[metaMagicOffset:]); got != magic {
		return meta{}, fmt.Errorf("pager: bad magic %#x (not a database file)", got)
	}

	m := meta{
		pageSize:  binary.BigEndian.Uint32(src[metaPageSizeOffset:]),
		pageCount: binary.BigEndian.Uint32(src[metaPageCountOffset:]),
	}
	if m.pageSize != PageSize {
		return meta{}, fmt.Errorf("pager: page size %d != compiled %d", m.pageSize, PageSize)
	}
	return m, nil
}
