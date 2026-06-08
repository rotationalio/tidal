package tidal

import (
	"database/sql"
	"fmt"
)

func QueryParams(args []sql.NamedArg, placeholder PlaceholderType) Params {
	params := Params{
		placeholder:  placeholder,
		placeholders: make(map[string]string),
		values:       make([]any, len(args)),
	}

	for i, arg := range args {
		switch placeholder {
		case Positional:
			params.placeholders[arg.Name] = "?"
		case Ordered:
			params.placeholders[arg.Name] = fmt.Sprintf("$%d", i+1)
		case Named:
			params.placeholders[arg.Name] = fmt.Sprintf(":%s", arg.Name)
		case AtP:
			params.placeholders[arg.Name] = "@p"
		}
		params.values = append(params.values, arg.Value)
	}

	return params
}

type Params struct {
	placeholder  PlaceholderType
	placeholders map[string]string
	values       []any
}

func (p Params) Placeholder(name string) (string, bool) {
	placeholder, ok := p.placeholders[name]
	return placeholder, ok
}

func (p Params) Args() []any {
	return p.values
}

type PlaceholderType uint8

const (
	UnknownPlaceholder PlaceholderType = iota
	Positional
	Ordered
	Named
	AtP
)

func (p PlaceholderType) String() string {
	switch p {
	case Positional:
		return "?"
	case Ordered:
		return "$N"
	case Named:
		return ":name"
	case AtP:
		return "@p"
	default:
		return "unknown"
	}
}
