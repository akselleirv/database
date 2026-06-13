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

## Design questions this is really testing

Reason about these before you write code — they're the point of the exercise:

1. **Page 0.** The file starts empty. What lives at offset 0? Most engines
   reserve page 0 as a **header/meta page** (magic number, page size, page
   count, later: root pointers, free-list head). Do you reserve it now or pretend
   it's a normal page? What goes wrong later if you don't?

2. **Partial pages.** What if the file length isn't a multiple of 4 KB — a crash
   mid-write left a torn tail? Is that an error, do you truncate, or do you round
   the page count down? What does a half-written page even mean for correctness?

3. **Read past EOF.** Someone asks for page 5 but the file holds 3 pages. Bug in
   the caller, or a valid "give me a fresh zeroed page" request? Pick a contract
   and make the code enforce it.

4. **Who owns the returned bytes?** If a read hands back a `[]byte`, can the
   caller hold onto it and mutate it? Does that mutation reach disk? Is it shared
   with the next reader of the same page? You have no cache yet — but the
   *contract* you pick now constrains whether you can add one later. (This is why
   the skeleton reads into a caller-provided `dst` rather than returning a slice
   — think about why that's the safer default.)

5. **What does `fsync` cost, and when do you pay it?** Per page write? Per
   logical commit? Never (and let the WAL handle it)? There's no free answer;
   state your choice and why.

6. **Allocation.** Growing the file by one page and returning its ID — do you
   write zeros immediately, or just hand back the next ID and let the first write
   extend the file? What does the file look like on disk in each case, and what
   happens if you allocate then crash before writing?

## The contract

It's in `internal/pager/pager.go` as signatures with doc comments and `// TODO`
bodies. Read the doc comments — they pin down the edge-case behavior you have to
decide. Standard library only, idiomatic errors, no panics.

## Done when

- `Open` on a fresh path creates a file and establishes page 0 / the meta page.
- `Open` on an existing file validates its size and recovers the page count.
- Round-trip: write a page, read it back, get the same bytes — including across a
  `Close`/`Open` cycle.
- Reads and allocations respect whatever contract you chose for the edge cases
  above, and a test in `pager_test.go` demonstrates each choice.

Start with the struct fields and `Open`; the rest follows from what state you
decide to hold. When you've got something, tell me and I'll review.
