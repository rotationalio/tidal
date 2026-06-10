package fields_test

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
	. "go.rtnl.ai/tidal/fields"
)

var (
	alphaMap   = map[string]int{"a": 1, "b": 2}
	bravoMap   = map[string]int{"b": 2, "a": 1}
	charlieMap = map[string]int{"c": 3, "d": 4}
	deltaMap   = map[string]int{"e": 5, "f": 6}

	alphaBytes   = []byte(`{"a":1,"b":2}`)
	bravoBytes   = []byte(`{"b":2,"a":1}`)
	charlieBytes = []byte(`{"c":3,"d":4}`)
	deltaBytes   = []byte(`{"e":5,"f":6}`)
)

func TestJSONB_Normalize(t *testing.T) {
	t.Run("StableKeyOrder", func(t *testing.T) {
		alpha := []byte(`{"a":1,"b":2}`)
		bravo := []byte(`{"b":2,"a":1}`)
		require.Equal(t, JSONB(alpha).Normalize(), JSONB(bravo).Normalize(), "expected equal canonical bytes:\n%q\n%q", alpha, bravo)
	})

	t.Run("Empty", func(t *testing.T) {
		require.Nil(t, JSONB(nil).Normalize(), "expected nil input to return nil")
		require.Nil(t, JSONB([]byte{}).Normalize(), "expected empty input to return nil")
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		invalid := []byte(`{"a":1,"b":2,}`)
		require.Equal(t, invalid, JSONB(invalid).Normalize(), "expected invalid input to return unchanged bytes:\n%q", invalid)
	})
}

func (s *FieldsSqliteTestSuite) TestJSONB() {
	s.Run("HappyPath", func() {
		require := s.Require()
		tx := s.BeginTx(nil)
		defer tx.Rollback()

		// Insert a new record into the database.
		params := []any{
			sql.Named("alpha", JSONB(alphaBytes)),
			sql.Named("bravo", JSONB(bravoBytes)),
		}
		result, err := tx.Exec("INSERT INTO testing (alpha, bravo) VALUES (:alpha, :bravo)", params...)
		require.NoError(err, "could not insert record")

		id, err := result.LastInsertId()
		require.NoError(err, "could not get last insert id")
		require.NotZero(id, "expected last insert id to be non-zero")

		// Fetch the record from the database.
		row := tx.QueryRow("SELECT alpha, bravo FROM testing WHERE id=:id", sql.Named("id", id))

		var alpha, bravo JSONB
		require.NoError(row.Scan(&alpha, &bravo), "could not scan record")
		require.Equal(JSONB(alphaBytes), alpha, "expected alpha to be equal to the input")
		require.Equal(JSONB(bravoBytes), bravo, "expected bravo to be equal to the input")
	})

	s.Run("Null", func() {
		require := s.Require()
		tx := s.BeginTx(nil)
		defer tx.Rollback()

		// Insert a new record into the database.
		result, err := tx.Exec("INSERT INTO testing (alpha, bravo) VALUES ('null', NULL)")
		require.NoError(err, "could not insert record")

		id, err := result.LastInsertId()
		require.NoError(err, "could not get last insert id")
		require.NotZero(id, "expected last insert id to be non-zero")

		// Fetch the record from the database.
		row := tx.QueryRow("SELECT alpha, bravo FROM testing WHERE id=:id", sql.Named("id", id))

		var alpha, bravo JSONB
		require.NoError(row.Scan(&alpha, &bravo), "could not scan record")
		require.Equal(JSONB(nil), alpha, "expected alpha to be equal to the input")
		require.Equal(JSONB(nil), bravo, "expected bravo to be equal to the input")
	})

	s.Run("Nil", func() {
		require := s.Require()
		tx := s.BeginTx(nil)
		defer tx.Rollback()

		// Insert a new record into the database.
		params := []any{
			sql.Named("alpha", `{}`),
			sql.Named("bravo", nil),
		}
		result, err := tx.Exec("INSERT INTO testing (alpha, bravo) VALUES (:alpha, :bravo)", params...)
		require.NoError(err, "could not insert record")

		id, err := result.LastInsertId()
		require.NoError(err, "could not get last insert id")
		require.NotZero(id, "expected last insert id to be non-zero")

		// Fetch the record from the database.
		row := tx.QueryRow("SELECT alpha, bravo FROM testing WHERE id=:id", sql.Named("id", id))

		var alpha, bravo JSONB
		require.NoError(row.Scan(&alpha, &bravo), "could not scan record")
		require.Equal(JSONB([]byte(`{}`)), alpha, "expected alpha to be equal to the input")
		require.Equal(JSONB(nil), bravo, "expected bravo to be equal to the input")
	})
}

