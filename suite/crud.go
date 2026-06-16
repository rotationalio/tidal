package suite

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"testing"
	"time"

	"go.rtnl.ai/tidal"
	"go.rtnl.ai/tidal/fields"
	"go.rtnl.ai/ulid"
	"go.rtnl.ai/x/typecase"
)

// Configuration for a model conformance run against [tidal.CRUD].
//
// The caller supplies a table name, a factory for fresh models, and an update
// mutator. Everything else is handled by the suite: shape checks, an isolated
// Scan check (no database), and a full CRUD round-trip against the real database
// connection already set up on [DatabaseSuite] (rolled back at the end).
type CRUDConformance[M tidal.Model] struct {
	// Database table the model maps to.
	Table string

	// Returns a fresh model suitable for insert. Values that must be unique across
	// runs (such as email) should be generated here.
	Create func() M

	// Mutates a model in place to exercise the update path during round-trip.
	// The same instance that was created and inserted is passed here.
	Update func(M)

	// Optional comparison for retrieved and scanned models. When nil, a default
	// comparison is used that tolerates database timestamp truncation and compares
	// ULIDs by value. Provide a custom function when models include field types that
	// need special handling (for example JSONB normalization).
	Equal func(a, b M) bool

	// Phases selects which conformance phases to run. When empty, all phases run.
	Phases []CRUDPhase
}

// CRUDPhase names a conformance phase run by [ConformsCRUD].
type CRUDPhase string

const (
	CRUDShape     CRUDPhase = "Shape"
	CRUDScan      CRUDPhase = "Scan"
	CRUDRoundTrip CRUDPhase = "RoundTrip"
)

// Runs three independent conformance phases against a [tidal.Model] implementation.
//
//   - Shape: static checks on Fields/Params/SQL only; no database, no CRUD calls.
//   - Scan: exercises Scan with a fake row built from Params; no database, no CRUD calls.
//   - RoundTrip: exercises real [tidal.CRUD] methods against the suite database inside
//     a transaction that is always rolled back; nothing is committed.
//
// This does not mock tidal, patch the model, or substitute a fake database for
// round-trip. The only "fake" piece is mockScanner, used exclusively in the Scan phase.
func ConformsCRUD[M tidal.Model](s *DatabaseSuite, cfg CRUDConformance[M]) {
	s.T().Helper()
	require := s.Require()
	require.NotEmpty(cfg.Table, "table is required")
	require.NotNil(cfg.Create, "create is required")
	require.NotNil(cfg.Update, "update is required")

	crud := tidal.New[M](cfg.Table)

	if len(cfg.Phases) == 0 {
		cfg.Phases = []CRUDPhase{CRUDShape, CRUDScan, CRUDRoundTrip}
	}

	for _, phase := range cfg.Phases {
		switch phase {
		case CRUDShape:
			s.Run("Shape", func() {
				testCRUDShape(s, cfg, crud)
			})
		case CRUDScan:
			s.Run("Scan", func() {
				testCRUDScan(s, cfg, cfg.Equal)
			})
		case CRUDRoundTrip:
			s.Run("RoundTrip", func() {
				testCRUDRoundTrip(s, cfg, crud, cfg.Equal)
			})
		default:
			require.Failf("unknown CRUD conformance phase %q", string(phase))
		}
	}
}

