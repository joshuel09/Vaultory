# Implementation Plan: Lock Sign-In per Email Address After Repeated Failures

**Branch**: `99-account-sign-in-lockout` | **Date**: 2026-11-20 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/013-sign-in-lockout/spec.md`

## Summary

Five consecutive failed sign-ins for one email address lock sign-in for that address for 15
minutes, whether or not an account exists. The rule lives in PostgreSQL as three functions —
record a failure, ask whether an address is locked, clear an address — over one table keyed by an
HMAC of the normalized address. Better Auth's `hooks.before` refuses a locked address with 429 and
one generic message; `hooks.after` records a 401 as a failure and clears on success;
`emailAndPassword.onPasswordReset` clears on a completed reset. The per-network ceiling stays.

## Technical Context

**Language/Version**: TypeScript (Next.js 15, Node 22) for the hook wiring; SQL (PostgreSQL 17) for the rule; Go 1.26 for the integration tests that exercise it

**Primary Dependencies**: Better Auth 1.7.5 (`hooks`, `APIError`, `onPasswordReset`), `pg`, `node:crypto` (HMAC)

**Storage**: PostgreSQL — table `sign_in_lockouts`, three functions; migration 000013

**Testing**: Go integration suite (functions, concurrency, the auth role's privileges); Vitest (hook wiring, key normalization); Playwright (end to end; runs in the browser CI job)

**Target Platform**: The existing frontend and database services

**Project Type**: Web application (frontend + backend + database)

**Performance Goals**: One indexed row read before each sign-in, one upsert or delete after; no measurable change to sign-in time

**Constraints**: Identical client-visible behaviour for unknown and known addresses (FR-006); no readable address stored (FR-009); shared across restarts and instances (FR-008)

**Scale/Scope**: One row per address that has failed since its last success or lock expiry

## Constitution Check

| Principle | Assessment |
|---|---|
| **I. Collector-First** | A locked collector is told plainly what happened and how to get back in (FR-007, password reset) |
| **II. Architecture** | Better Auth already owns sign-in (feature 004). The rule is not left in the frontend: it lives in PostgreSQL functions, the authoritative store, and the frontend only calls them. The Go service is untouched, as it is for every sign-in rule |
| **III. Contract-First** | The sign-in response gains one status, 429 with a fixed message; recorded in `contracts/sign-in.md`. No change to the Go API's OpenAPI contract |
| **IV. Data Integrity and Security** | No readable address stored (HMAC with the server secret); the migration is reversible; the restricted auth role gains exactly this table and these functions, asserted by the existing privilege test |
| **V. Quality, Simplicity** | One table, three functions, two hooks. Concurrency tested, not argued (SC-005) |

**Gate**: passes. Post-design re-check: passes.

## Project Structure

### Documentation (this feature)

```text
specs/013-sign-in-lockout/
├── spec.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/sign-in.md
├── checklists/requirements.md
└── tasks.md
```

### Source Code (repository root)

```text
backend/migrations/000013_create_sign_in_lockouts.{up,down}.sql
backend/tests/integration/sign_in_lockout_test.go
backend/tests/integration/auth_role_privileges_test.go   (the new table and functions are permitted)
frontend/lib/sign-in-lockout.ts                          (key, and calls to the three functions)
frontend/lib/auth.ts                                     (hooks, onPasswordReset, corrected comment)
frontend/tests/unit/sign-in-lockout.test.ts
frontend/tests/e2e/sign-in-lockout.spec.ts
specs/004-collector-authentication/spec.md               (FR-027 amended)
```

**Structure Decision**: The rule goes where every instance of the frontend shares it and where the
backend's test suite can reach it: the database. The frontend gains one small module so `auth.ts`
stays a configuration file.

## Complexity Tracking

None.
