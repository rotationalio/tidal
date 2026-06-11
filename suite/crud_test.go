package suite

import (
	"database/sql"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.rtnl.ai/tidal"
	"go.rtnl.ai/ulid"
)

//============================================================================
// columnName
//============================================================================

// Verifies CamelCase → snake_case mapping used by [fieldByColumn] and [equalListFields].
func TestColumnName(t *testing.T) {
	tests := []struct {
		field string
		want  string
	}{
		// no split — all lowercase, all caps, acronym-with-digit tail, rule 2 OA/IPv exception
		{"lowercase", "lowercase"},
		{"A", "a"},
		{"ID", "id"},
		{"HTTP", "http"},
		{"SOS", "sos"},
		{"CAPTCHA", "captcha"},
		{"UTF8", "utf8"},
		{"OAuth", "oauth"},
		{"OAuth2", "oauth2"},
		{"IPv6", "ipv6"},
		{"IPv666", "ipv666"}, // the devil's favorite layer 3 protocol

		// Rule 1: lower → Upper → lower
		{"LastSeen", "last_seen"},
		{"GoVariable", "go_variable"},
		{"VeryLongCapitalizedNameHasALotOfWords", "very_long_capitalized_name_has_a_lot_of_words"},
		{"QualityOfLife", "quality_of_life"},
		{"DateOfBirth", "date_of_birth"},

		// Rule 2: Upper… → lower
		{"URLPath", "url_path"},
		{"APIKey", "api_key"},
		{"JSONData", "json_data"},
		{"HTMLParser", "html_parser"},
		{"XMLHttp", "xml_http"},
		{"HTTPServer", "http_server"},
		{"ABTest", "ab_test"},

		// Rule 3: lower → Upper…
		{"UserID", "user_id"},
		{"UserA", "user_a"},
		{"NetflixID", "netflix_id"},
		{"UserAB", "user_ab"},
		{"UserHTTP", "user_http"},
		{"getHTTP", "get_http"},

		// Rule 4: Cap–lower–Cap particle — suppress split
		{"DoB", "dob"},
		{"QoL", "qol"},
		{"MoU", "mou"},
		{"SoS", "sos"},
		{"IoT", "iot"},

		// Rule 5: digit → letter
		{"v2API", "v2_api"},
		{"V2API", "v2_api"},
		{"v2api", "v2_api"},
		{"V2api", "v2_api"},
		{"12angrymen", "12_angrymen"},

		// combinations — multiple rules in one name
		{"UserQoL", "user_qol"},                                // word → acronym particle
		{"HasAMiddleQoLParticle", "has_a_middle_qol_particle"}, // acronym particle in middle of words
		{"UserIDAndEmail", "user_id_and_email"},                // word → acronym → word
		{"SomethingHTTPHeader", "something_http_header"},       // word → acronym → word
		{"QoLUser", "qol_user"},                                // acronym particle → word
		{"SoSUser", "sos_user"},                                // acronym particle → word
		{"IoTDevice", "iot_device"},                            // acronym particle → word
		{"OAuth2Token", "oauth2_token"},                        // initial capital character exception + digit boundary
		{"12AngryMen", "12_angry_men"},                         // digit boundary + word

		// known limitations — proper nouns, lowercase brand prefixes, etc.
		{"iOS", "i_os"},
		{"iOSSDK", "i_ossdk"},
		{"iOSApp", "i_os_app"},
		{"tvOS", "tv_os"},
		{"macOS", "mac_os"},
		{"eBay", "e_bay"},
		{"iTunes", "i_tunes"},
		{"McDonalds", "mc_donalds"},
	}
	for _, tc := range tests {
		t.Run(tc.field, func(t *testing.T) {
			require.Equal(t, tc.want, columnName(t, tc.field))
		})
	}
}

//============================================================================
// valuesForScan
//============================================================================

