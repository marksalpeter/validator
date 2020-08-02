package validate

import (
	"fmt"
	"strings"
	"unicode"
)

var eof = rune(-1)

func lex(s string) <-chan *token {
	out := make(chan *token)
	go func() {
		defer close(out)
		l := newLexer(s)
		for {
			token := l.Next()
			out <- token
			if token.typ == typeError || token.typ == typeEOF {
				break
			}
		}
	}()
	return out
}

type lexer struct {
	buffer     string
	start      int
	pos        int
	len        int
	parenStack int
	debug      bool
}

func newLexer(s string) *lexer {
	return &lexer{
		buffer: s,
		len:    len(s),
		pos:    0,
	}
}

func (l *lexer) Peak() *token {
	t := l.Next()
	l.Backup()
	return t
}

func (l *lexer) Next() *token {
	l.start = l.pos
	if !l.hasNext() {
		if l.parenStack > 0 {
			err := fmt.Errorf("missing %d closed parenthasis at EOF", l.parenStack)
			l.backup()
			return l.emitError(err)
		}
		return l.emit(typeEOF)
	} else if isAnd := l.acceptPrefix("&"); isAnd {
		return l.emit(typeAnd)
	} else if isOr := l.acceptPrefix("|"); isOr {
		return l.emit(typeOr)
	} else if isColon := l.acceptPrefix(":"); isColon {
		return l.emit(typeColon)
	} else if isComma := l.acceptPrefix(","); isComma {
		return l.emit(typeComma)
	} else if isOpenParen := l.acceptPrefix("("); isOpenParen {
		l.parenStack++
		return l.emit(typeOpenParen)
	} else if isClosedParen := l.acceptPrefix(")"); isClosedParen {
		if l.parenStack == 0 {
			return l.emitError(fmt.Errorf("closed paren with no open paren at char %d near \"%.10s...\"", l.pos, l.buffer[l.pos:]))
		}
		l.parenStack--
		return l.emit(typeCloseParen)
	} else if isBool := l.acceptPrefix("true") || l.acceptPrefix("false"); isBool {
		return l.emit(typeBool)
	} else if isString, err := l.acceptString(); isString {
		return l.emit(typeString)
	} else if err != nil {
		return l.emitError(err)
	} else if isNumber := l.acceptNumber(); isNumber {
		return l.emit(typeNumber)
	} else if isWhiteSpace := l.acceptSpace(); isWhiteSpace {
		return l.emit(typeSpace)
	} else if err != nil {
		return l.emitError(err)
	} else if isFunction := l.acceptFunction(); isFunction {
		return l.emit(typeFunction)
	} else if err != nil {
		return l.emitError(err)
	}
	return l.emitError(fmt.Errorf("error at char %d near \"%.10s...\"", l.pos, l.buffer[l.pos:]))
}

func (l *lexer) Backup() {
	l.pos = l.start
}

func (l *lexer) emitError(err error) *token {
	return &token{typeError, err.Error()}
}

func (l *lexer) emit(t tokenType) *token {
	if l.debug {
		fmt.Printf("emit(%s) -> l.buffer[%d:%d] = %s\n", t, l.start, l.pos, l.buffer[l.start:l.pos])
	}
	tkn := token{
		t, l.buffer[l.start:l.pos],
	}
	return &tkn
}

func (l *lexer) hasNext() bool {
	return l.pos < l.len
}

func (l *lexer) next() rune {
	if !l.hasNext() {
		return eof
	}
	if l.debug {
		fmt.Printf("next[%d] = %s\n", l.pos, string(l.buffer[l.pos]))
	}
	r := rune(l.buffer[l.pos])
	l.pos++
	return r
}

func (l *lexer) peak() rune {
	r := l.next()
	l.backup()
	return r
}

func (l *lexer) backup() bool {
	if l.pos == 0 {
		return false
	}
	l.pos--
	if l.debug {
		fmt.Printf("backup() -> l.pos = %d\n", l.pos)
	}
	return true
}

// accept accepts one of the passed in characters in the set next
func (l *lexer) accept(valid string) bool {
	if l.hasNext() && strings.ContainsRune(valid, rune(l.buffer[l.pos])) {
		l.pos++
		if l.debug {
			fmt.Printf("accept(%s) -> l.pos = %d\n", valid, l.pos)
		}
		return true
	}
	return false
}

// acceptRun accepts one or more of the passed in characters in the set next
func (l *lexer) acceptRun(valid string) bool {
	var isAccepted bool
	for l.hasNext() && strings.ContainsRune(valid, rune(l.buffer[l.pos])) {
		l.pos++
		if l.debug {
			fmt.Printf("acceptRun(%s) -> l.pos = %d\n", valid, l.pos)
		}
		isAccepted = true
	}
	return isAccepted
}

// acceptPrefix accepts the entire valid string next
func (l *lexer) acceptPrefix(valid string) bool {
	if strings.HasPrefix(l.buffer[l.pos:], valid) {
		l.pos += len(valid)
		if l.debug {
			fmt.Printf("acceptPrefix(%s) -> l.pos = %d\n", valid, l.pos)
		}
		return true
	}
	return false
}

// acceptNumber scans a number (taken from the go standard librarys template lexer)
func (l *lexer) acceptNumber() bool {
	// Optional leading sign.
	l.accept("+-")
	// Is it hex?
	digits := "0123456789_"
	if l.accept("0") {
		// Note: Leading 0 does not mean octal in floatl.
		if l.accept("xX") {
			digits = "0123456789abcdefABCDEF_"
		} else if l.accept("oO") {
			digits = "01234567_"
		} else if l.accept("bB") {
			digits = "01_"
		}
	}
	l.acceptRun(digits)

	// ignore exponents +/- imaginary numbers and decimials that don't have a number component
	if hasNumbers := l.pos != l.start; !hasNumbers {
		return false
	}

	if l.accept(".") {
		l.acceptRun(digits)
	}
	if len(digits) == 10+1 && l.accept("eE") {
		l.accept("+-")
		l.acceptRun("0123456789_")
	}
	if len(digits) == 16+6+1 && l.accept("pP") {
		l.accept("+-")
		l.acceptRun("0123456789_")
	}
	// Is it imaginary?
	l.accept("i")

	// Next thing mustn't be alphanumeric
	if isAlphaNumeric(l.peak()) {
		l.pos = l.start
		return false
	}

	return l.pos != l.start
}

// acceptString accepts a string started by either a single or double quote
func (l *lexer) acceptString() (bool, error) {
	var isSingleQuote, isDoubleQuote bool
	if isSingleQuote = l.accept("'"); !isSingleQuote {
		if isDoubleQuote = l.accept("\""); !isDoubleQuote {
			return false, nil
		}
	}
	for {
		if isSingleQuote && l.accept("'") {
			return true, nil
		} else if isDoubleQuote && l.accept("\"") {
			return true, nil
		} else if l.next() == eof {
			break
		}
	}
	return false, fmt.Errorf("string not closed. char %d near \"%.10q...\"", l.pos, l.buffer[l.pos:])
}

// acceptSpace accepts all unicode spaces
func (l *lexer) acceptSpace() bool {
	for {
		if r := l.next(); !unicode.IsSpace(r) {
			if r != eof {
				l.backup()
			}
			break
		}
	}
	return l.start != l.pos
}

func (l *lexer) acceptFunction() bool {
	for {
		if r := l.next(); !isAlphaNumeric(r) {
			if r != eof {
				l.backup()
			}
			break
		}
	}
	return l.start != l.pos
}

// isAlphaNumeric reports whether r is an alphabetic, digit, or underscore.
func isAlphaNumeric(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}
