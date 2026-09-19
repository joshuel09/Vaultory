-- Better Auth's schema (feature 004, issue #15).
--
-- Transcribed by hand from `npx @better-auth/cli generate` against Better Auth 1.7.5, rather than
-- applied by that tool. The constitution requires schema changes to ship as version-controlled
-- reversible migrations, and this project already has exactly one mechanism for that. Two tools
-- writing to one schema, each with its own idea of the current version, is how a database ends up
-- in a state neither can reconcile (research Decision 4).
--
-- The column names are camelCase and therefore quoted. That is Better Auth's convention, not
-- Vaultory's, and it is left visible rather than renamed: renaming through the library's `fields`
-- options must then be kept in step with it forever, and a mismatch fails at runtime instead of
-- here.

CREATE TABLE "user" (
    "id"            text        NOT NULL PRIMARY KEY,
    "name"          text        NOT NULL,
    "email"         text        NOT NULL UNIQUE,
    "emailVerified" boolean     NOT NULL,
    "image"         text,
    "createdAt"     timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt"     timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- The table the Go service reads on every request. "token" holds the value carried in the cookie,
-- stored as-is and not hashed, which is what lets another process perform the same lookup.
-- "createdAt" is what Go compares against the 90-day cap; nothing else enforces it.
CREATE TABLE "session" (
    "id"        text        NOT NULL PRIMARY KEY,
    "expiresAt" timestamptz NOT NULL,
    "token"     text        NOT NULL UNIQUE,
    "createdAt" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" timestamptz NOT NULL,
    "ipAddress" text,
    "userAgent" text,
    "userId"    text        NOT NULL REFERENCES "user" ("id") ON DELETE CASCADE
);

-- One row per authentication method. For email and password there is exactly one, with
-- "providerId" = 'credential' and the hash in "password". The Go service never reads this table.
CREATE TABLE "account" (
    "id"                    text        NOT NULL PRIMARY KEY,
    "accountId"             text        NOT NULL,
    "providerId"            text        NOT NULL,
    "userId"                text        NOT NULL REFERENCES "user" ("id") ON DELETE CASCADE,
    "accessToken"           text,
    "refreshToken"          text,
    "idToken"               text,
    "accessTokenExpiresAt"  timestamptz,
    "refreshTokenExpiresAt" timestamptz,
    "scope"                 text,
    "password"              text,
    "createdAt"             timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt"             timestamptz NOT NULL
);

-- Unused in this feature: email verification is out of scope. Created anyway so the schema matches
-- what the library expects, which is what makes the drift check in quickstart walkthrough I mean
-- something.
CREATE TABLE "verification" (
    "id"         text        NOT NULL PRIMARY KEY,
    "identifier" text        NOT NULL,
    "value"      text        NOT NULL,
    "expiresAt"  timestamptz NOT NULL,
    "createdAt"  timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt"  timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- FR-027's counters. In the database rather than in memory so the limit survives a restart and is
-- shared between instances; an in-memory counter would quietly reset on every deploy.
CREATE TABLE "rateLimit" (
    "id"          text    NOT NULL PRIMARY KEY,
    "key"         text    NOT NULL UNIQUE,
    "count"       integer NOT NULL,
    "lastRequest" bigint  NOT NULL
);

CREATE INDEX "session_userId_idx" ON "session" ("userId");
CREATE INDEX "account_userId_idx" ON "account" ("userId");
CREATE INDEX "verification_identifier_idx" ON "verification" ("identifier");
