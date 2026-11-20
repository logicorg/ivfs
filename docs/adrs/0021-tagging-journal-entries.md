# ADR 0021: Tagging journal entries

- Status: Proposed
- Date: 2026-11-20
- Depends on: [ADR 0003](0003-categories-and-tags.md) (tags, assignment
  deferred) and [ADR 0017](0017-editing-journal-entries.md) (replacing an entry)
- Amends: ADR 0003 (deleting an assigned tag)
- Related: [ADR 0018](0018-transactions-page.md) (the tag filter it reserves).
  Issue: logicorg/ivfs#35

## Decision

A journal entry has a set of tags, stored in `ledger.journal_entry_tag
(journal_entry_id, tag_id)`, keyed by the pair. Every transaction, investment,
and conversion is one entry, so tagging an entry tags the transaction.

- **Posting** takes `TagIDs` and writes them in the transaction that writes the
  entry. An unknown tag fails the post, and duplicates are dropped. A repeated
  idempotency key returns the entry with the tags it was first posted with.
- **Replacing** an entry (and so every `Update*` workflow) replaces its tags
  with the input's; none clears them.
- **`SetJournalEntryTags`** replaces the tags of an entry and nothing else, for
  a change of tags that should not restate the transaction. It is exposed as
  `PUT /api/journal-entries/{id}/tags`.
- **Deleting** an entry or a tag deletes their assignments (`ON DELETE
  CASCADE`). A tag is no longer archived once assigned: it is deleted, and the
  entries keep their other tags.
- There is no limit on the tags of an entry.

## Consequences

- The record endpoints take `tagIds`, entries are returned with their tags, and
  the record forms and the journal entry sheet pick them.
- The index on `tag_id` serves the cascade and the filter ADR 0018 reserves.
- MCP tools do not take tags yet.
