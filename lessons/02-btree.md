# Lesson 02 — B+ Tree

This is the heart of the engine. In a SQLite-lineage database **the table *is* a
B+ tree**: the tree's keys are the primary keys, its leaves hold the rows, and
"scan the table" means "walk the leaves left to right." Indexes are just more B+
trees. Get this layer right and everything above it is plumbing.

We build it in three parts, each green before the next:

1. **Node format** (this part) — lay a tree node out in one 4 KB page and
   round-trip it through `encode`/`decode`. No tree logic yet.
2. **Search + Insert** — point lookup, insertion, and node splits; the tree
   grows in height through the root.
3. **Delete + Range** — deletion with merge/borrow, and ordered range scans
   across the linked leaves.

## What a B+ tree is (the 60-second version)

A balanced search tree where **all data lives in the leaves** and internal nodes
hold only *separator keys* that route you down. Two node types:

- **Leaf**: a sorted array of `(key, value)` pairs. Leaves are chained
  left-to-right (`next` pointer) so a range scan is a linked-list walk — no tree
  traversal per row.
- **Internal**: `n` separator keys and `n+1` child pointers. To find a key you
  binary-search the separators and follow the matching child. Separators are
  *copies* of keys that also live in some leaf (that's the "+" in B+: keys may
  appear twice, values only in leaves).

Why a B+ tree and not a binary tree or a hashmap:

- **Fanout matches the disk.** One node = one page = one I/O. With ~255 entries
  per 4 KB node, a tree of a few levels indexes millions of rows in 3–4 page
  reads. A binary tree would be ~20+ reads for the same data.
- **Sorted leaves give you range scans and `ORDER BY` for free.** A hashmap
  can't.
- **It stays balanced** under insert/delete, so worst-case lookup is bounded.

## Decisions (decided — just implement)

1. **Keys are `int64`, values are `uint64`, keys are unique.** Fixed-size
   entries keep node layout a flat array — no slotted page yet. The `uint64`
   value is an opaque payload (think "row location"); we generalize to
   variable-length tuples in Lesson 03 (slotted pages). Inserting an existing key
   *updates* its value.

2. **Node = one page.** Two types, distinguished by a type byte in the header.
   A node never spans pages; that's what bounds fanout.

3. **Common 8-byte header**: `type` (1 byte: 0=internal, 1=leaf), `numKeys`
   (uint16), then type-specific fields. We pad to 8 bytes so entry arrays start
   aligned and there's room to grow the header later.

4. **Leaf layout**: header, then a `next` leaf `PageID` (uint32; 0 = none, safe
   because page 0 is the meta page and never a leaf), then `numKeys` entries of
   `(key int64, value uint64)` = 16 bytes each, sorted ascending by key. Max ≈
   255 entries.

5. **Internal layout**: header, then `numKeys` separator keys (`int64`), then
   `numKeys+1` child `PageID`s (uint32). Invariant: `child[i]` holds keys
   `< key[i]`; `child[i+1]` holds keys `>= key[i]`. (Right-biased: equal keys go
   right.) Max ≈ 340 keys.

6. **Endianness: big-endian**, consistent with the pager's meta page.

7. **The root lives in the meta page.** An empty tree is a single empty leaf that
   is also the root. When the root splits, we allocate a new internal node and
   record its `PageID` as the new root — this is the only way the tree gains
   height. (We'll extend the pager's meta page with a root field in Part 2; Part
   1 is just the node format.)

## This part's contract

In `internal/btree/node.go`: a `node` type that is the in-memory form of a page,
with `encode(dst)` and `decodeNode(src)` mirroring the pager's meta pattern.
Sorted-array helpers (find the slot for a key) live here too — they're the
primitives Part 2's search and insert call.

Standard library only, idiomatic errors, no panics.

## Done when

`make test` is green for `internal/btree`: a leaf and an internal node survive an
`encode`→`decode` round-trip with all keys/values/children intact, and the
key-search helper returns the correct slot for hits, misses, and the boundaries
(before-first, after-last, equal-key).

Implement `node.go` against the tests in `node_test.go`. Ping me when green or
stuck.
