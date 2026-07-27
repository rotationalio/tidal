package store

type Options struct {
	// Add LIMIT 1 to the end of the UPDATE query for safety if true.
	UpdateLimit bool

	// Default immutable identifier field for updates (default: "id")
	// This identifier is used if the model does not implement the model.Identifier
	// interface. It must be a field returned by the Params() method for update
	// operations or an error will be returned.
	//
	// NOTE: this field is ignored in the update set list.
	// Valid values for this are uuid, pk, etc. E.g. a field that is set on create and
	// then never changed. If you want to use a unique field for updates that can change,
	// e.g. slug or email, then you should implement the model.Identifier interface.
	IDField string

	// If debug is true, the CRUD will print the SQL queries and parameters to the console.
	Debug bool
}

type Option func(*Options)

// For Update operations, and a LIMIT 1 clause to the end of the query. This is useful
// to ensure only one row is updated but is not available in all databases (e.g. SQLite).
func WithUpdateLimit() Option {
	return func(o *Options) {
		o.UpdateLimit = true
	}
}

// Specify the default immutable identifier field for updates.
func WithIDField(field string) Option {
	return func(o *Options) {
		o.IDField = field
	}
}

// If debug is true, the CRUD will print the SQL queries and parameters to the console.
func WithDebug() Option {
	return func(o *Options) {
		o.Debug = true
	}
}

func makeOptions(opts ...Option) Options {
	o := Options{
		UpdateLimit: false,
		IDField:     "id",
	}
	for _, opt := range opts {
		opt(&o)
	}
	return o
}
