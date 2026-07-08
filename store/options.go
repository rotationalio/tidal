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
}

type Option func(*Options)

func WithNoUpdateLimit() Option {
	return func(o *Options) {
		o.UpdateLimit = false
	}
}

func WithIDField(field string) Option {
	return func(o *Options) {
		o.IDField = field
	}
}

func makeOptions(opts ...Option) Options {
	o := Options{
		UpdateLimit: true,
		IDField:     "id",
	}
	for _, opt := range opts {
		opt(&o)
	}
	return o
}