func (s *FieldsPostgresTestSuite) TestJSONB() {
	s.Run("HappyPath", func() {
		require := s.Require()
		tx := s.BeginTx(nil)
		defer tx.Rollback()

		params := []any{
			sql.Named("alpha", JSONB(alphaBytes)),
			sql.Named("bravo", JSONB(bravoBytes)),
			sql.Named("charlie", JSONB(charlieBytes)),
			sql.Named("delta", JSONB(deltaBytes)),
		}

		ins := tx.QueryRow("INSERT INTO testing (alpha, bravo, charlie, delta) VALUES ($1, $2, $3, $4) RETURNING id", params...)

		var id int64
		require.NoError(ins.Scan(&id), "could not insert record or scan ID")
		require.NotZero(id, "expected last insert id to be non-zero")

		// Fetch the record from the database.
		row := tx.QueryRow("SELECT alpha, bravo, charlie, delta FROM testing WHERE id=$1", id)

		var alpha, bravo, charlie, delta JSONB
		require.NoError(row.Scan(&alpha, &bravo, &charlie, &delta), "could not scan record")
		require.JSONEq(string(alphaBytes), string(alpha), "expected alpha to be equal to the input")
		require.JSONEq(string(bravoBytes), string(bravo), "expected bravo to be equal to the input")
		require.JSONEq(string(charlieBytes), string(charlie), "expected charlie to be equal to the input")
		require.JSONEq(string(deltaBytes), string(delta), "expected delta to be equal to the input")
	})

	s.Run("Null", func() {
		require := s.Require()
		tx := s.BeginTx(nil)
		defer tx.Rollback()

		// Insert a new record into the database.
		ins := tx.QueryRow("INSERT INTO testing (alpha, bravo, charlie, delta) VALUES ('null', NULL, 'null', NULL) RETURNING id")

		var id int64
		require.NoError(ins.Scan(&id), "could not scan record")
		require.NotZero(id, "expected last insert id to be non-zero")

		// Fetch the record from the database.
		row := tx.QueryRow("SELECT alpha, bravo, charlie, delta FROM testing WHERE id=$1", id)

		var alpha, bravo, charlie, delta JSONB
		require.NoError(row.Scan(&alpha, &bravo, &charlie, &delta), "could not scan record")
		require.Equal(JSONB(nil), alpha, "expected alpha to be equal to the input")
		require.Equal(JSONB(nil), bravo, "expected bravo to be equal to the input")
		require.Equal(JSONB(nil), charlie, "expected charlie to be equal to the input")
		require.Equal(JSONB(nil), delta, "expected delta to be equal to the input")
	})

	s.Run("Nil", func() {
		require := s.Require()
		tx := s.BeginTx(nil)
		defer tx.Rollback()

		params := []any{
			sql.Named("bravo", nil),
			sql.Named("delta", nil),
		}
		ins := tx.QueryRow("INSERT INTO testing (alpha, bravo, charlie, delta) VALUES ('{}', $1, '{}', $2) RETURNING id", params...)

		var id int64
		require.NoError(ins.Scan(&id), "could not scan record")
		require.NotZero(id, "expected last insert id to be non-zero")

		// Fetch the record from the database.
		row := tx.QueryRow("SELECT alpha, bravo, charlie, delta FROM testing WHERE id=$1", id)

		var alpha, bravo, charlie, delta JSONB
		require.NoError(row.Scan(&alpha, &bravo, &charlie, &delta), "could not scan record")

		require.Equal(JSONB([]byte(`{}`)), alpha, "expected alpha to be equal to the input")
		require.Equal(JSONB(nil), bravo, "expected bravo to be equal to the input")
		require.Equal(JSONB([]byte(`{}`)), charlie, "expected charlie to be equal to the input")
		require.Equal(JSONB(nil), delta, "expected delta to be equal to the input")
	})
}