// Static metadata checks. Does not open a transaction, does not call CRUD Create/
// Retrieve/Update/Delete, and does not read or write the database.
//
// A single model from Create() is inspected. We only call Fields and Params on it.
func testCRUDShape[M tidal.Model](s *DatabaseSuite, cfg CRUDConformance[M], crud *tidal.CRUD[M]) {
	s.T().Helper()
	require := s.Require()
	m := cfg.Create()

	for _, op := range []tidal.Operation{tidal.List, tidal.Create, tidal.Retrieve, tidal.Update, tidal.Delete} {
		s.Run(op.String(), func() {
			// --- Fields(op): column names this operation expects ---
			fields := m.Fields(op)
			require.NotEmpty(fields, "Fields(%s) must not be empty", op)

			for _, field := range fields {
				require.NotEmpty(field, "Fields(%s) must not contain empty names", op)
			}

			// List, Retrieve, and Delete only define which columns Scan reads or which
			// columns appear in generated SQL; CRUD does not call Params for them.
			if op == tidal.List || op == tidal.Retrieve || op == tidal.Delete {
				return
			}

			// --- Params(op): named bind values for write/select-full-row operations ---
			params := m.Params(op)
			require.NotEmpty(params, "Params(%s) must not be empty", op)

			names := make([]string, 0, len(params))
			for _, param := range params {
				require.NotEmpty(param.Name, "Params(%s) must not contain empty names", op)
				require.False(slices.Contains(names, param.Name), "Params(%s) contains duplicate name %q", op, param.Name)
				names = append(names, param.Name)

				// Every param name must appear in Fields for this operation so SQL and
				// bindings stay aligned.
				require.Contains(fields, param.Name, "Params(%s) name %q is not in Fields(%s)", op, param.Name, op)
			}
		})
	}

	// --- Query strings: smoke-check that [tidal.New] produced non-empty SQL ---
	s.Run("Queries", func() {
		require.NotEmpty(crud.Queries.List)
		require.NotEmpty(crud.Queries.Create)
		require.NotEmpty(crud.Queries.Retrieve)
		require.NotEmpty(crud.Queries.Update)
		require.NotEmpty(crud.Queries.Delete)
	})
}

// Isolated Scan check. Does not use [tidal.CRUD] and does not touch the database.
//
// For each operation we:
//  1. Build a model with Create() ("original").
//  2. Derive fake row values from Params in Fields order (valuesForScan).
//  3. Feed those values into Scan via mockScanner (pretends to be *sql.Rows).
//  4. Compare the scanned model to original.
//
// This catches Scan/Fields/Params mismatches before paying for a DB round-trip.
// List is omitted because it uses a different column subset than Create/Retrieve.
func testCRUDScan[M tidal.Model](s *DatabaseSuite, cfg CRUDConformance[M], equal func(a, b M) bool) {
	s.T().Helper()
	require := s.Require()
	equalFn := equal
	if equalFn == nil {
		equalFn = func(a, b M) bool { return defaultEqual(s.T(), a, b) }
	}

	for _, op := range []tidal.Operation{tidal.Create, tidal.Retrieve, tidal.Update} {
		s.Run(op.String(), func() {
			// Source of truth: what the caller's factory produces.
			original := cfg.Create()

			// NOT a database row — values assembled from Params to mimic what a row
			// would contain if Params and Fields are consistent.
			values := valuesForScan(s.T(), original, op)
			require.Len(values, len(original.Fields(op)), "scan values and Fields(%s) length mismatch", op)

			// Fresh zero instance; only Scan should populate it.
			got := tidal.Make[M]()
			require.NoError(got.Scan(op, &mockScanner{values: values}))
			require.True(equalFn(original, got), "Scan(%s) did not round-trip model values", op)
		})
	}
}

