package fields_test

import (
	"database/sql"

	. "go.rtnl.ai/tidal/fields"
)

func (s *FieldsSqliteTestSuite) TestStringArray() {
	var (
		alpha StringArray = StringArray{"a", "b", "c"}
		bravo StringArray = StringArray{"b", "a", "d"}
	)

	s.Run("HappyPath", func() {
		require := s.Require()

		tx := s.BeginTx(nil)
		defer tx.Rollback()

		params := []sql.NamedArg{
			sql.Named("alpha", alpha),
			sql.Named("bravo", bravo),
		}

		result, err := tx.Exec("INSERT INTO testing (alpha, bravo) VALUES (:alpha, :bravo)", params...)
		require.NoError(err, "could not insert record")

		id, err := result.LastInsertId()
		require.NoError(err, "could not get last insert id")
		require.NotZero(id, "expected last insert id to be non-zero")

		row := tx.QueryRow("SELECT alpha, bravo FROM testing WHERE id=:id", sql.Named("id", id))
		var aOut, bOut StringArray
		require.NoError(row.Scan(&aOut, &bOut), "could not scan record")
		require.Equal(alpha, aOut, "expected alpha to be equal to the input")
		require.Equal(bravo, bOut, "expected bravo to be equal to the input")
	})

	s.Run("Null", func() {
		require := s.Require()

		tx := s.BeginTx(nil)
		defer tx.Rollback()

		result, err := tx.Exec("INSERT INTO testing (alpha, bravo) VALUES ('[]', NULL)")
		require.NoError(err, "could not insert record")

		id, err := result.LastInsertId()
		require.NoError(err, "could not get last insert id")
		require.NotZero(id, "expected last insert id to be non-zero")

		row := tx.QueryRow("SELECT alpha, bravo FROM testing WHERE id=:id", sql.Named("id", id))
		var aOut, bOut StringArray
		require.NoError(row.Scan(&aOut, &bOut), "could not scan record")
		require.Equal(StringArray{}, aOut, "expected alpha to be equal to the input")
		require.Equal(StringArray(nil), bOut, "expected bravo to be equal to the input")
	})

	s.Run("Nil", func() {
		require := s.Require()

		tx := s.BeginTx(nil)
		defer tx.Rollback()

		// Insert a new record into the database.
		params := []sql.NamedArg{
			sql.Named("alpha", `[]`),
			sql.Named("bravo", nil),
		}
		result, err := tx.Exec("INSERT INTO testing (alpha, bravo) VALUES (:alpha, :bravo)", params...)
		require.NoError(err, "could not insert record")

		id, err := result.LastInsertId()
		require.NoError(err, "could not get last insert id")

		// Fetch the record from the database.
		row := tx.QueryRow("SELECT alpha, bravo FROM testing WHERE id=:id", sql.Named("id", id))

		var aOut, bOut StringArray
		require.NoError(row.Scan(&aOut, &bOut), "could not scan record")
		require.Equal(StringArray{}, aOut, "expected alpha to be equal to the input")
		require.Equal(StringArray(nil), bOut, "expected bravo to be equal to the input")
	})
}

