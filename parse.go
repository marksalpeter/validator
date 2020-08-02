package validator

import (
	"encoding/json"
	"fmt"
	"runtime"
	"strings"
)

type parser struct {
	debug bool
	cache map[string]*node
}

func newParser() *parser {
	return &parser{
		cache: make(map[string]*node),
	}
}

func (p *parser) parse(validator string, rules map[string]Rule) (*node, error) {
	// get the cached version
	if parsed, ok := p.cache[validator]; ok {
		return parsed, nil
	}

	// parse new validators
	l := newLexer(validator)
	l.debug = p.debug
	if p.debug {
		fmt.Println("***")
		fmt.Println(validator)
		defer fmt.Println("***")
	}
	parsed, err := p.parseBools(l, rules)
	if err != nil {
		return nil, err
	}

	// cache the parsed value and return
	p.cache[validator] = parsed
	return parsed, nil
}

func (p *parser) parseBools(l *lexer, rules map[string]Rule) (*node, error) {
	var current *node
	for {
		t := l.Next()
		isEmptyNode := current == nil

		if p.debug {
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
				return nil, p.errorf("bad '|' at %d", l.start)
			}
			return current, nil
		case typeSpace:
			// ignore all whitespace
			continue
		case typeError:
			// failed due to a lexing error
			return nil, p.errorf(t.val)
		case typeColon, typeComma:
			// we have bad function syntax, such as `t & : f,`
			return nil, p.errorf("bad '%s' at %d", t.val, l.start)
		case typeFunction:
			// check for bad function syntax, such as `t f & t`
			isOperator := !isEmptyNode && (current.Type == typeAnd || current.Type == typeOr)
			hasBadFunctionSyntax := !isEmptyNode && !isOperator
			if hasBadFunctionSyntax {
				return nil, p.errorf("bad '%s' at %d", t.val, l.start)
			}

			// parse the function and append it to the tree
			if n, err := p.parseFunction(l, t.val, rules); err != nil {
				return nil, err
			} else if isEmptyNode {
				current = n
			} else if current.A == nil {
				current.A = n
			} else if current.B == nil {
				current.B = n
			} else {
				return nil, p.errorf("bad '%s' at %d", t.val, l.start)
			}
		case typeAnd, typeOr:
			// check for bad operator syntax, such as `t & & f`
			isOperator := !isEmptyNode && (current.Type == typeAnd || current.Type == typeOr)
			isFull := !isEmptyNode && (current.A != nil && current.B != nil)
			hasBadOperatorSyntax := isOperator && !isFull
			if hasBadOperatorSyntax {
				return nil, p.errorf("bad '%s' at %d", t.val, l.start)
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
				return nil, p.errorf("bad '%s' at %d", t.val, l.start)
			}

			// recursively parse the function and append it to the tree
			if n, err := p.parseBools(l, rules); err != nil {
				return nil, err
			} else if isEmptyNode {
				current = n
			} else if current.A != nil && current.B == nil {
				current.B = n
			} else {
				return nil, p.errorf("bad '(' at %d", l.start)
			}
		default:
			return nil, p.errorf("bad '%s' at %d", t.val, l.start)
		}
	}
}

// parseFunction parses and returns a function node
func (p *parser) parseFunction(l *lexer, val string, rules map[string]Rule) (*node, error) {
	var n node
	r, ok := rules[val]
	if !ok {
		return nil, p.errorf("'%s' is not a valid rule", val)
	}
	n.Rule = r
	n.Type = typeFunction
	n.Value = val
	needsParam := false
	parse := true
	for parse {
		t := l.Next()
		if p.debug {
			fmt.Printf("%s\n", t)
		}
		switch t.typ {
		case typeColon, typeComma:
			needsParam = true
		case typeBool, typeNumber, typeString, typeFunction: /* note: adding `typeFunction` interprets non-quoted strings as string params if possible */
			if !needsParam {
				return nil, p.errorf("bad '%s' at %d", t.val, l.start)
			}
			n.Params = append(n.Params, t.val)
			needsParam = false
		case typeSpace:
			continue
		default:
			l.Backup()
			return &n, nil
		}
	}

	return &n, nil
}

// errorf formats the internal error messages related to parsing and executing within the framework
func (p *parser) errorf(v string, is ...interface{}) error {
	var tag string
	if p.debug {
		_, file, line, _ := runtime.Caller(1)
		pieces := strings.Split(file, "/")
		tag = fmt.Sprintf("%s:%d: ", pieces[len(pieces)-1], line)
	}
	return fmt.Errorf(tag+v, is...)
}

type node struct {
	Rule   Rule      `json:"-"`
	Params []string  `json:"params,omitempty"`
	Type   tokenType `json:"type"`
	Value  string    `json:"value,omitempty"`
	A      *node     `json:"a,omitempty"`
	B      *node     `json:"b,omitempty"`
}

func (n *node) execute(ps *RuleParams) error {
	// execute functions
	if n.Type == typeFunction {
		ps.Params = n.Params
		return n.Rule(ps)
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
