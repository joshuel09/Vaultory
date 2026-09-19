# Contracts: Collector Authentication

## The Go API contract does not change

`specs/001-add-browse-collectibles/contracts/openapi.yaml` describes four operations. None of their
paths, request bodies, or response shapes change in this feature. What changes is how the acting
collector is established behind them — and that was never in the contract, because the contract
does not describe it.

The one visible consequence: `401` stops being a state reachable only by forgetting a development
cookie and becomes the normal answer for "not signed in". The response shape is unchanged.

**Therefore there is no OpenAPI change in this feature, and no regeneration of frontend types.**
A task checks this rather than assuming it: the generated `frontend/lib/types/api.ts` must be
byte-identical after this feature to what it was before.

## The authentication endpoints are not in the contract, and should not be

Better Auth mounts its own routes under `/api/auth/*` in Next.js — sign-up, sign-in, sign-out,
session. They are not part of the Go API, they are not consumed by any other service, and their
shapes are owned by the library rather than by this project.

Constitution III requires the OpenAPI contract to be the source of truth **for the frontend/backend
seam**. These routes do not cross that seam: they are the frontend talking to itself. Writing them
into `openapi.yaml` would assert that Go serves them, which is false, and would need maintaining
against a library this project does not control.

## The real contract in this feature is the session

It is not expressible in OpenAPI, which is exactly why it is written down here. Two independent
programs must agree on it, and if they drift, requests fail in a way that looks like a database
problem rather than a contract problem.

### Cookie

```
Name:   better-auth.session_token        (default; "__Secure-" prefixed when served over HTTPS)
Value:  urlencode( <token> "." <signature> )

signature = base64( HMAC-SHA256( secret, <token> ) )     standard base64, with padding
```

Taken from `better-call/dist/crypto.cjs`:

```js
export const signCookieValue = async (value, secret) => {
  const signature = await makeSignature(value, secret);
  value = `${value}.${signature}`;
  return encodeURIComponent(value);
};
```

`makeSignature` ends in `btoa(...)`, so the signature is **standard** base64 — not base64url. Go
must use `base64.StdEncoding`. Feature 001's dev resolver used `base64.RawURLEncoding` for its own
cookie; copying that pattern here would fail every verification.

### Shared secret

Both sides read the same secret. Next.js as `BETTER_AUTH_SECRET`, Go as
`VAULTORY_SESSION_SECRET`, and they must hold the same value or nothing verifies. Compose passes
one variable to both services so they cannot disagree by accident.

### Verification, in order

1. URL-decode the cookie value.
2. Split on the **last** `.` — base64 padding can contain `=` but never `.`, and splitting on the
   first would break if a future token format contained one.
3. Recompute the HMAC over the token and compare in constant time. Reject on mismatch.
4. Look up the session by token, joining `user` and `collectors`.
5. Reject unless `expiresAt` is in the future and `createdAt` is within 90 days.

Steps 3 and 4 are both required. The signature alone proves the token was issued by something
holding the secret; the row proves it has not been revoked or expired. Dropping step 3 would let a
database-only attacker skip the secret; dropping step 4 would make sign-out take up to 30 days to
matter.

### What Go must never do

Accept a collector identifier from a header, a body field, or a query parameter — FR-013, and
Principle IV's "Client-provided ownership, user IDs, roles, permissions, prices, or trusted values
MUST NOT be accepted without verification". A contract test asserts that a request carrying a
forged identity and no valid session is refused.