// Full integration check against the real suite database ([DatabaseSuite].DB).
//
// Uses [tidal.CRUD] exactly as production code would. All work happens inside one
// transaction that is rolled back in defer — no commits, no persistent side effects.
//
// Does not use mockScanner or a mock database here.
func testCRUDRoundTrip[M tidal.Model](s *DatabaseSuite, cfg CRUDConformance[M], crud *tidal.CRUD[M], equal func(a, b M) bool) {
	s.T().Helper()
	require := s.Require()
	equalFn := equal
	if equalFn == nil {
		equalFn = func(a, b M) bool { return defaultEqual(s.T(), a, b) }
	}

	// Real transaction on the suite's live DB connection (SQLite file or Postgres).
	tx := s.BeginTx(nil)
	defer tx.Rollback() // always discarded — nothing survives this test

	// --- CREATE ---
	// Model from caller's factory; CRUD Create will call Prepare/Validate if implemented,
	// then Exec INSERT using the model's Params(Create).
	created := cfg.Create()
	_, err := crud.Create(tx, created)
	require.NoError(err, "Create failed")

	// If the model embeds BaseModel (Preparer), Create should have assigned an ID.
	if _, ok := any(created).(tidal.Preparer); ok {
		require.False(reflect.ValueOf(modelID(s.T(), created).Value).IsZero(), "ID should be set after Create")
	}

	// --- RETRIEVE (after create) ---
	// Reads the row back with SELECT ... WHERE id = :id. Compare in-memory created
	// (post-Prepare) against what Scan pulled from the database.
	retrieved, err := crud.Retrieve(tx, modelID(s.T(), created))
	require.NoError(err, "Retrieve after Create failed")
	require.True(equalFn(created, retrieved), "Retrieve after Create returned unexpected model")

	// --- LIST ---
	// Uses CRUD List with a manual WHERE clause filter. Expect exactly the row we inserted.
	// List scans fewer columns than Retrieve (per model's Fields(List)).
	filter := &tidal.Clause{
		SQL:  "WHERE id = :id",
		Args: []sql.NamedArg{modelID(s.T(), created)},
	}

	cursor, err := crud.List(tx, filter)
	require.NoError(err, "List failed")

	models, err := cursor.List()
	require.NoError(err, "List cursor failed")

	// SQLite blocks writes in the same transaction while sql.Rows is open; close
	// the result set only (not the transaction — cursor.Close would Rollback).
	require.NoError(cursor.CloseRows(), "List cursor close failed")
	require.Len(models, 1, "List should return exactly one model")

	// Compare only List columns (not full struct — List Scan omits password, etc.).
	require.True(equalListFields(s.T(), created, models[0], created.Fields(tidal.List), cfg.Equal), "List returned unexpected model")

	// --- UPDATE ---
	// Caller mutates the same in-memory instance; CRUD Update runs Prepare/Validate
	// then Exec UPDATE using Params(Update).
	cfg.Update(created) // created is mutated in place
	require.NoError(crud.Update(tx, created), "Update failed")

	// --- RETRIEVE (after update) ---
	updated, err := crud.Retrieve(tx, modelID(s.T(), created))
	require.NoError(err, "Retrieve after Update failed")
	require.True(equalFn(created, updated), "Retrieve after Update returned unexpected model")

	// --- DELETE ---
	_, err = crud.Delete(tx, modelID(s.T(), created))
	require.NoError(err, "Delete failed")

	// --- RETRIEVE (after delete) ---
	// Row should be gone; tidal returns sql.ErrNoRows from Scan on empty result.
	_, err = crud.Retrieve(tx, modelID(s.T(), created))
	require.Error(err, "Retrieve after Delete should fail")
	require.True(errors.Is(err, sql.ErrNoRows), "Retrieve after Delete should return sql.ErrNoRows")
}

// Reads the primary key from Params(Create) on an already-populated model.
func modelID[M tidal.Model](t testing.TB, m M) sql.NamedArg {
	t.Helper()
	for _, param := range m.Params(tidal.Create) {
		if param.Name == "id" {
			return sql.Named("id", param.Value)
		}
	}
	t.Fatalf("model %T does not expose id in Params(Create)", m)
	return sql.NamedArg{}
}