// Verifies [valuesForScan] assembles fake row values in Fields(op) order from Params.
func TestValuesForScan(t *testing.T) {
	uid := ulid.MustParse("01KTESYNDPVTRWK05N2TXFKGQZ")
	other := ulid.MustParse("01KTESYNDPVTRWK05N2TXFKGQ0")
	seen := sql.NullTime{Valid: true, Time: time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)}
	created := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	modified := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)

	m := &scanHelperModel{
		BaseModel: tidal.BaseModel{ID: uid, Created: created, Modified: modified},
		URLPath:   "/v1/items",
		UserID:    other,
		LastSeen:  seen,
	}

	t.Run("RetrieveFieldsOrder", func(t *testing.T) {
		// Full-row operations return values in Fields(Retrieve) column order.
		values := valuesForScan(t, m, tidal.Retrieve)
		require.Equal(t, []any{uid, "/v1/items", other, seen, created, modified}, values)
	})

	t.Run("UpdateOverlay", func(t *testing.T) {
		// Params(Update) overlays Create — only update columns appear, in Fields(Update) order.
		updated := modified.Add(24 * time.Hour)
		m.Modified = updated
		values := valuesForScan(t, m, tidal.Update)
		require.Equal(t, []any{uid, "/v1/items", updated}, values)
	})

	t.Run("MissingParamFails", func(t *testing.T) {
		// A column in Fields with no matching Params entry is a model bug; helper must fail.
		cap := &fatalCapture{T: t}
		valuesForScan(cap, &brokenScanModel{}, tidal.Retrieve)
		require.Contains(t, cap.msg, `missing value for Fields(Retrieve) entry "missing_col"`)
	})
}

//============================================================================
// modelID
//============================================================================

// Verifies [modelID] reads the primary key from Params(Create).
func TestModelID(t *testing.T) {
	t.Run("Found", func(t *testing.T) {
		uid := ulid.MustParse("01KTESYNDPVTRWK05N2TXFKGQZ")
		m := &scanHelperModel{BaseModel: tidal.BaseModel{ID: uid}}
		arg := modelID(t, m)
		require.Equal(t, "id", arg.Name)
		require.Equal(t, uid, arg.Value)
	})

	t.Run("MissingFails", func(t *testing.T) {
		// Models without id in Params(Create) cannot drive Retrieve/List/Update/Delete.
		cap := &fatalCapture{T: t}
		modelID(cap, noIDModel{})
		require.Contains(t, cap.msg, "does not expose id in Params(Create)")
	})
}

//============================================================================
// fieldByColumn
//============================================================================

// Verifies [fieldByColumn] maps database column names (snake_case) back to struct fields,
// including embedded [tidal.BaseModel] and acronym fields like URLPath → url_path.
func TestFieldByColumn(t *testing.T) {
	uid := ulid.MustParse("01KTESYNDPVTRWK05N2TXFKGQZ")
	m := &scanHelperModel{
		BaseModel: tidal.BaseModel{ID: uid},
		URLPath:   "/x",
		UserID:    uid,
	}

	v := reflect.ValueOf(m).Elem()

	fv, ok := fieldByColumn(t, v, "url_path")
	require.True(t, ok)
	require.Equal(t, "/x", fv.String())

	fv, ok = fieldByColumn(t, v, "id")
	require.True(t, ok)
	require.Equal(t, uid, fv.Interface())

	_, ok = fieldByColumn(t, v, "no_such_column")
	require.False(t, ok)
}

//============================================================================
// equalListFields
//============================================================================

// Verifies [equalListFields] compares only the Fields(List) column subset, ignoring
// columns that List Scan does not populate (e.g. last_seen on [scanHelperModel]).
func TestEqualListFields(t *testing.T) {
	uid := ulid.MustParse("01KTESYNDPVTRWK05N2TXFKGQZ")
	other := ulid.MustParse("01KTESYNDPVTRWK05N2TXFKGQ0")

	a := &scanHelperModel{
		BaseModel: tidal.BaseModel{ID: uid},
		URLPath:   "/a",
		UserID:    other,
		LastSeen:  sql.NullTime{Valid: true, Time: time.Now()},
	}
	b := &scanHelperModel{
		BaseModel: tidal.BaseModel{ID: uid},
		URLPath:   "/a",
		UserID:    other,
	}

	require.True(t, equalListFields(t, a, b, a.Fields(tidal.List)))
	require.False(t, equalListFields(t, a, &scanHelperModel{URLPath: "/b", UserID: other}, a.Fields(tidal.List)))
}

//============================================================================
// mockScanner
//============================================================================

