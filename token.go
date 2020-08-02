package validate

import (
	"fmt"
)

// tokenType is the type of type emitted by the lexer
type tokenType int8

// MarshalText implements encoding.TextMarshaler
func (t *tokenType) MarshalText() ([]byte, error) {
	switch *t {
	case typeError:
		return []byte("typeError"), nil
	case typeEOF:
		return []byte("typeEOF"), nil
	case typeAnd:
		return []byte("typeAnd"), nil
	case typeOr:
		return []byte("typeOr"), nil
	case typeFunction:
		return []byte("typeFunction"), nil
	case typeColon:
		return []byte("typeColon"), nil
	case typeComma:
		return []byte("typeComma"), nil
	case typeOpenParen:
		return []byte("typeOpenParen"), nil
	case typeCloseParen:
		return []byte("typeCloseParen"), nil
	case typeBool:
		return []byte("typeBool"), nil
	case typeNumber:
		return []byte("typeNumber"), nil
	case typeString:
		return []byte("typeString"), nil
	case typeSpace:
		return []byte("typeSpace"), nil
	}
	return nil, fmt.Errorf("not a valid type")
}

func (t tokenType) String() string {
	bs, err := t.MarshalText()
	if err != nil {
		return err.Error()
	}
	return string(bs)
}

const (
	// typeError represents an error
	typeError = tokenType(iota)

	// typeEOF is the end of the file
	typeEOF

	// typeAnd is `&&`
	typeAnd

	// typeOr is `||`
	typeOr

	// typeFunction is a method signature
	typeFunction

	// typeColon is `:`
	typeColon

	// typeComma is `,`
	typeComma

	// typeOpenParen is `(`
	typeOpenParen

	// typeCloseParen is `)`
	typeCloseParen

	// typeBool is a boolean
	typeBool

	// typeNumber is a number
	typeNumber

	// typeString is a string surrounded by a `"` or `'`
	typeString

	// typeSpace is white space
	typeSpace
)

// type is a type emitted by the lexer
type token struct {
	typ tokenType
	val string
}

func (t token) String() string {
	switch t.typ {
	case typeEOF:
		return "EOF"
	case typeError:
		return t.val
	case typeAnd:
		return fmt.Sprintf("and: %s", t.val)
	case typeOr:
		return fmt.Sprintf("or: %s", t.val)
	case typeFunction:
		return fmt.Sprintf("function: %s", t.val)
	case typeColon:
		return fmt.Sprintf("colon: %s", t.val)
	case typeComma:
		return fmt.Sprintf("comma: %s", t.val)
	case typeOpenParen:
		return fmt.Sprintf("open paren: %s", t.val)
	case typeCloseParen:
		return fmt.Sprintf("close paren: %s", t.val)
	case typeBool:
		return fmt.Sprintf("bool: %s", t.val)
	case typeNumber:
		return fmt.Sprintf("number: %s", t.val)
	case typeString:
		return fmt.Sprintf("string: %s", t.val)
	case typeSpace:
		return fmt.Sprintf("space: %s", t.val)
	}
	if len(t.val) > 10 {
		return fmt.Sprintf("%.10s...", t.val)
	}
	return fmt.Sprintf("%s", t.val)
}