func TestJSONB_IsNull(t *testing.T) {
	t.Run("Null", func(t *testing.T) {
		require.True(t, JSONB(nil).IsNull(), "expected nil input to be null")
		require.True(t, JSONB([]byte{}).IsNull(), "expected empty input to be null")
		require.True(t, JSONB([]byte("null")).IsNull(), "expected null string input to be null")
	})

	t.Run("Not Null", func(t *testing.T) {
		require.False(t, JSONB([]byte(`{"a":1}`)).IsNull(), "expected non-null input to be not null")
		require.False(t, JSONB([]byte(`""`)).IsNull(), "expected non-null string input to be not null")
		require.False(t, JSONB([]byte(`"foo"`)).IsNull(), "expected non-null string input to be not null")
		require.False(t, JSONB([]byte(`1`)).IsNull(), "expected non-null number input to be not null")
		require.False(t, JSONB([]byte(`true`)).IsNull(), "expected non-null boolean input to be not null")
		require.False(t, JSONB([]byte(`false`)).IsNull(), "expected non-null boolean input to be not null")
		require.False(t, JSONB([]byte(`[1,2,3]`)).IsNull(), "expected non-null array input to be not null")
	})
}

func TestJSONB_UnmarshalTo(t *testing.T) {
	t.Run("Nil", func(t *testing.T) {
		var v map[string]any
		require.NoError(t, JSONB(nil).UnmarshalTo(&v), "expected unmarshal to succeed")
		require.Nil(t, v, "expected unmarshaled value to be empty")
	})

	t.Run("Empty", func(t *testing.T) {
		var v map[string]int
		require.NoError(t, JSONB([]byte{}).UnmarshalTo(&v), "expected unmarshal to succeed")
		require.Nil(t, v, "expected unmarshaled value to be empty")
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		invalid := []byte(`{"a":1,"b":2,}`)
		var v map[string]int
		require.Error(t, JSONB(invalid).UnmarshalTo(&v), "expected unmarshal to fail")
		require.Nil(t, v, "expected unmarshaled value to be empty")
	})

	t.Run("ValidJSON", func(t *testing.T) {
		var v map[string]int
		require.NoError(t, JSONB(alphaBytes).UnmarshalTo(&v), "expected unmarshal to succeed")
		require.Equal(t, alphaMap, v, "expected unmarshaled value to be equal to the input")
	})
}

func TestJSONB_MarshalFrom(t *testing.T) {
	t.Run("Nil", func(t *testing.T) {
		b := JSONB{}
		require.NoError(t, b.MarshalFrom(nil), "could not marshal from nil")
		require.Nil(t, b, "expected the field to be nil")
	})

	t.Run("Valid", func(t *testing.T) {
		b := JSONB{}
		require.NoError(t, b.MarshalFrom(bravoMap), "could not marshal from valid value")
		require.JSONEq(t, string(bravoBytes), string(b), "expected the field to be equal to the input")
	})

	t.Run("Invalid", func(t *testing.T) {
		b := JSONB{}
		require.Error(t, b.MarshalFrom(func() {}), "expected marshal from to fail")
		require.Empty(t, b, "expected the field to be empty")
	})
}

//============================================================================
// NullJSONB Tests
//============================================================================