func (s *FieldsPostgresTestSuite) TestStringArray() {
	var (
		alpha   StringArray = StringArray{"a", "b", "c"}
		bravo   StringArray = StringArray{"b", "a", "d"}
		charlie StringArray = StringArray{"c", "d", "e"}
		delta   StringArray = StringArray{"e", "f", "g"}
	)

	s.Run("HappyPath", func() {
		require := s.Require()

		tx := s.BeginTx(nil)
		defer tx.Rollback()

		params := []sql.NamedArg{
			sql.Named("alpha", alpha),
			sql.Named("bravo", bravo),
			sql.Named("charlie", charlie),
			sql.Named("delta", delta),
		}

		row := tx.QueryRow("INSERT INTO testing (alpha, bravo, charlie, delta) VALUES (:alpha, :bravo, :charlie, :delta) RETURNING id", params...)

		var id int64
		require.NoError(row.Scan(&id), "could not scan record")
		require.NotZero(id, "expected last insert id to be non-zero")

		// Fetch the record from the database.
		row = tx.QueryRow("SELECT alpha, bravo, charlie, delta FROM testing WHERE id=:id", sql.Named("id", id))

		var aOut, bOut, cOut, dOut StringArray
		require.NoError(row.Scan(&aOut, &bOut, &cOut, &dOut), "could not scan record")
		require.Equal(alpha, aOut, "expected alpha to be equal to the input")
		require.Equal(bravo, bOut, "expected bravo to be equal to the input")
		require.Equal(charlie, cOut, "expected charlie to be equal to the input")
		require.Equal(delta, dOut, "expected delta to be equal to the input")
	})

	s.Run("Null", func() {
		require := s.Require()

		tx := s.BeginTx(nil)
		defer tx.Rollback()

		row := tx.QueryRow("INSERT INTO testing (alpha, bravo, charlie, delta) VALUES ('[]', NULL, '[]', NULL) RETURNING id")

		var id int64
		require.NoError(row.Scan(&id), "could not scan record")
		require.NotZero(id, "expected last insert id to be non-zero")

		// Fetch the record from the database.
		row = tx.QueryRow("SELECT alpha, bravo, charlie, delta FROM testing WHERE id=:id", sql.Named("id", id))

		var aOut, bOut, cOut, dOut StringArray
		require.NoError(row.Scan(&aOut, &bOut, &cOut, &dOut), "could not scan record")
		require.Equal(StringArray{}, aOut, "expected alpha to be equal to the input")
		require.Equal(StringArray(nil), bOut, "expected bravo to be equal to the input")
		require.Equal(StringArray{}, cOut, "expected charlie to be equal to the input")
		require.Equal(StringArray(nil), dOut, "expected delta to be equal to the input")
	})

	s.Run("Nil", func() {
		require := s.Require()

		tx := s.BeginTx(nil)
		defer tx.Rollback()

		params := []sql.NamedArg{
			sql.Named("alpha", `[]`),
			sql.Named("bravo", nil),
			sql.Named("charlie", `[]`),
			sql.Named("delta", nil),
		}
		row := tx.QueryRow("INSERT INTO testing (alpha, bravo, charlie, delta) VALUES (:alpha, :bravo, :charlie, :delta) RETURNING id", params...)

		var id int64
		require.NoError(row.Scan(&id), "could not scan record")
		require.NotZero(id, "expected last insert id to be non-zero")

		// Fetch the record from the database.
		row = tx.QueryRow("SELECT alpha, bravo, charlie, delta FROM testing WHERE id=:id", sql.Named("id", id))

		var aOut, bOut, cOut, dOut StringArray
		require.NoError(row.Scan(&aOut, &bOut, &cOut, &dOut), "could not scan record")
		require.Equal(StringArray{}, aOut, "expected alpha to be equal to the input")
		require.Equal(StringArray(nil), bOut, "expected bravo to be equal to the input")
		require.Equal(StringArray{}, cOut, "expected charlie to be equal to the input")
		require.Equal(StringArray(nil), dOut, "expected delta to be equal to the input")
	})
}

func (s *FieldsSqliteTestSuite) TestNullStringArray() {
	var (
		alpha NullStringArray = NullStringArray{StringArray: StringArray{"a", "b", "c"}, Valid: true}
		bravo NullStringArray = NullStringArray{StringArray: StringArray{"b", "a", "d"}, Valid: true}
	)

	s.Run("HappyPath", func() {
		require := s.Require()

		tx := s.BeginTx(nil)
		defer tx.Rollback()

		params := []sql.NamedArg{
			sql.Named("alpha", alpha),
			sql.Named("bravo", bravo),
		}

		result, err := tx.Exec("INSERT INTO testing (alpha, bravo) VALUES (:alpha, :bravo)", params...)
		require.NoError(err, "could not insert record")

		id, err := result.LastInsertId()
		require.NoError(err, "could not get last insert id")
		require.NotZero(id, "expected last insert id to be non-zero")

		row := tx.QueryRow("SELECT alpha, bravo FROM testing WHERE id=:id", sql.Named("id", id))
		var aOut, bOut StringArray
		require.NoError(row.Scan(&aOut, &bOut), "could not scan record")
		require.Equal(alpha.StringArray, aOut, "expected alpha to be equal to the input")
		require.Equal(bravo.StringArray, bOut, "expected bravo to be equal to the input")
	})

	s.Run("Null", func() {
		require := s.Require()

		tx := s.BeginTx(nil)
		defer tx.Rollback()

		result, err := tx.Exec("INSERT INTO testing (alpha, bravo) VALUES ('null', NULL)")
		require.NoError(err, "could not insert record")

		id, err := result.LastInsertId()
		require.NoError(err, "could not get last insert id")
		require.NotZero(id, "expected last insert id to be non-zero")

		row := tx.QueryRow("SELECT alpha, bravo FROM testing WHERE id=:id", sql.Named("id", id))
		var aOut, bOut NullStringArray
		require.NoError(row.Scan(&aOut, &bOut), "could not scan record")
		require.False(aOut.Valid, "expected alpha to be invalid")
		require.False(bOut.Valid, "expected bravo to be invalid")
		require.Equal(StringArray(nil), aOut.StringArray, "expected alpha to be equal to the input")
		require.Equal(StringArray(nil), bOut.StringArray, "expected bravo to be equal to the input")
	})
}