// Assembles fake row values for the Scan phase only.
//
// Takes values from Params (Create as base, then overlaid with Params(op)) and
// orders them to match Fields(op) — the same order Scan passes to sql.Rows.Scan.
// If a field appears in Fields but not in Params, the test fails (that is a model bug).
func valuesForScan[M tidal.Model](t testing.TB, m M, op tidal.Operation) []any {
	t.Helper()
	// Scan reads columns in Fields(op) order — this slice must match that exactly.
	fields := m.Fields(op)

	// Build a name→value lookup. Start with Create params (full row shape), then
	// overlay Params(op) so operation-specific bindings win (e.g. Update timestamps).
	byName := make(map[string]any, len(fields))
	for _, param := range m.Params(tidal.Create) {
		byName[param.Name] = param.Value
	}
	for _, param := range m.Params(op) {
		byName[param.Name] = param.Value
	}

	// Emit values in Fields order, not map iteration order.
	values := make([]any, len(fields))
	for i, field := range fields {
		value, ok := byName[field]
		if !ok {
			// Every column Scan expects must have a bind value somewhere in Params.
			t.Fatalf("model %T is missing value for Fields(%s) entry %q", m, op, field)
		}
		values[i] = value
	}
	return values
}

// Default Equal when the caller does not provide one.
func defaultEqual[M tidal.Model](tb testing.TB, a, b M) bool {
	tb.Helper()
	return equalValues(tb, reflect.ValueOf(a), reflect.ValueOf(b))
}

// Compares two models on the subset of columns returned by Fields(List). List
// Scan does not populate password, dob, etc., so we must not use full struct
// Equal.
func equalListFields[M tidal.Model](tb testing.TB, a, b M, columns []string, equal func(a, b M) bool) bool {
	tb.Helper()
	if equal != nil {
		return equal(a, b)
	}

	av := reflect.ValueOf(a)
	bv := reflect.ValueOf(b)

	// Models may be values or pointers; compare the struct underneath.
	for av.Kind() == reflect.Ptr {
		av = av.Elem()
	}
	for bv.Kind() == reflect.Ptr {
		bv = bv.Elem()
	}

	// Only check columns List actually scanned — unlisted fields stay zero-valued
	// on the List result and must not be compared.
	for _, column := range columns {
		af, ok := fieldByColumn(tb, av, column)
		if !ok {
			return false
		}
		bf, ok := fieldByColumn(tb, bv, column)
		if !ok {
			return false
		}
		if !equalValues(tb, af, bf) {
			return false
		}
	}
	return true
}

// Reflective deep compare with special cases for types that round-trip through
// SQL poorly.
func equalValues(tb testing.TB, a, b reflect.Value) bool {
	tb.Helper()

	// Peel pointers and interface wrappers until we reach concrete values.
	for a.Kind() == reflect.Ptr || a.Kind() == reflect.Interface {
		if a.IsNil() || b.IsNil() {
			return a.IsNil() && b.IsNil()
		}
		a = a.Elem()
		b = b.Elem()
	}

	if a.Type() != b.Type() {
		return false
	}

	// Custom field wrappers that need DB round-trip normalization.
	if a.Type() == reflect.TypeOf(fields.StringArray{}) {
		return a.Interface().(fields.StringArray).Equal(b.Interface().(fields.StringArray))
	}
	if a.Type() == reflect.TypeOf(fields.NullStringArray{}) {
		return a.Interface().(fields.NullStringArray).Equal(b.Interface().(fields.NullStringArray))
	}
	if a.Type() == reflect.TypeOf(fields.JSONB{}) {
		return a.Interface().(fields.JSONB).Equal(b.Interface().(fields.JSONB))
	}
	if a.Type() == reflect.TypeOf(fields.NullJSONB{}) {
		return a.Interface().(fields.NullJSONB).Equal(b.Interface().(fields.NullJSONB))
	}

	switch a.Kind() {
	case reflect.Struct:
		// Types that survive a DB round-trip but not reflect.DeepEqual:
		// time.Time, sql.NullTime, ulid.ULID.
		if a.Type() == reflect.TypeOf(time.Time{}) {
			return timeEqual(tb, a.Interface().(time.Time), b.Interface().(time.Time))
		}
		if a.Type() == reflect.TypeOf(sql.NullTime{}) {
			at := a.Interface().(sql.NullTime)
			bt := b.Interface().(sql.NullTime)
			if at.Valid != bt.Valid {
				return false
			}
			if !at.Valid {
				return true // both NULL — skip time comparison
			}
			return timeEqual(tb, at.Time, bt.Time)
		}
		if a.Type() == reflect.TypeOf(ulid.ULID{}) {
			// ULID is a [16]byte array; == compares bytes, not pointer identity.
			return a.Interface().(ulid.ULID) == b.Interface().(ulid.ULID)
		}
		// Generic struct: recurse field-by-field (handles embedded fields too).
		for i := 0; i < a.NumField(); i++ {
			if !equalValues(tb, a.Field(i), b.Field(i)) {
				return false
			}
		}
		return true
	default:
		return reflect.DeepEqual(a.Interface(), b.Interface())
	}
}

