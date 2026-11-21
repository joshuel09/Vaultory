# Implementation Plan: An Audit Log of Account and Collection Changes

**Branch**: `100-audit-log` | **Date**: 2026-11-21 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/014-audit-log/spec.md`

## Summary

One append-only table, `audit_events`, written by both services. The Go service writes collection
events inside the same transaction as the change (FR-004). The frontend writes account events from
Better Auth's hooks and database hooks, and a failure to write is logged, never refused (FR-005).
PostgreSQL triggers make entries unchangeable and undeletable until a year old (FR-006); a purge
function removes expired entries in batches, called at the Go service's start-up and after each
deletion (FR-007). Stacked on #121 (feature 013): it reuses the address key and the sign-in hooks.

## Technical Context

**Language/Version**: Go 1.26 (collection events, purge); TypeScript, Better Auth 1.7.5 (account events); SQL (PostgreSQL 17)

**Primary Dependencies**: pgx (Go store), `pg` and Better Auth `hooks` / `databaseHooks` / `onPasswordReset` (frontend)

**Storage**: PostgreSQL — `audit_events`, two triggers, `audit_purge_expired`; migration 000014

**Testing**: Go integration (collection events, atomicity, append-only enforcement, privileges, purge); Vitest (which account event each hook records, nothing sensitive); the real server driven with curl, as feature 013 was

**Target Platform / Project Type**: the existing web application

**Performance Goals**: one insert per audited event; purge reads an index on `occurred_at`

**Constraints**: no readable email, network address, password, token or collectible content (FR-003)

**Scale/Scope**: a handful of rows per collector per day

## Constitution Check

| Principle | Assessment |
|---|---|
| **II. Architecture** | Collection events are written by Go, which owns those changes; account events by the frontend, which owns sign-in (feature 004). The rules — append-only, retention — are enforced by PostgreSQL, not by either service |
| **III. Contract-First** | No API change |
| **IV. Data Integrity and Security** | Change and entry in one transaction; no personal data stored; the frontend's role may insert and nothing else; reversible migration |
| **V. Simplicity** | One table, two triggers, one function. No reader UI |

**Gate**: passes.

## Project Structure

```text
backend/migrations/000014_create_audit_events.{up,down}.sql
backend/internal/store/postgres/audit.go                 (record in a tx; purge)
backend/internal/store/postgres/collectibles.go          (Add, Edit, Delete record their event)
backend/internal/collection/                              (purge after delete)
backend/cmd/vaultory-api/main.go                          (purge at start-up)
backend/tests/integration/audit_test.go
backend/tests/integration/auth_role_privileges_test.go
frontend/lib/audit.ts                                     (record an account event; never throws)
frontend/lib/auth.ts                                      (hooks, databaseHooks, onPasswordReset)
frontend/tests/unit/audit.test.ts
```

## Complexity Tracking

None.
