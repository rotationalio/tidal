package mock

import (
	"database/sql"
	"sync"

	"go.rtnl.ai/tidal/model"
)

var (
	mu            sync.Mutex
	onFields      func(op model.Operation) []string
	onParams      func(op model.Operation) []sql.NamedArg
	defaultFields = []string{"foo", "bar"}
	defaultParams = []sql.NamedArg{{Name: "foo", Value: "qux"}, {Name: "bar", Value: "baz"}}
	globalCalls   map[string]int
)

func OnFields(fn func(op model.Operation) []string) {
	mu.Lock()
	defer mu.Unlock()
	onFields = fn
}

func OnParams(fn func(op model.Operation) []sql.NamedArg) {
	mu.Lock()
	defer mu.Unlock()
	onParams = fn
}

func Reset() {
	mu.Lock()
	defer mu.Unlock()
	onFields = nil
	onParams = nil
	globalCalls = nil
}

const (
	Scan        = "Scan"
	Fields      = "Fields"
	Params      = "Params"
	Prepare     = "Prepare"
	Validate    = "Validate"
	Identifiers = "Identifiers"
)

// Mocks the [tidal.SimpleModel] interface.
type SimpleModel struct {
	Mock
	OnScan   func(op model.Operation, s model.Scanner) error
	OnFields func(op model.Operation) []string
	OnParams func(op model.Operation) []sql.NamedArg
}

// Mocks the [model.Scanner] interface.
type Scanner struct {
	Mock
	OnScan func(dest ...any) error
}

// Mocks the [tidal.Model] and [model.Preparer] interfaces.
type Preparer struct {
	SimpleModel
	OnPrepare func(op model.Operation)
}

// Mocks the [tidal.Model] and [model.Validator] interfaces.
type Validator struct {
	SimpleModel
	OnValidate func(op model.Operation) error
}

// Mocks the [tidal.Model] and [tidal.Identifier] interfaces.
type Identifier struct {
	SimpleModel
	OnIdentifier func() []sql.NamedArg
}

// Mocks the [tidal.Model], [model.Preparer], and [model.Validator] interfaces.
type Model struct {
	SimpleModel
	OnPrepare  func(op model.Operation)
	OnValidate func(op model.Operation) error
}

// Mocks the [tidal.Model], [model.Preparer], [model.Validator], and [tidal.Identifier] interfaces.
type CompleteModel struct {
	SimpleModel
	OnPrepare    func(op model.Operation)
	OnValidate   func(op model.Operation) error
	OnIdentifier func() []sql.NamedArg
}

//============================================================================
// Helper Methods
//============================================================================

func (m *SimpleModel) Reset() {
	m.Mock.Reset()
	m.OnScan = nil

	mu.Lock()
	defer mu.Unlock()
	onFields = nil
	onParams = nil
	globalCalls = nil
}

func (s *Scanner) Reset() {
	s.Mock.Reset()
	s.OnScan = nil
}

func (p *Preparer) Reset() {
	p.SimpleModel.Reset()
	p.OnPrepare = nil
}

func (v *Validator) Reset() {
	v.SimpleModel.Reset()
	v.OnValidate = nil
}

func (i *Identifier) Reset() {
	i.SimpleModel.Reset()
	i.OnIdentifier = nil
}

func (m *Model) Reset() {
	m.SimpleModel.Reset()
	m.OnPrepare = nil
	m.OnValidate = nil
}

func (m *CompleteModel) Reset() {
	m.SimpleModel.Reset()
	m.OnPrepare = nil
	m.OnValidate = nil
	m.OnIdentifier = nil
}

//============================================================================
// Implement Interfaces
//============================================================================

var (
	_ model.Model      = (*SimpleModel)(nil)
	_ model.Preparer   = (*Preparer)(nil)
	_ model.Validator  = (*Validator)(nil)
	_ model.Identifier = (*Identifier)(nil)
	_ model.Model      = (*Model)(nil)
	_ model.Model      = (*CompleteModel)(nil)
)

func (m *SimpleModel) Scan(op model.Operation, s model.Scanner) error {
	m.increment(Scan)
	if m.OnScan == nil {
		return nil
	}
	return m.OnScan(op, s)
}

func (m *SimpleModel) Fields(op model.Operation) []string {
	if m.OnFields != nil {
		m.increment(Fields)
		return m.OnFields(op)
	}

	mu.Lock()
	defer mu.Unlock()

	if globalCalls == nil {
		globalCalls = make(map[string]int)
	}
	globalCalls[Fields]++

	if onFields != nil {
		return onFields(op)
	}
	return defaultFields
}

func (m *SimpleModel) Params(op model.Operation) []sql.NamedArg {
	if m.OnParams != nil {
		m.increment(Params)
		return m.OnParams(op)
	}

	mu.Lock()
	defer mu.Unlock()

	if globalCalls == nil {
		globalCalls = make(map[string]int)
	}
	globalCalls[Params]++

	if onParams != nil {
		return onParams(op)
	}
	return defaultParams
}

var (
	_ model.Preparer = (*Preparer)(nil)
	_ model.Preparer = (*Model)(nil)
	_ model.Preparer = (*CompleteModel)(nil)
)

func (p *Preparer) Prepare(op model.Operation) {
	p.increment(Prepare)
	p.OnPrepare(op)
}

func (p *Model) Prepare(op model.Operation) {
	p.increment(Prepare)
	p.OnPrepare(op)
}

func (p *CompleteModel) Prepare(op model.Operation) {
	p.increment(Prepare)
	p.OnPrepare(op)
}

var (
	_ model.Validator = (*Validator)(nil)
	_ model.Validator = (*Model)(nil)
	_ model.Validator = (*CompleteModel)(nil)
)

func (v *Validator) Validate(op model.Operation) error {
	v.increment(Validate)
	return v.OnValidate(op)
}

func (v *Model) Validate(op model.Operation) error {
	v.increment(Validate)
	return v.OnValidate(op)
}

func (v *CompleteModel) Validate(op model.Operation) error {
	v.increment(Validate)
	return v.OnValidate(op)
}

var (
	_ model.Identifier = (*Identifier)(nil)
	_ model.Identifier = (*CompleteModel)(nil)
)

func (i *Identifier) Identifiers() []sql.NamedArg {
	i.increment(Identifiers)
	return i.OnIdentifier()
}

func (i *CompleteModel) Identifiers() []sql.NamedArg {
	i.increment(Identifiers)
	return i.OnIdentifier()
}
