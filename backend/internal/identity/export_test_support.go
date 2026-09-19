package identity

// ExportedVerify exposes signature verification to the unit suite, which lives in a separate
// package. Verification is deliberately separable from the database lookup: it is pure, it is
// where the library-compatibility risk lives, and it can therefore be tested without PostgreSQL.
func ExportedVerify(s *SessionResolver, rawCookie string) (string, bool) {
	return s.verify(rawCookie)
}
