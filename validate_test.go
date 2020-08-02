package validator

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

const verboseLogs = false

func TestLexer(t *testing.T) {
	types := []tokenType{typeFunction, typeColon, typeSpace, typeString, typeComma, typeSpace, typeNumber, typeSpace, typeAnd, typeSpace, typeFunction, typeSpace, typeAnd, typeSpace, typeFunction, typeSpace, typeOr, typeSpace, typeFunction, typeSpace, typeAnd, typeSpace, typeFunction, typeSpace, typeOr, typeSpace, typeOpenParen, typeFunction, typeSpace, typeAnd, typeSpace, typeFunction, typeCloseParen, typeSpace, typeOr, typeSpace, typeFunction, typeColon, typeSpace, typeString, typeComma, typeSpace, typeNumber, typeSpace, typeAnd, typeSpace, typeFunction, typeColon, typeSpace, typeBool, typeEOF}
	values := []string{"func", ":", " ", "'param1'", ",", " ", "2", " ", "&", " ", "empty", " ", "&", " ", "i", " ", "|", " ", "one", " ", "&", " ", "two", "  ", "|", " ", "(", "three", " ", "&", " ", "four", ")", "	", "|", " ", "five", ":", " ", "'six'", ",", " ", "02.0", " ", "&", " ", "seven", ":", " ", "false", ""}
	l := newLexer("func: 'param1', 2 & empty & i | one & two  | (three & four)	| five: 'six', 02.0 & seven: false")
	l.debug = verboseLogs
	for i := 0; true; i++ {
		token := l.Next()
		if token.typ != types[i] {
			t.Log(token)
			t.Fatalf("token[%d].typ: '%s' != '%s'", i, token.typ, types[i])
			return
		} else if token.val != values[i] {
			t.Fatalf("token[%d].val: '%+v' != '%+v'", i, token.val, values[i])
			return
		}
		if token.typ == typeEOF {
			break
		}
	}

	for _, s := range []string{
		"func: param1, 2",
		"f & t",
		"t & (f | t | f)",
		"t & (f | f | t) & t",
	} {
		t.Run(s, func(t *testing.T) {
			l = newLexer(s)
			for token := l.Next(); token.typ != typeEOF; token = l.Next() {
				if token.typ == typeError {
					t.Fatal(token.val)
					break
				}
			}
		})
	}
}

func TestParser(t *testing.T) {
	parser := newParser()
	parser.debug = verboseLogs
	tr := func(ps *RuleParams) error {
		return nil
	}
	fl := func(ps *RuleParams) error {
		return fmt.Errorf("error called")
	}
	var params []string
	rules := map[string]Rule{
		"t": tr,
		"f": fl,
		"a": tr,
		"b": tr,
		"c": fl,
		"d": fl,
		"e": tr,
		"func": func(ps *RuleParams) error {
			params = ps.Params
			return nil
		},
	}

	// test function
	for _, s := range []string{
		"func: 'hello world', 2",
	} {
		if isValid := t.Run(s, func(t *testing.T) {
			if parsed, err := parser.parse(s, rules); err != nil {
				t.Fatal(err)
			} else if err := parsed.execute(&RuleParams{}); err != nil {
				t.Fatalf("execution failed: %s", err)
			} else {
				a := assert.New(t)
				a.Equal(params, []string{`'hello world'`, `2`})
			}
		}); !isValid {
			t.Fatal("failed")
			return
		}
	}

	// resolves to true
	for _, s := range []string{
		"t & t",
		"t & (f | t | f)",
		"a & (b | c | d) & e",
	} {
		if isValid := t.Run(s, func(t *testing.T) {
			if parsed, err := parser.parse(s, rules); err != nil {
				t.Fatalf("parse failed: %s", err)
			} else if err := parsed.execute(&RuleParams{}); err != nil {
				t.Fatalf("execution failed: %s", err)
			}
		}); !isValid {
			t.Fatal("failed")
			return
		}
	}

	// resolves to false
	for _, s := range []string{
		"t & f",
		"t & (f | t & f)",
		"t & (f | f & t) & t",
		"t & (f | f | t) & f",
	} {
		if isValid := t.Run(s, func(t *testing.T) {
			if parsed, err := parser.parse(s, rules); err != nil {
				t.Fatalf("parse failed: %s", err)
			} else if err := parsed.execute(&RuleParams{}); err == nil {
				t.Fatal("there should be an error returned")
			}
		}); !isValid {
			t.Fatal("failed")
			return
		}
	}

	// parse with bad bad syntax
	for _, s := range []string{
		"t f",
		"t (f | t & f)",
		"t & (f | f & t) t",
		"t & (f | f t) & f",
		"t & (f | f | t & f",
		"t & : f",
	} {
		if isValid := t.Run(s, func(t *testing.T) {
			if _, err := parser.parse(s, rules); err == nil {
				t.Fatal("should return a parse error")
			}
		}); !isValid {
			t.Fatal("failed")
			return
		}
	}
}