// Locates a struct field for a database column name (snake_case), including embedded structs.
func fieldByColumn(tb testing.TB, v reflect.Value, column string) (reflect.Value, bool) {
	tb.Helper()
	typ := v.Type()
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if field.Anonymous {
			// Embedded BaseModel, etc. — search inside before giving up.
			if fv, ok := fieldByColumn(tb, v.Field(i), column); ok {
				return fv, true
			}
			continue
		}

		// Match DB column name (snake_case) to Go field name (CamelCase).
		if typecase.Snake(field.Name) == column {
			return v.Field(i), true
		}
	}
	return reflect.Value{}, false
}

// Database drivers often truncate timestamps; compare at second precision in UTC.
func timeEqual(tb testing.TB, a, b time.Time) bool {
	tb.Helper()
	if a.IsZero() && b.IsZero() {
		return true
	}
	// SQLite stores datetimes without sub-second precision; Postgres may differ
	// in location. Normalize before comparing in-memory vs retrieved values.
	return a.UTC().Truncate(time.Second).Equal(b.UTC().Truncate(time.Second))
}

// Fake [tidal.Scanner] for the Scan phase only. Holds predetermined cell values
// and assigns them into the pointers Scan passes — same contract as *sql.Rows.Scan.
// Never connects to a database.
type mockScanner struct {
	values []any
}

// Satisfies [tidal.Scanner]. Copies predetermined values into dest pointers.
func (m *mockScanner) Scan(dest ...any) error {
	// model.Scan passes one pointer per Fields(op) column — count must match our
	// fake row built by valuesForScan.
	if len(dest) != len(m.values) {
		return fmt.Errorf("mockScanner: expected %d destinations, got %d", len(m.values), len(dest))
	}

	for i, d := range dest {
		// dest entries are *T pointers; assignValue writes through them like sql.Rows.Scan.
		if err := assignValue(d, m.values[i]); err != nil {
			return fmt.Errorf("mockScanner: destination %d: %w", i, err)
		}
	}
	return nil
}

// Reflective assignment helper for mockScanner.Scan.
func assignValue(dest, value any) error {
	dv := reflect.ValueOf(dest)
	if dv.Kind() != reflect.Ptr || dv.IsNil() {
		return fmt.Errorf("destination must be a non-nil pointer")
	}

	dv = dv.Elem() // *T → T, the field Scan wants populated
	vv := reflect.ValueOf(value)

	if !vv.IsValid() {
		// nil interface in the fake row → zero the destination (SQL NULL).
		dv.SetZero()
		return nil
	}

	if vv.Type().AssignableTo(dv.Type()) {
		dv.Set(vv)
		return nil
	}

	// e.g. int64 source into a typed alias or narrower numeric field.
	if vv.Type().ConvertibleTo(dv.Type()) {
		dv.Set(vv.Convert(dv.Type()))
		return nil
	}

	return fmt.Errorf("cannot assign %T to %T", value, dest)
}
