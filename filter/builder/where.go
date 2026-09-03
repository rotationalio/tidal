package builder

import (
	"database/sql"
	"fmt"
	"slices"
	"strings"
)

//============================================================================
// Where
//============================================================================

// Where builds a WHERE expression from conditions, logical operators, and groups.
type Where struct {
	root *whereGroup

	// Parameters used by trusted SQL expressions such as subqueries.
	params []sql.NamedArg
}

// Clears the WHERE expression and any parameters attached to it.
func (w *Where) Reset() {
	w.root = nil
	w.params = nil
}

// Reports whether the WHERE expression has no conditions.
func (w *Where) IsEmpty() bool {
	return w.root == nil || len(w.root.children) == 0
}

// Replaces any existing expression with a single condition.
func (w *Where) Set(field string, op WhereOp, value any) {
	w.root = &whereGroup{
		children: []whereNode{{node: &whereCondition{field: field, op: op, value: value}}},
	}
}

// Appends a condition joined with AND.
func (w *Where) Where(field string, op WhereOp, value any) *Where {
	w.And(field, op, value)
	return w
}

// Appends a condition joined with AND.
func (w *Where) And(field string, op WhereOp, value any) *Where {
	w.append(logAnd, field, op, value)
	return w
}

// Appends a condition joined with OR.
func (w *Where) Or(field string, op WhereOp, value any) *Where {
	w.append(logOr, field, op, value)
	return w
}

// Appends a parenthesized group joined with AND.
func (w *Where) AndGroup(fn func(*Where)) {
	w.appendGroup(logAnd, fn)
}

// Appends a parenthesized group joined with OR.
func (w *Where) OrGroup(fn func(*Where)) {
	w.appendGroup(logOr, fn)
}

// Adds or replaces a named parameter available to SQL expressions in the WHERE
// clause.
func (w *Where) Param(name string, value any) *Where {
	for i := range w.params {
		if w.params[i].Name == name {
			w.params[i] = sql.Named(name, value)
			return w
		}
	}
	w.params = append(w.params, sql.Named(name, value))
	return w
}

// Copies the WHERE expression and its named parameters.
func (w *Where) Clone() *Where {
	if w == nil {
		return nil
	}

	return &Where{
		root:   cloneWhereGroup(w.root),
		params: slices.Clone(w.params),
	}
}

// Renders the WHERE clause and named parameters in traversal order.
func (w *Where) Render() (string, []sql.NamedArg) {
	if w.IsEmpty() {
		return "", slices.Clone(w.params)
	}

	var params []sql.NamedArg
	idx := 0
	clause := w.root.render(&params, &idx)
	params = append(params, w.params...)
	return "WHERE " + clause, params
}

// Appends a condition to the current WHERE clause.
func (w *Where) append(logical logicalOp, field string, op WhereOp, value any) {
	cond := &whereCondition{field: field, op: op, value: value}
	if w.IsEmpty() {
		w.root = &whereGroup{children: []whereNode{{node: cond}}}
		return
	}
	w.root.children = append(w.root.children, whereNode{logical: logical, node: cond})
}

// Appends a parenthesized group to the current WHERE clause using the provided
// function to build the group.
func (w *Where) appendGroup(logical logicalOp, fn func(*Where)) {
	sub := &Where{}
	fn(sub)
	if sub.IsEmpty() {
		return
	}

	group := sub.root
	group.grouped = true

	if w.IsEmpty() {
		w.root = group
		return
	}
	w.root.children = append(w.root.children, whereNode{logical: logical, node: group})
}

// Copies a WHERE group and all of its expression nodes.
func cloneWhereGroup(group *whereGroup) *whereGroup {
	if group == nil {
		return nil
	}

	clone := &whereGroup{
		children: make([]whereNode, len(group.children)),
		grouped:  group.grouped,
	}
	for i, child := range group.children {
		clone.children[i].logical = child.logical
		switch node := child.node.(type) {
		case *whereCondition:
			clone.children[i].node = &whereCondition{
				field: node.field,
				op:    node.op,
				value: node.value,
			}
		case *whereGroup:
			clone.children[i].node = cloneWhereGroup(node)
		}
	}
	return clone
}

//============================================================================
// Expression model
//============================================================================

type logicalOp uint8

const (
	logNone logicalOp = iota
	logAnd
	logOr
)

func (op logicalOp) String() string {
	switch op {
	case logAnd:
		return "AND"
	case logOr:
		return "OR"
	default:
		return "unknown"
	}
}

type whereCondition struct {
	field string
	op    WhereOp
	value any
}