func (s *FieldsSqliteTestSuite) TestNullJSONB() {
	s.Run("HappyPath", func() {
		require := s.Require()
		tx := s.BeginTx(nil)
		defer tx.Rollback()

		params := []any{
			sql.Named("alpha", NullJSONB{JSONB: JSONB(alphaBytes), Valid: true}),
			sql.Named("bravo", NullJSONB{JSONB: JSONB(bravoBytes), Valid: true}),
		}
		result, err := tx.Exec("INSERT INTO testing (alpha, bravo) VALUES (:alpha, :bravo)", params...)
		require.NoError(err, "could not insert record")

		id, err := result.LastInsertId()
		require.NoError(err, "could not get last insert id")
		require.NotZero(id, "expected last insert id to be non-zero")

		// Fetch the record from the database.
		row := tx.QueryRow("SELECT alpha, bravo FROM testing WHERE id=:id", sql.Named("id", id))

		var alpha, bravo NullJSONB
		require.NoError(row.Scan(&alpha, &bravo), "could not scan record")
		require.Equal(NullJSONB{JSONB: JSONB(alphaBytes), Valid: true}, alpha, "expected alpha to be equal to the input")
		require.Equal(NullJSONB{JSONB: JSONB(bravoBytes), Valid: true}, bravo, "expected bravo to be equal to the input")
	})

	s.Run("Null", func() {
		require := s.Require()
		tx := s.BeginTx(nil)
		defer tx.Rollback()

		// Insert a new record into the database.
		result, err := tx.Exec("INSERT INTO testing (alpha, bravo) VALUES ('null', NULL)")
		require.NoError(err, "could not insert record")

		id, err := result.LastInsertId()
		require.NoError(err, "could not get last insert id")
		require.NotZero(id, "expected last insert id to be non-zero")

		// Fetch the record from the database.
		row := tx.QueryRow("SELECT alpha, bravo FROM testing WHERE id=:id", sql.Named("id", id))

		var alpha, bravo NullJSONB
		require.NoError(row.Scan(&alpha, &bravo), "could not scan record")
		require.Equal(NullJSONB{JSONB: JSONB(nil), Valid: false}, alpha, "expected alpha to be equal to the input")
		require.Equal(NullJSONB{JSONB: JSONB(nil), Valid: false}, bravo, "expected bravo to be equal to the input")
	})
}

func (s *FieldsPostgresTestSuite) TestNullJSONB() {
	s.Run("HappyPath", func() {
		require := s.Require()
		tx := s.BeginTx(nil)
		defer tx.Rollback()

		params := []any{
			sql.Named("alpha", NullJSONB{JSONB: JSONB(alphaBytes), Valid: true}),
			sql.Named("bravo", NullJSONB{JSONB: JSONB(bravoBytes), Valid: true}),
			sql.Named("charlie", NullJSONB{JSONB: JSONB(charlieBytes), Valid: true}),
			sql.Named("delta", NullJSONB{JSONB: JSONB(deltaBytes), Valid: true}),
		}
		ins := tx.QueryRow("INSERT INTO testing (alpha, bravo, charlie, delta) VALUES ($1, $2, $3, $4) RETURNING id", params...)

		var id int64
		require.NoError(ins.Scan(&id), "could not insert record or scan ID")
		require.NotZero(id, "expected last insert id to be non-zero")

		// Fetch the record from the database.
		row := tx.QueryRow("SELECT alpha, bravo, charlie, delta FROM testing WHERE id=$1", id)

		var alpha, bravo, charlie, delta NullJSONB
		require.NoError(row.Scan(&alpha, &bravo, &charlie, &delta), "could not scan record")
		require.True(alpha.Valid, "expected alpha to be valid")
		require.True(bravo.Valid, "expected bravo to be valid")
		require.True(charlie.Valid, "expected charlie to be valid")
		require.True(delta.Valid, "expected delta to be valid")

		require.JSONEq(string(alphaBytes), string(alpha.JSONB), "expected alpha to be equal to the input")
		require.JSONEq(string(bravoBytes), string(bravo.JSONB), "expected bravo to be equal to the input")
		require.JSONEq(string(charlieBytes), string(charlie.JSONB), "expected charlie to be equal to the input")
		require.JSONEq(string(deltaBytes), string(delta.JSONB), "expected delta to be equal to the input")
	})

	s.Run("Null", func() {
		require := s.Require()
		tx := s.BeginTx(nil)
		defer tx.Rollback()

		// Insert a new record into the database.
		ins := tx.QueryRow("INSERT INTO testing (alpha, bravo, charlie, delta) VALUES ('null', NULL, 'null', NULL) RETURNING id")

		var id int64
		require.NoError(ins.Scan(&id), "could not scan record")
		require.NotZero(id, "expected last insert id to be non-zero")

		// Fetch the record from the database.
		row := tx.QueryRow("SELECT alpha, bravo, charlie, delta FROM testing WHERE id=$1", id)

		var alpha, bravo, charlie, delta NullJSONB
		require.NoError(row.Scan(&alpha, &bravo, &charlie, &delta), "could not scan record")
		require.Equal(NullJSONB{JSONB: JSONB(nil), Valid: false}, alpha, "expected alpha to be equal to the input")
		require.Equal(NullJSONB{JSONB: JSONB(nil), Valid: false}, bravo, "expected bravo to be equal to the input")
		require.Equal(NullJSONB{JSONB: JSONB(nil), Valid: false}, charlie, "expected charlie to be equal to the input")
		require.Equal(NullJSONB{JSONB: JSONB(nil), Valid: false}, delta, "expected delta to be equal to the input")
	})
}