// Verifies [mockScanner] satisfies the [tidal.Scanner] contract used by the Scan
// conformance phase.
func TestMockScanner(t *testing.T) {
	uid := ulid.MustParse("01KTESYNDPVTRWK05N2TXFKGQZ")
	values := []any{uid, "/path", uid}

	t.Run("HappyPath", func(t *testing.T) {
		got := &scanHelperModel{}
		require.NoError(t, got.Scan(tidal.List, &mockScanner{values: values}))
		require.Equal(t, uid, got.ID)
		require.Equal(t, "/path", got.URLPath)
		require.Equal(t, uid, got.UserID)
	})

	t.Run("CountMismatch", func(t *testing.T) {
		// Destination count must match Fields(op) length — same rule as *sql.Rows.Scan.
		got := &scanHelperModel{}
		err := got.Scan(tidal.List, &mockScanner{values: values[:1]})
		require.ErrorContains(t, err, "expected 1 destinations, got 3")
	})

	t.Run("NilValue", func(t *testing.T) {
		// nil in the fake row represents a SQL NULL and zeroes the destination.
		var s string
		require.NoError(t, (&mockScanner{values: []any{nil}}).Scan(&s))
		require.Empty(t, s)
	})
}

//============================================================================
// Test Helpers
//============================================================================

// fatalCapture records Fatalf/Fatal without stopping the test runner, so negative-path
// helper tests can assert on the failure message.
type fatalCapture struct {
	*testing.T
	msg string
}

func (f *fatalCapture) Fatal(args ...any) {
	f.msg = fmt.Sprint(args...)
}

func (f *fatalCapture) Fatalf(format string, args ...any) {
	f.msg = fmt.Sprintf(format, args...)
}

//============================================================================
// Test Models
//============================================================================

// scanHelperModel exercises acronym column naming (URLPath → url_path) and embedded
// BaseModel fields. Not used against a real database.
type scanHelperModel struct {
	tidal.BaseModel
	URLPath  string
	UserID   ulid.ULID
	LastSeen sql.NullTime
}

var _ tidal.Model = (*scanHelperModel)(nil)

func (m *scanHelperModel) Fields(op tidal.Operation) []string {
	switch op {
	case tidal.List:
		return []string{"id", "url_path", "user_id"}
	case tidal.Update:
		return []string{"id", "url_path", "modified"}
	default:
		return []string{"id", "url_path", "user_id", "last_seen", "created", "modified"}
	}
}

func (m *scanHelperModel) Params(op tidal.Operation) []sql.NamedArg {
	switch op {
	case tidal.Update:
		return []sql.NamedArg{
			sql.Named("id", m.ID),
			sql.Named("url_path", m.URLPath),
			sql.Named("modified", m.Modified),
		}
	default:
		return []sql.NamedArg{
			sql.Named("id", m.ID),
			sql.Named("url_path", m.URLPath),
			sql.Named("user_id", m.UserID),
			sql.Named("last_seen", m.LastSeen),
			sql.Named("created", m.Created),
			sql.Named("modified", m.Modified),
		}
	}
}

func (m *scanHelperModel) Scan(op tidal.Operation, s tidal.Scanner) error {
	switch op {
	case tidal.List:
		return s.Scan(&m.ID, &m.URLPath, &m.UserID)
	case tidal.Update:
		return s.Scan(&m.ID, &m.URLPath, &m.Modified)
	default:
		return s.Scan(&m.ID, &m.URLPath, &m.UserID, &m.LastSeen, &m.Created, &m.Modified)
	}
}

// noIDModel has no id in Params(Create) — used to test [modelID] failure path.
type noIDModel struct{}

func (noIDModel) Fields(tidal.Operation) []string { return []string{"id"} }
func (noIDModel) Params(tidal.Operation) []sql.NamedArg {
	return []sql.NamedArg{sql.Named("name", "x")}
}
func (noIDModel) Scan(tidal.Operation, tidal.Scanner) error { return nil }

// brokenScanModel declares a Fields column with no Params value — used to test
// [valuesForScan] failure path.
type brokenScanModel struct {
	tidal.BaseModel
}

func (brokenScanModel) Fields(tidal.Operation) []string { return []string{"id", "missing_col"} }
func (brokenScanModel) Params(tidal.Operation) []sql.NamedArg {
	return []sql.NamedArg{sql.Named("id", ulid.ULID{})}
}
func (brokenScanModel) Scan(tidal.Operation, tidal.Scanner) error { return nil }
