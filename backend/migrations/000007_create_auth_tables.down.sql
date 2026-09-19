-- Reverse dependency order: session and account reference user.
DROP TABLE IF EXISTS "rateLimit";
DROP TABLE IF EXISTS "verification";
DROP TABLE IF EXISTS "account";
DROP TABLE IF EXISTS "session";
DROP TABLE IF EXISTS "user";