func (s *FieldsPostgresTestSuite) TestNullStringArray() {
	var (
		alpha   NullStringArray = NullStringArray{StringArray: StringArray{"a", "b", "c"}, Valid: true}
		bravo   NullStringArray = NullStringArray{StringArray: StringArray{"b", "a", "d"}, Valid: true}
		charlie NullStringArray = NullStringArray{StringArray: StringArray{"c", "d", "e"}, Valid: true}
		delta   NullStringArray = NullStringArray{StringArray: StringArray{"e", "f", "g"}, Valid: true}
	)

	s.Run("HappyPath", func() {
		require := s.Require()

		tx := s.BeginTx(nil)
		defer tx.Rollback()

		params := []sql.NamedArg{
			sql.Named("alpha", alpha),
			sql.Named("bravo", bravo),
			sql.Named("charlie", charlie),
			sql.Named("delta", delta),
		}
		row := tx.QueryRow("INSERT INTO testing (alpha, bravo, charlie, delta) VALUES (:alpha, :bravo, :charlie, :delta) RETURNING id", params...)

		var id int64
		require.NoError(row.Scan(&id), "could not scan record")
		require.NotZero(id, "expected last insert id to be non-zero")

		// Fetch the record from the database.
		row = tx.QueryRow("SELECT alpha, bravo, charlie, delta FROM testing WHERE id=:id", sql.Named("id", id))

		var aOut, bOut, cOut, dOut NullStringArray
		require.NoError(row.Scan(&aOut, &bOut, &cOut, &dOut), "could not scan record")
		require.True(aOut.Valid, "expected alpha to be valid")
		require.True(bOut.Valid, "expected bravo to be valid")
		require.True(cOut.Valid, "expected charlie to be valid")
		require.True(dOut.Valid, "expected delta to be valid")
		require.Equal(alpha.StringArray, aOut.StringArray, "expected alpha to be equal to the input")
		require.Equal(bravo.StringArray, bOut.StringArray, "expected bravo to be equal to the input")
		require.Equal(charlie.StringArray, cOut.StringArray, "expected charlie to be equal to the input")
		require.Equal(delta.StringArray, dOut.StringArray, "expected delta to be equal to the input")
	})

	s.Run("Null", func() {
		require := s.Require()

		tx := s.BeginTx(nil)
		defer tx.Rollback()

		params := []sql.NamedArg{
			sql.Named("bravo", NullStringArray{Valid: false}),
			sql.Named("delta", NullStringArray{Valid: false}),
		}

		row := tx.QueryRow("INSERT INTO testing (alpha, bravo, charlie, delta) VALUES ('[]', :bravo, '[]', :delta) RETURNING id", params...)

		var id int64
		require.NoError(row.Scan(&id), "could not scan record")
		require.NotZero(id, "expected last insert id to be non-zero")

		row = tx.QueryRow("SELECT alpha, bravo, charlie, delta FROM testing WHERE id=:id", sql.Named("id", id))

		var aOut, bOut, cOut, dOut NullStringArray
		require.NoError(row.Scan(&aOut, &bOut, &cOut, &dOut), "could not scan record")
		require.False(aOut.Valid, "expected alpha to be invalid")
		require.False(bOut.Valid, "expected bravo to be invalid")
		require.False(cOut.Valid, "expected charlie to be invalid")
		require.False(dOut.Valid, "expected delta to be invalid")
		require.Equal(StringArray{}, aOut.StringArray, "expected alpha to be equal to the input")
		require.Equal(StringArray(nil), bOut.StringArray, "expected bravo to be equal to the input")
		require.Equal(StringArray{}, cOut.StringArray, "expected charlie to be equal to the input")
		require.Equal(StringArray(nil), dOut.StringArray, "expected delta to be equal to the input")
	})
}
