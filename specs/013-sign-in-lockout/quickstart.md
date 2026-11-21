# Quickstart: verifying feature 013

## Without Docker

```bash
# backend: the functions, concurrency, and the auth role's privileges
cd backend && go test -tags=integration -run 'SignInLockout|AuthRole' ./tests/integration/
# frontend: the hook wiring and the key
cd frontend && npm test -- sign-in-lockout
```

## With Docker (browser suite)

1. `make up`, register and verify an account.
2. Sign in five times with a wrong password: each says the email or password is wrong.
3. The sixth, even with the right password: "Too many sign-in attempts…".
4. Repeat 2–3 with an address that has no account: the same responses.
5. Reset the password of the locked account, then sign in with the new one: it works at once.
