package suite

import "testing"

type subtestContextSuite struct {
	SQLiteSuite
}

func TestSubtestContextSQLite(t *testing.T) {
	Run(t, &subtestContextSuite{})
}

// Ensures that the parent context is properly restored between subtests. This
// test reproduces an issue where using a transaction in the parent context
// between subtests could cause a panic.
func (s *subtestContextSuite) TestRestoresParentContextBetweenSubtests() {
	// First subtest should get its own child context and allow BeginTx.
	s.Run("first", func() {
		tx := s.BeginTx(nil)
		s.Require().NoError(tx.Rollback())
	})

	// This call runs in parent test scope between subtests; it previously panicked
	// when TearDownSubTest nulled out the suite context.
	tx := s.BeginTx(nil)
	s.Require().NoError(tx.Rollback())

	// Second subtest confirms we still derive a fresh child context afterward.
	s.Run("second", func() {
		tx := s.BeginTx(nil)
		s.Require().NoError(tx.Rollback())
	})
}
