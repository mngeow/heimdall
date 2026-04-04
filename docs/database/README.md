# Database Design

This folder describes the SQLite database Symphony should use in v1.

## Goals

- persist polling cursors and work item snapshots
- prevent duplicate workflow execution
- track branches, OpenSpec changes, pull requests, and slash commands
- support retries, reconciliation, and auditability on a single Linux host

## Design Rules

- store runtime state in SQLite
- keep secrets out of the database
- keep repo routing rules and allowlists in config, not in database tables
- treat the database as the durable control plane for automation state

## Files In This Folder

- `schema.md`: table-by-table notes plus the Mermaid ERD
