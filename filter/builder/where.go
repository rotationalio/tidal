package builder

import (
	"database/sql"
	"fmt"
	"strings"
)

//============================================================================
// Where
//============================================================================

// Where builds a WHERE expression from conditions, logical operators, and groups.
type Where struct {
	root *whereGroup
}

// Reset clears any built WHERE expression.
func (w *Where) Reset() {
	w.root = nil
}

// IsEmpty reports whether the WHERE expression has no conditions.
func (w *Where) IsEmpty() bool {
	return w.root == nil || len(w.root.children) == 0
}

// Set replaces any existing expression with a single condition.
func (w *Where) Set(field string, op WhereOp, value any) {
	w.root = &whereGroup{
		children: []whereNode{{node: &whereCondition{field: field, op: op, value: value}}},
	}
}

// Where replaces any existing expression with a single condition.
func (w *Where) Where(field string, op WhereOp, value any) *Where {
	w.Set(field, op, value)
	return w
}

// And appends a condition joined with AND.
func (w *Where) And(field string, op WhereOp, value any) *Where {
	w.append(logAnd, field, op, value)
	return w
}

// Or appends a condition joined with OR.
func (w *Where) Or(field string, op WhereOp, value any) *Where {
	w.append(logOr, field, op, value)
	return w
}

// AndGroup appends a parenthesized group joined with AND.
func (w *Where) AndGroup(fn func(*Where)) {
	w.appendGroup(logAnd, fn)
}

// OrGroup appends a parenthesized group joined with OR.
func (w *Where) OrGroup(fn func(*Where)) {
	w.appendGroup(logOr, fn)
}

// Render returns the WHERE clause and named parameters in traversal order.
func (w *Where) Render() (string, []sql.NamedArg) {
	if w.IsEmpty() {
		return "", nil
	}

	var params []sql.NamedArg
	idx := 0
	clause := w.root.render(&params, &idx)
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
		*idx++
		name := fmt.Sprintf("w%d", *idx)
		*params = append(*params, sql.Named(name, n.value))
		return fmt.Sprintf("%s %s :%s", n.field, n.op, name)
	case *whereGroup:
		return n.render(params, idx)
	default:
		return ""
	}
}
