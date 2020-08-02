package validate

import (
	"encoding/json"
	"fmt"
)

func parse(validator string, rules map[string]Rule) (*node, error) {
	l := newLexer(validator)
	if debug {
		fmt.Println("***")
		fmt.Println(validator)
		defer fmt.Println("***")
	}
	return parseBools(l, rules)
}

func parseBools(l *lexer, rules map[string]Rule) (*node, error) {
	var current *node
	for {
		t := l.Next()
		isEmptyNode := current == nil

		if debug {
			fmt.Printf("%s\n", t)
			fmt.Printf("%s\n", current)
			fmt.Println("--")
		}

		// build the logic tree
		switch t.typ {
		case typeEOF, typeCloseParen:
			// we reached the end of the line and we have a dangling operator eg `t & f &`
			hasDangelingOperator := !isEmptyNode && (current.Type == typeAnd || current.Type == typeOr) && current.B == nil
			if hasDangelingOperator {
				return nil, ierrorf("bad '|' at %d", l.start)
			}
			return current, nil
		case typeSpace:
			// ignore all whitespace
			continue
		case typeError:
			// failed due to a lexing error
			return nil, ierrorf(t.val)
		case typeColon, typeComma:
			// we have bad function syntax, such as `t & : f,`
			return nil, ierrorf("bad '%s' at %d", t.val, l.start)
		case typeFunction:
			// check for bad function syntax, such as `t f & t`
			isOperator := !isEmptyNode && (current.Type == typeAnd || current.Type == typeOr)
			hasBadFunctionSyntax := !isEmptyNode && !isOperator
			if hasBadFunctionSyntax {
				return nil, ierrorf("bad '%s' at %d", t.val, l.start)
			}

			// parse the function and append it to the tree
			if n, err := parseFunction(l, t.val, rules); err != nil {
				return nil, err
			} else if isEmptyNode {
				current = n
			} else if current.A == nil {
				current.A = n
			} else if current.B == nil {
				current.B = n
			} else {
				return nil, ierrorf("bad '%s' at %d", t.val, l.start)
			}
		case typeAnd, typeOr:
			// check for bad operator syntax, such as `t & & f`
			isOperator := !isEmptyNode && (current.Type == typeAnd || current.Type == typeOr)
			isFull := !isEmptyNode && (current.A != nil && current.B != nil)
			hasBadOperatorSyntax := isOperator && !isFull
			if hasBadOperatorSyntax {
				return nil, ierrorf("bad '%s' at %d", t.val, l.start)
			}

			// append the operation to the tree
			var n node
			n.Type = t.typ
			n.A = current
			current = &n
		case typeOpenParen:
			// check for missing operator syntax such as `t (f | t)` or `(f & t) t`
			hasMissingOperator := !isEmptyNode && !(current.Type == typeAnd || current.Type == typeOr)
			if hasMissingOperator {
				return nil, ierrorf("bad '%s' at %d", t.val, l.start)
			}

			// recursively parse the function and append it to the tree
			if n, err := parseBools(l, rules); err != nil {
				return nil, err
			} else if isEmptyNode {
				current = n
			} else if current.A != nil && current.B == nil {
				current.B = n
			} else {
				return nil, ierrorf("bad '(' at %d", l.start)
			}
		default:
			return nil, ierrorf("bad '%s' at %d", t.val, l.start)
		}
	}
}

// parseFunction parses and returns a function node
func parseFunction(l *lexer, val string, rules map[string]Rule) (*node, error) {
	// parse a function node
	var n node
	if t := l.Next(); t.typ == typeColon {
		for {
			t = l.Next()
			if t.typ == typeSpace {
				continue
			}

			t = l.Next()
			if isParam := t.typ == typeBool || t.typ == typeNumber || t.typ == typeString; isParam {
				n.params = append(n.params, t.val)
			} else {
				return nil, ierrorf("'%s' is not a function parameter", t.val)
			}

			t = l.Next()
			if t.typ != typeComma {
				break
			}
		}
		l.Backup() // TODO: theres still a parse bug here somewhere
	}
	r, ok := rules[val]
	if !ok {
		return nil, ierrorf("'%s' is not a valid rule", val)
	}
	n.rule = r
	n.Type = typeFunction
	n.Value = val
	return &n, nil
}

type node struct {
	rule   Rule
	params []string
	Type   tokenType `json:"type"`
	Value  string    `json:"value,omitempty"`
	A      *node     `json:"a,omitempty"`
	B      *node     `json:"b,omitempty"`
}

func (n *node) execute(ps *RuleParams) error {
	// execute functions
	if n.Type == typeFunction {
		ps.Params = n.params
		return n.rule(ps)
	}

	// execute ands and ors
	err := n.A.execute(ps)
	if (err == nil && n.Type == typeAnd) || (err != nil && n.Type == typeOr) {
		return n.B.execute(ps)
	}
	return err
}

func (n *node) String() string {
	bs, err := json.MarshalIndent(n, "|", "	")
	if err != nil {
		panic(err)
	}
	return string(bs)
}
