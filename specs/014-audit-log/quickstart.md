# Quickstart: verifying feature 014

```bash
cd backend && go test -tags=integration -run 'Audit|AuthRole' ./tests/integration/
cd frontend && npm test -- audit
```

Against the running stack: register, fail a sign-in, sign in, reset the password, sign out, add,
edit and delete a collectible; then `SELECT occurred_at, action, outcome, account_id, subject_id
FROM audit_events ORDER BY id` shows each, and nothing personal.