func TestNullJSONB_UnmarshalTo(t *testing.T) {
	t.Run("Nil", func(t *testing.T) {
		var v map[string]int
		require.NoError(t, NullJSONB{}.UnmarshalTo(&v), "expected unmarshal to succeed")
		require.Nil(t, v, "expected unmarshaled value to be empty")
	})

	t.Run("Null", func(t *testing.T) {
		b := JSONB(charlieBytes)
		var v map[string]int
		require.NoError(t, NullJSONB{JSONB: b}.UnmarshalTo(&v), "expected no error")
		require.Nil(t, v, "expected unmarshaled value to be empty")
	})

	t.Run("NotNull", func(t *testing.T) {
		b := JSONB(deltaBytes)
		var v map[string]int
		require.NoError(t, NullJSONB{JSONB: b, Valid: true}.UnmarshalTo(&v), "expected no error")
		require.Equal(t, deltaMap, v, "expected unmarshaled value to be equal to the input")
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		invalid := []byte(`{"a":1,"b":2,}`)
		var v map[string]int
		require.Error(t, NullJSONB{JSONB: JSONB(invalid), Valid: true}.UnmarshalTo(&v), "expected unmarshal to fail")
		require.Nil(t, v, "expected unmarshaled value to be empty")
	})
}

func TestNullJSONB_MarshalFrom(t *testing.T) {
	t.Run("Nil", func(t *testing.T) {
		b := NullJSONB{}
		require.NoError(t, b.MarshalFrom(nil), "could not marshal from nil")
		require.False(t, b.Valid, "expected the field to be invalid")
		require.Empty(t, b.JSONB, "expected the field to be empty")
	})

	t.Run("Null", func(t *testing.T) {
		b := NullJSONB{}
		require.NoError(t, b.MarshalFrom(&NullJSON{}), "could not marshal from valid value")
		require.False(t, b.Valid, "expected the field to be valid")
		require.Nil(t, b.JSONB, "expected the field to be nil")
	})

	t.Run("Error", func(t *testing.T) {
		b := NullJSONB{}
		require.Error(t, b.MarshalFrom(func() {}), "expected marshal from to fail")
		require.False(t, b.Valid, "expected the field to be invalid")
		require.Empty(t, b.JSONB, "expected the field to be empty")
	})

	t.Run("Valid", func(t *testing.T) {
		b := NullJSONB{}
		require.NoError(t, b.MarshalFrom(charlieMap), "could not marshal from valid value")
		require.True(t, b.Valid, "expected the field to be valid")
		require.JSONEq(t, string(charlieBytes), string(b.JSONB), "expected the field to be equal to the input")
	})

}

type NullJSON struct{}

func (n NullJSON) MarshalJSON() ([]byte, error) {
	return []byte("null"), nil
}
