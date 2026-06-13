// Package btree implements the B+ tree that backs a table: keys are int64,
// values are opaque uint64 payloads, and one node occupies one pager page.
package btree

import "github.com/akselleirv/database/internal/pager"

// nodeType distinguishes the two kinds of node stored in a page.
type nodeType uint8

const (
	internalNode nodeType = 0
	leafNode     nodeType = 1
)

// On-disk layout (big-endian), within a single pager.PageSize page:
//
//	Common header (8 bytes):
//	  off 0: type     uint8   (0=internal, 1=leaf)
//	  off 1: numKeys  uint16
//	  off 3: padding  (5 bytes, reserved)
//
//	Leaf body (starts at off 8):
//	  off 8:  next    uint32  (PageID of the next leaf; 0 = none)
//	  off 12: entries[numKeys], each: key int64 (8) + value uint64 (8) = 16 bytes
//
//	Internal body (starts at off 8):
//	  off 8:  keys[numKeys]        each int64  (8 bytes)
//	  then:   children[numKeys+1]  each uint32 (4 bytes, a PageID)
//
// Invariant for internal nodes: child[i] holds keys < keys[i]; child[i+1] holds
// keys >= keys[i].
const (
	nodeHeaderSize = 8
	leafEntrySize  = 16 // int64 key + uint64 value
	keySize        = 8
	childSize      = 4 // a PageID
)

// node is the in-memory form of a tree node (one page). The active fields
// depend on typ: leaves use entries+next, internals use keys+children.
type node struct {
	typ nodeType

	// leaf fields
	entries []entry
	next    pager.PageID

	// internal fields
	keys     []int64
	children []pager.PageID
}

// entry is one key/value pair in a leaf.
type entry struct {
	key   int64
	value uint64
}

// encode serializes n into dst, which must be pager.PageSize bytes. It writes
// the header and the type-appropriate body, leaving unused tail bytes zeroed.
func (n *node) encode(dst []byte) {
	// TODO
}

// decodeNode parses a node from src (pager.PageSize bytes). It reads the header
// to learn the type and key count, then the matching body.
func decodeNode(src []byte) (*node, error) {
	// TODO
	return nil, nil
}

// search returns the index of key within a sorted slice of keys and whether it
// was found. On a miss it returns the index where key would be inserted to keep
// the slice sorted (0..len). This is the primitive both leaf lookup and
// internal routing are built on in Part 2.
//
// Hint: this is a binary search; sort.Search from the standard library does the
// heavy lifting, but think about what it returns on an exact hit vs. a miss.
func search(keys []int64, key int64) (idx int, found bool) {
	// TODO
	return 0, false
}
