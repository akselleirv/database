# Lesson 01 — Pager

## What it is

The pager is the lowest layer. It turns one file on disk into an array of
fixed-size **pages** (we use 4 KB) and hands them to the layers above as byte
slices. Everything else — B+ tree nodes, slotted record pages, the catalog —
is just *bytes laid out inside a page*. The pager doesn't know or care what's
in them.

Its whole job:

- Translate a `PageID` into a byte offset: `offset = pageID * pageSize`.
- Read a page from the file into memory.
- Write a page back to the file.
- Allocate new pages (grow the file).
- Tell you how many pages exist.

## Why it matters

Two reasons this layer exists at all:

1. **Uniform addressing.** Higher layers reference data by `PageID`, never by
   byte offset. A B+ tree child pointer is a `PageID`. This indirection is what
   lets the tree live on disk instead of in RAM.

2. **It's the durability boundary.** Every byte that survives a crash passes
   through here. A `write()` to a file does *not* mean the bytes are on the
   platter — the OS buffers them in the page cache. Only `fsync` forces them
   down. Get this wrong and your database silently loses committed data on power
   loss. We build the real crash-safety in the WAL lesson, but the pager is
   where `fsync` physically happens, so we confront it now.

## Decisions

These are decided for you. You don't need to deliberate — just implement them.
Each line names the tradeoff so you know *why*.

1. **Page 0 is the reserved meta page.** It holds a magic number + page size so
   `Open` can validate the file is ours (and later: root pointer, free-list
   head). Tradeoff: wastes most of a 4 KB page now, but gives a fixed, known
   location to bootstrap from — without it, you can't answer "where is the root?"

2. **`Open` errors on a non-multiple file length.** A length that isn't a whole
   number of pages means a wrong file or an unrecovered crash. Tradeoff: we
   refuse to "repair" by truncating — silently dropping a torn tail destroys data
   before we have a WAL to reason about it. Recovery is the WAL's job (Lesson 05),
   not the pager's.

3. **Reading an unallocated page (id >= PageCount) is an error.** Allocation is
   explicit via `AllocatePage`; reading a page you never allocated is a caller
   bug. Tradeoff: no "zero-fill on read" convenience, but reads and allocation
   stay cleanly separated and logic errors surface loudly instead of as a page of
   zeros that masquerades as a valid empty node.

4. **Reads copy into a caller-provided `dst`; the pager hands out nothing it
   owns.** Tradeoff: a copy per read, but when we add a page cache later, no
   caller can mutate-through to a shared cached page — that aliasing bug is
   designed out from the start.

5. **`WritePage` does NOT fsync; durability is the explicit `Sync()`.** `Close`
   calls `Sync`. Tradeoff: per-write fsync is correct but ~10–100× slower, and
   true crash-safety needs the WAL's write-ahead ordering anyway. The pager just
   exposes the `Sync()` primitive; the WAL decides when to call it.

6. **`AllocatePage` zero-fills the new page immediately** (extends the file by a
   zeroed page). Tradeoff: one extra 4 KB write per allocation, but it keeps the
   invariant *file length == PageCount × PageSize* true at all times — no phantom
   pages that are counted but not yet on disk.

## The contract

It's in `internal/pager/pager.go` as signatures with doc comments and `// TODO`
bodies. The doc comments state the decided behavior above. Standard library
only, idiomatic errors, no panics.

## Done when

`make test` is green. The tests in `pager_test.go` encode every decision above —
make them pass and you've implemented the contract correctly.

Start with the struct fields and `Open`; the rest follows from what state you
hold. When green, commit and tell me — I'll review.
