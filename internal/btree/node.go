// Package btree implements the B+ tree that backs a table: keys are int64,
// values are opaque uint64 payloads, and one node occupies one pager page.
package btree

import (
	"encoding/binary"
	"fmt"
	"sort"

	"github.com/akselleirv/database/internal/pager"
)

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
	// --- Header (worked example — study the pattern, then mirror it below) ---
	dst[0] = byte(n.typ)
	// numKeys is len(entries) for a leaf, len(keys) for an internal node.
	// They're the same count conceptually; pick the one that's populated.
	numKeys := len(n.keys)
	if n.typ == leafNode {
		numKeys = len(n.entries)
	}
	binary.BigEndian.PutUint16(dst[1:], uint16(numKeys))
	// bytes [3:8] stay zero (reserved padding)

	off := nodeHeaderSize
	if n.typ == leafNode {
		binary.BigEndian.PutUint32(dst[off:], uint32(n.next))
		for i, entry := range n.entries {
			// nodeHeaderSize + next (4) + i*leafEntrySize
			off = nodeHeaderSize + 4 + i*leafEntrySize
			binary.BigEndian.PutUint64(dst[off:], uint64(entry.key))
			// off + keySize (8)
			binary.BigEndian.PutUint64(dst[off+8:], entry.value)
		}
		return
	}

	// internal body:
	//   keys first, then children (see the layout block above).
	for i, key := range n.keys {
		off = nodeHeaderSize + i*keySize
		binary.BigEndian.PutUint64(dst[off:], uint64(key))
	}

	for j, child := range n.children {
		//   child j    at off = nodeHeaderSize + numKeys*keySize + j*childSize  (uint32)
		off = nodeHeaderSize + numKeys*keySize + j*childSize
		binary.BigEndian.PutUint32(dst[off:], uint32(child))
	}
}

// decodeNode parses a node from src (pager.PageSize bytes). It reads the header
// to learn the type and key count, then the matching body.
func decodeNode(src []byte) (*node, error) {
	if len(src) < nodeHeaderSize {
		return nil, fmt.Errorf("node cannot be less than 8 bytes")
	}
	n := &node{
		typ: nodeType(src[0]),
	}

	off := nodeHeaderSize
	numKeys := int(binary.BigEndian.Uint16(src[1:]))
	if n.typ == leafNode {
		n.next = pager.PageID(binary.BigEndian.Uint32(src[off:]))
		n.entries = make([]entry, numKeys)
		for i := range numKeys {
			// nodeHeaderSize + next (4) + i*leafEntrySize
			off = nodeHeaderSize + 4 + i*leafEntrySize
			n.entries[i] = entry{
				key:   int64(binary.BigEndian.Uint64(src[off:])),
				value: binary.BigEndian.Uint64(src[off+8:]),
			}
		}
		return n, nil
	}

	n.keys = make([]int64, numKeys)
	for i := range n.keys {
		off = nodeHeaderSize + i*keySize
		n.keys[i] = int64(binary.BigEndian.Uint64(src[off:]))
	}

	n.children = make([]pager.PageID, numKeys+1)
	for j := range n.children {
		off = nodeHeaderSize + numKeys*keySize + j*childSize
		n.children[j] = pager.PageID(binary.BigEndian.Uint32(src[off:]))
	}

	return n, nil
}

// search returns the index of key within a sorted slice of keys and whether it
// was found. On a miss it returns the index where key would be inserted to keep
// the slice sorted (0..len). This is the primitive both leaf lookup and
// internal routing are built on in Part 2.
//
// Hint: this is a binary search; sort.Search from the standard library does the
// heavy lifting, but think about what it returns on an exact hit vs. a miss.
func search(keys []int64, key int64) (idx int, found bool) {
	idx = sort.Search(len(keys), func(i int) bool {
		return keys[i] >= key
	})
	found = idx < len(keys) && keys[idx] == key
	return idx, found
}
