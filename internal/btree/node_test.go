package btree

import (
	"testing"

	"github.com/akselleirv/database/internal/pager"
)

// roundTrip encodes n into a fresh page and decodes it back.
func roundTrip(t *testing.T, n *node) *node {
	t.Helper()
	buf := make([]byte, pager.PageSize)
	n.encode(buf)
	got, err := decodeNode(buf)
	if err != nil {
		t.Fatalf("decodeNode: %v", err)
	}
	return got
}

func TestLeafRoundTrip(t *testing.T) {
	n := &node{
		typ:  leafNode,
		next: 42,
		entries: []entry{
			{key: -5, value: 100},
			{key: 0, value: 200},
			{key: 7, value: 300},
			{key: 9000, value: 400},
		},
	}
	got := roundTrip(t, n)

	if got.typ != leafNode {
		t.Fatalf("typ = %d, want leaf", got.typ)
	}
	if got.next != 42 {
		t.Errorf("next = %d, want 42", got.next)
	}
	if len(got.entries) != len(n.entries) {
		t.Fatalf("len(entries) = %d, want %d", len(got.entries), len(n.entries))
	}
	for i, e := range n.entries {
		if got.entries[i] != e {
			t.Errorf("entry[%d] = %+v, want %+v", i, got.entries[i], e)
		}
	}
}

func TestLeafEmptyRoundTrip(t *testing.T) {
	n := &node{typ: leafNode, next: 0}
	got := roundTrip(t, n)
	if got.typ != leafNode {
		t.Fatalf("typ = %d, want leaf", got.typ)
	}
	if len(got.entries) != 0 {
		t.Errorf("len(entries) = %d, want 0", len(got.entries))
	}
	if got.next != 0 {
		t.Errorf("next = %d, want 0", got.next)
	}
}

func TestInternalRoundTrip(t *testing.T) {
	n := &node{
		typ:      internalNode,
		keys:     []int64{10, 20, 30},
		children: []pager.PageID{1, 2, 3, 4}, // numKeys+1
	}
	got := roundTrip(t, n)

	if got.typ != internalNode {
		t.Fatalf("typ = %d, want internal", got.typ)
	}
	if len(got.keys) != len(n.keys) {
		t.Fatalf("len(keys) = %d, want %d", len(got.keys), len(n.keys))
	}
	for i, k := range n.keys {
		if got.keys[i] != k {
			t.Errorf("key[%d] = %d, want %d", i, got.keys[i], k)
		}
	}
	if len(got.children) != len(n.children) {
		t.Fatalf("len(children) = %d, want %d", len(got.children), len(n.children))
	}
	for i, c := range n.children {
		if got.children[i] != c {
			t.Errorf("child[%d] = %d, want %d", i, got.children[i], c)
		}
	}
}

func TestSearch(t *testing.T) {
	keys := []int64{10, 20, 30, 40}
	cases := []struct {
		key       int64
		wantIdx   int
		wantFound bool
	}{
		{10, 0, true},   // first
		{30, 2, true},   // middle
		{40, 3, true},   // last
		{5, 0, false},   // before first
		{15, 1, false},  // between 10 and 20
		{25, 2, false},  // between 20 and 30
		{100, 4, false}, // after last
	}
	for _, c := range cases {
		idx, found := search(keys, c.key)
		if idx != c.wantIdx || found != c.wantFound {
			t.Errorf("search(%d) = (%d,%v), want (%d,%v)",
				c.key, idx, found, c.wantIdx, c.wantFound)
		}
	}
}

func TestSearchEmpty(t *testing.T) {
	idx, found := search(nil, 5)
	if idx != 0 || found {
		t.Errorf("search(nil,5) = (%d,%v), want (0,false)", idx, found)
	}
}