func TestValidator(t *testing.T) {
	debug = verboseLogs
	if pass := t.Run("test tag name parsing", func(t *testing.T) {
		// create a rule that always fails
		rules := Rules{
			"fail": func(*RuleParams) error {
				return errors.New("this will always fail")
			},
		}

		// create a validator with a default tag and a different tag with this rule
		v, v1 := New(&Config{
			Rules: rules,
		}), New(&Config{
			Tag:   "test",
			Rules: rules,
		})

		// create structs with a default tag and a test tag
		var s struct {
			Field string `validate:"fail"`
		}
		var s1 struct {
			Field string `test:"fail"`
		}

		// make sure that only the correct tag is parsed
		a := assert.New(t)
		a.NotNil(v.Validate(&s))
		a.Nil(v.Validate(&s1))
		a.Nil(v1.Validate(&s))
		a.NotNil(v1.Validate(&s1))
	}) && t.Run("test and / or logic", func(t *testing.T) {
		// create a rule that always passes and a rule that always fails fails
		rules := Rules{
			"pass": func(*RuleParams) error {
				return nil
			},
			"fail": func(*RuleParams) error {
				return errors.New("this will always fail")
			},
		}

		// create structs with a default tag and a test tag
		var s1 struct {
			Field string `validate:"fail | pass | fail & pass"`
		}
		var s2 struct {
			Field string `validate:"fail | fail | pass & fail"`
		}
		var s3 struct {
			Field string `validate:"pass | fail | fail & pass"`
		}
		var s4 struct {
			Field string `validate:"false | pass | false & pass & false | pass & false"`
		}

		// create a validator with a default tag and a different tag with this rule
		v := New(&Config{
			Rules: rules,
		})

		// make sure that only the correct tag is parsed
		a := assert.New(t)
		a.Nil(v.Validate(&s1))
		a.NotNil(v.Validate(&s2))
		a.Nil(v.Validate(&s3))
		a.NotNil(v.Validate(&s4))
	}) && t.Run("multiple errors are returned", func(t *testing.T) {
		// create a validator with a default tag and a different tag with this rule
		v := New(&Config{
			Rules: Rules{
				"fail": func(*RuleParams) error {
					return errors.New("fail")
				},
			},
		})
		var s1 struct {
			One   string `validate:"fail"`
			Two   string `validate:"fail"`
			Three string `validate:"fail"`
		}
		a := assert.New(t)
		a.EqualError(v.Validate(&s1), `["fail","fail","fail"]`)
	}) && t.Run("checks for bad syntax in the validate tag", func(t *testing.T) {
		type s struct {
			String string `json:"a" validate:"required & : empty"`
		}
		a := assert.New(t)
		v := New()
		if passed := a.EqualError(v.CheckSyntax(&s{}), `["bad ':' at 11"]`); !passed {
			t.FailNow()
		}
	}); !pass {
		t.Fatal("tests failed!")
	}
}