type whereGroup struct {
	children []whereNode
	grouped  bool
}

type whereNode struct {
	logical logicalOp
	node    any // *whereCondition or *whereGroup
}

func (g *whereGroup) render(params *[]sql.NamedArg, idx *int) string {
	parts := make([]string, 0, len(g.children))
	for _, child := range g.children {
		part := renderNode(child.node, params, idx)
		if child.logical != logNone {
			parts = append(parts, child.logical.String(), part)
			continue
		}
		parts = append(parts, part)
	}

	clause := strings.Join(parts, " ")
	if g.grouped {
		return "(" + clause + ")"
	}
	return clause
}

func renderNode(node any, params *[]sql.NamedArg, idx *int) string {
	switch n := node.(type) {
	case *whereCondition:
		// If we have a quantified operator (ALL/ANY), render it as a quantified
		// condition like: "field op quantifier(values)" (ex: `foo != ANY(:w1)`).
		if comparison, quantifier, ok := quantifiedOperator(n.op); ok {
			return renderQuantified(n.field, comparison, quantifier, n.value, params, idx)
		}

		switch n.op {
		case In, NotIn:
			// Render subqueries directly
			if query, ok := subqueryValue(n.value); ok {
				return fmt.Sprintf("%s %s (%s)", n.field, n.op, query)
			}

			// If there are no values, return a condition that always evaluates
			// to false/true
			vals := inValues(n.value)
			if len(vals) == 0 {
				if n.op == In {
					return "1=0"
				}
				return "1=1"
			}

			// Render the IN/NOT IN condition with placeholders
			placeholders := make([]string, len(vals))
			for i, v := range vals {
				*idx++
				name := fmt.Sprintf("w%d", *idx)
				*params = append(*params, sql.Named(name, v))
				placeholders[i] = ":" + name
			}
			return fmt.Sprintf("%s %s (%s)", n.field, n.op, strings.Join(placeholders, ", "))
		case Is, IsNot:
			if sym, ok := n.value.(Literal); ok {
				// Literal value like TRUE/FALSE/etc.
				return fmt.Sprintf("%s %s %s", n.field, n.op, sym)
			}
			fallthrough // non-literal values render "field op value"
		default:
			// All other operators are rendered as: "field op value"
			*idx++
			name := fmt.Sprintf("w%d", *idx)
			*params = append(*params, sql.Named(name, n.value))
			return fmt.Sprintf("%s %s :%s", n.field, n.op, name)
		}
	case *whereGroup:
		return n.render(params, idx)
	default:
		return ""
	}
}

// Renders a quantified comparison against an array parameter or subquery.
func renderQuantified(field string, comparison WhereOp, quantifier string, value any, params *[]sql.NamedArg, idx *int) string {
	if query, ok := subqueryValue(value); ok {
		return fmt.Sprintf("%s %s %s (%s)", field, comparison, quantifier, query)
	}

	if isComparisonOperator(comparison) && isEmptySet(value) {
		// Supported quantifiers return their boolean identity above. Any other
		// quantifier combination is invalid; leave it malformed so SQL validation
		// fails instead of binding a made-up parameter.
		if quantifier == "ANY" {
			return "1=0"
		}
		if quantifier == "ALL" {
			return "1=1"
		}
		return fmt.Sprintf("%s %s %s", field, comparison, quantifier)
	}

	*idx++
	name := fmt.Sprintf("w%d", *idx)
	*params = append(*params, sql.Named(name, value))
	return fmt.Sprintf("%s %s %s (:%s)", field, comparison, quantifier, name)
}

//============================================================================
// Field Prefixing for Joins
//============================================================================

var (
	_ Prefixer = (*Where)(nil)
	_ Prefixer = (*whereNode)(nil)
	_ Prefixer = (*whereGroup)(nil)
	_ Prefixer = (*whereCondition)(nil)
)

func (w *Where) Prefix(tableAlias string, fields ...string) {
	if w.root != nil {
		w.root.Prefix(tableAlias, fields...)
	}
}

func (w *whereNode) Prefix(tableAlias string, fields ...string) {
	if prefixer, ok := w.node.(Prefixer); ok {
		prefixer.Prefix(tableAlias, fields...)
	}
}

func (w *whereGroup) Prefix(tableAlias string, fields ...string) {
	for _, child := range w.children {
		child.Prefix(tableAlias, fields...)
	}
}

func (c *whereCondition) Prefix(tableAlias string, fields ...string) {
	c.field = Prefix(c.field, tableAlias, fields...)
}
