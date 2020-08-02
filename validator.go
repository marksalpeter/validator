// Package validate validates the fields of structs by applying the rules embeded in a fields "validate" tag to the value of that field.
// It is designed to return plain english error messages that refer to field names by their json key.
// These validation error messages are intended to be presented to the outside world.
//
// Rule Syntax
//
// Rules can be joined together with "and"s (&) and "or"s (|)
//
//  type Struct struct {
//    Field   string `json:"field" validate:"omitempty | email"`   // 'field' must be a valid email address or not set at all
//    Field2  string `json:"field2" validate:"required & letters"` // 'field' is required and must be comprised of only letters and spaces
//  }
//
// Comma seperated params can also be passed to a rule, but not every rule has parameters. Check the godoc of the spefic rule
// you're applying for an example of how to use it.
//
//  type Struct struct {
//    Field  string `json:"field" validate:"eq:one,two,three"` // 'field' must equal either "one", "two", or "three"
//  }
//
// Finally, its worth noting the validators can cross reference other fields.
//
//  type Struct struct {
//    Field  string `json:"field" validate:" xor:Field2"` // either "field" or "field2" must be set
//    Field2 string `json:"field2"`
//  }
//
//
package validate

import (
	"fmt"
	"reflect"
	"strings"

	"golang.org/x/text/language"
)

// debug can be set by the test suite to get full verbose logging from the parser
var debug bool

// DefaultTag is the tage used if Config.Tag is not set
const DefaultTag = "validate"

// Validator validates structs and slices
type Validator interface {
	// CheckSyntax cycles though all of the validation tags and returns bad syntax errors instead of panicing
	CheckSyntax(interface{}) error

	// Validate validates a struct or a slice based on the information passed to the 'validate' tag.
	// The error returned will be in English by default, but thay can be changed to Spanish by setting the optional language.Tag.
	Validate(interface{}, ...language.Tag) error
}

// Config configures the validator
type Config struct {
	Tag   string
	Rules Rules
}

// New returns a new Validator
// It will parse the validation tags in the following fashion:
//
// Example
//
//   type Example struct {
// 	  Field string `validator:"one | (two & three)"`
//   }
//   New().Validate(&Example{})
//
// The field will be deemed valid if
//   one(Example.Field) == nil || (two(Example.Field) == nil && three(Example.Field) == nil)
//
func New(cfg ...*Config) Validator {
	var v validator
	v.tag = DefaultTag
	v.rules = DefaultRules
	v.parser = newParser()
	v.parser.debug = debug
	if cfg == nil || len(cfg) == 0 {
		return &v
	}
	if len(cfg[0].Tag) > 0 {
		v.tag = cfg[0].Tag
	}
	if cfg[0].Rules != nil && len(cfg[0].Rules) > 0 {
		v.rules = cfg[0].Rules
	}
	return &v
}

type validator struct {
	tag    string
	rules  Rules
	parser *parser
}

// Validate returns an implementation of Validate
func (v *validator) Validate(i interface{}, tags ...language.Tag) error {
	iValue := reflect.ValueOf(i)
	tag := language.English
	if len(tags) > 0 {
		tag = tags[0]
	}
	if errs := v.traverse(tag, false, iValue, iValue); len(errs) > 0 {
		return errs
	}
	return nil
}

// traverse walks slices, arrays, and struct searching for validation tags
func (v *validator) traverse(tag language.Tag, isSyntaxCheck bool, iRoot, iValue reflect.Value) FieldErrors {
	var errs FieldErrors
	iType := iValue.Type()
	iKind := iType.Kind()

	// dereference pointers
	if iKind == reflect.Ptr {
		iValue = iValue.Elem()
		iType = iValue.Type()
		iKind = iType.Kind()
	}

	// traverse slices and arrays
	if iKind == reflect.Slice || iKind == reflect.Array {
		for i, l := 0, iValue.Len(); i < l; i++ {
			if es := v.traverse(tag, isSyntaxCheck, iRoot, iValue.Index(i)); len(es) > 0 {
				errs.Add(es...)
			}
		}
	}

	// traverse fields in a struct and validate
	if iKind == reflect.Struct {
		for i, l := 0, iType.NumField(); i < l; i++ {
			field := iType.Field(i)
			fValue := iValue.Field(i)
			fType := fValue.Type()
			fKind := fType.Kind()

			// dereference pointers
			if fKind == reflect.Ptr && !fValue.IsNil() {
				fValue = fValue.Elem()
				fType = fValue.Type()
				fKind = fType.Kind()
			}

			// validate a field with the validation tag
			if validator, ok := field.Tag.Lookup(v.tag); ok {
				fieldName, ok := field.Tag.Lookup("json")
				if ok {
					fieldName = strings.Split(fieldName, ",")[0]
				} else {
					fieldName = field.Name
				}

				// create params
				var ps RuleParams
				ps.Root = iRoot
				ps.Parent = iValue
				ps.Field = fValue
				ps.FieldName = fieldName
				ps.Tag = tag

				// get the parse tree
				if parsed, err := v.parser.parse(validator, v.rules); err != nil {
					errs.Add(&FieldError{
						Message: err.Error(),
					})
				} else if err := parsed.execute(&ps); err != nil {
					if !isSyntaxCheck {
						errs.Add(&FieldError{
							Message: err.Error(),
						})
					}
				}

			}

			// traverse the field if possible
			if fKind == reflect.Struct || fKind == reflect.Array || fKind == reflect.Slice {
				if es := v.traverse(tag, isSyntaxCheck, iRoot, fValue); len(es) > 0 {
					errs.Add(es...)
				}
			}
		}
	}
	return errs
}

func (v *validator) CheckSyntax(i interface{}) error {
	out := make(chan error)
	go func() {
		defer close(out)
		defer func() {
			if err := recover(); err != nil {
				out <- fmt.Errorf("%+v", err)
			}
		}()
		iValue := reflect.ValueOf(i)
		out <- v.traverse(language.English, true, iValue, iValue)
	}()
	return <-out
}