func TestRules(t *testing.T) {
	debug = verboseLogs
	if pass := t.Run("required", func(t *testing.T) {
		var s1 struct {
			Field string `validate:"required"`
		}
		s2 := struct {
			Field string `validate:"required"`
		}{
			Field: "populated",
		}
		v := New()
		a := assert.New(t)
		a.EqualError(v.Validate(&s1), `["'Field' is required"]`)
		a.Nil(v.Validate(&s2))
	}) && t.Run("empty", func(t *testing.T) {
		var s1 struct {
			Field string `validate:"empty | fail"`
		}
		s2 := struct {
			Field string `validate:"empty | fail"`
		}{
			Field: "populated",
		}
		v := New(&Config{
			Rules: Rules{
				"empty": Empty,
				"fail": func(*RuleParams) error {
					return errors.New("this will always fail")
				},
			},
		})
		a := assert.New(t)
		a.Nil(v.Validate(&s1))
		a.EqualError(v.Validate(&s2), `["this will always fail"]`)
	}) && t.Run("email", func(t *testing.T) {
		var s1 struct {
			EmailAddress string `validate:"email"`
		}
		var s2 struct {
			EmailAddress uint `validate:"email"`
		}
		v := New(&Config{
			Rules: Rules{
				"email": Email,
			},
		})
		a := assert.New(t)

		// empty emails fail
		a.EqualError(v.Validate(&s1), `["'EmailAddress' must be a valid email address"]`)

		// incorrect emails fail
		s1.EmailAddress = "notAnEmail@"
		a.EqualError(v.Validate(&s1), `["'EmailAddress' must be a valid email address"]`)

		// empty emails fail
		s1.EmailAddress = "hello@dealyze.com"
		a.Nil(v.Validate(&s1))

		// syntax check
		a.EqualError(v.CheckSyntax(&s2), "the email tag must be applied to a string")
	}) && t.Run("password", func(t *testing.T) {
		var s1 struct {
			Password string `validate:"password"`
		}
		var s2 struct {
			Password []byte `validate:"password"`
		}
		v := New(&Config{
			Rules: Rules{
				"password": Password,
			},
		})
		a := assert.New(t)

		// empty emails fail
		a.EqualError(v.Validate(&s1), `["'Password' must be a at least 6 characters long and contain at least one number or special character (eg. @!#)"]`)

		// password without special characters fails
		s1.Password = "notavalidpassword"
		a.EqualError(v.Validate(&s1), `["'Password' must be a at least 6 characters long and contain at least one number or special character (eg. @!#)"]`)

		// password that is too short fails
		s1.Password = "abc12"
		a.EqualError(v.Validate(&s1), `["'Password' must be a at least 6 characters long and contain at least one number or special character (eg. @!#)"]`)

		// valid password succeeds
		s1.Password = "abc123"
		a.Nil(v.Validate(&s1))

		// syntax check
		a.EqualError(v.CheckSyntax(&s2), "the password tag must be applied to a string")
	}) && t.Run("number", func(t *testing.T) {
		var s1 struct {
			Number string `validate:"number"`
		}
		var s2 struct {
			Number string `validate:"number:2,4"`
		}
		var s3 struct {
			Number int `validate:"number:2,4"`
		}
		var s4 struct {
			Int     int     `validate:"number"`
			Int8    int8    `validate:"number"`
			Int16   int16   `validate:"number"`
			Int32   int32   `validate:"number"`
			Int64   int64   `validate:"number"`
			Uint    uint    `validate:"number"`
			Uint8   uint8   `validate:"number"`
			Uint16  uint16  `validate:"number"`
			Uint32  uint32  `validate:"number"`
			Uint64  uint64  `validate:"number"`
			Float32 float32 `validate:"number"`
			Float64 float64 `validate:"number"`
		}
		v := New(&Config{
			Rules: Rules{
				"number": Number,
			},
		})
		a := assert.New(t)

		// empty strings fail
		a.NotNil(v.Validate(&s1))

		// incorrect string number fails
		s1.Number = "not a number"
		a.NotNil(v.Validate(&s1))

		// strings that is only numbers works
		s1.Number = "12345"
		a.Nil(v.Validate(&s1))

		// digit range
		s2.Number = "0"
		a.EqualError(v.Validate(&s2), `["'Number' must be 2 to 4 digits"]`)
		s2.Number = "000"
		a.Nil(v.Validate(&s2))
		s2.Number = "00000"
		a.EqualError(v.Validate(&s2), `["'Number' must be 2 to 4 digits"]`)

		// int range
		s3.Number = 1
		a.EqualError(v.Validate(&s3), `["'Number' must be 2 to 4"]`)
		s3.Number = 3
		a.Nil(v.Validate(&s3))
		s3.Number = 5
		a.EqualError(v.Validate(&s3), `["'Number' must be 2 to 4"]`)

		// all numbers are valid
		a.Nil(v.Validate(&s4))

	}) && t.Run("eq", func(t *testing.T) {
		type s struct {
			Uint   uint   `json:"a" validate:"eq:1,2,3"`
			Int    int    `json:"b" validate:"eq:1,2,3"`
			String string `json:"c" validate:"eq:1,2,3"`
		}
		s1 := s{
			1, 2, "3",
		}
		var s2 s
		var s3 struct {
			Uint uint `json:"a" validate:"eq"`
		}
		v := New()
		a := assert.New(t)
		a.Nil(v.Validate(&s1))
		a.EqualError(v.Validate(&s2), `["'a' must equal '1', '2' or '3'","'b' must equal '1', '2' or '3'","'c' must equal '1', '2' or '3'"]`)
		a.EqualError(v.CheckSyntax(&s3), "eq requires at least one parameter")
	}) && t.Run("xor", func(t *testing.T) {
		type s struct {
			Uint   uint   `json:"a" validate:"xor:Int,String"`
			Int    int    `json:"b"`
			String string `json:"c"`
		}
		s1 := s{
			Uint: 1,
		}
		s2 := s{
			Int: 1,
		}
		s3 := s{
			String: "1",
		}
		s4 := s{
			Uint:   1,
			Int:    1,
			String: "1",
		}
		s5 := s{}
		var s6 struct {
			Uint uint `json:"a" validate:"xor:Int,String"`
		}
		v := New()
		a := assert.New(t)
		a.Nil(v.Validate(&s1))
		a.Nil(v.Validate(&s2))
		a.Nil(v.Validate(&s3))
		a.EqualError(v.Validate(&s4), `["either 'a', 'b' or 'c' must be set"]`)
		a.EqualError(v.Validate(&s5), `["either 'a', 'b' or 'c' must be set"]`)
		a.EqualError(v.CheckSyntax(&s6), "'.Int' is not a valid field")
	}) && t.Run("or", func(t *testing.T) {
		type s struct {
			Uint   uint   `json:"a" validate:"or:Int,String"`
			Int    int    `json:"b"`
			String string `json:"c"`
		}
		s1 := s{
			Uint: 1,
		}
		s2 := s{
			Int: 1,
		}
		s3 := s{
			String: "1",
		}
		s4 := s{
			Uint:   1,
			Int:    1,
			String: "1",
		}
		s5 := s{}
		var s6 struct {
			Uint uint `json:"a" validate:"or:Int,String"`
		}
		v := New()
		a := assert.New(t)
		a.Nil(v.Validate(&s1))
		a.Nil(v.Validate(&s2))
		a.Nil(v.Validate(&s3))
		a.Nil(v.Validate(&s4))
		a.EqualError(v.Validate(&s5), `["either 'a', 'b' and/or 'c' must be set"]`)
		a.EqualError(v.CheckSyntax(&s6), "'.Int' is not a valid field")
	}) && t.Run("and", func(t *testing.T) {
		type s struct {
			Uint   uint   `json:"a" validate:"and:Int,String"`
			Int    int    `json:"b"`
			String string `json:"c"`
		}
		s1 := s{
			Uint: 1,
		}
		s2 := s{
			Int: 1,
		}
		s3 := s{
			String: "1",
		}
		s4 := s{
			Uint:   1,
			Int:    1,
			String: "1",
		}
		s5 := s{}
		var s6 struct {
			Uint uint `json:"a" validate:"or:Int,String"`
		}
		v := New()
		a := assert.New(t)
		a.EqualError(v.Validate(&s1), `["'a', 'b' and 'c' must be set"]`)
		a.EqualError(v.Validate(&s2), `["'a', 'b' and 'c' must be set"]`)
		a.EqualError(v.Validate(&s3), `["'a', 'b' and 'c' must be set"]`)
		a.Nil(v.Validate(&s4))
		a.EqualError(v.Validate(&s5), `["'a', 'b' and 'c' must be set"]`)
		a.EqualError(v.CheckSyntax(&s6), "'.Int' is not a valid field")
	}); !pass {
		t.Fatal("error")
	}
}
