package validator

import (
	"encoding"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/text/language"
)

// Rules are a set of rules that the `Validator` will look up by name in order to appy them to fields in a struct
type Rules map[string]Rule

// Rule is a rule that is applied to a field in a struct
type Rule func(*RuleParams) error

// RuleParams is the set of parameters a rule processes to determine if there was a validation error
type RuleParams struct {
	// Tag represents the language the error message should be in
	Tag language.Tag

	// FieldName is the name of the field the rule is validating
	// TODO: add example
	FieldName string

	// Params are arguments that were passed to the rule
	// TODO: add example
	Params []string

	// Root is the interface{} that was passed to the Validator.Validate method
	Root reflect.Value

	// Parent is the struct{} that the Field belongs to. This can be the same as Root if a simple struct was passed in to the Validator.Validate func
	Parent reflect.Value

	// Field is the field on the struct whose value is being validated
	Field reflect.Value
}

// DefaultRules is the default set of rules the validator will be created with
var DefaultRules = Rules{
	"required": Required,
	"empty":    Empty,
	"name":     Name,
	"email":    Email,
	"password": Password,
	"number":   Number,
	"letters":  Letters,
	"eq":       EQ,
	"xor":      XOR,
	"or":       OR,
	"and":      AND,
	// TODO: create and add neq, lt, gt, lte, and gte
}

// Required returns an error if the filed contains the zero value of the type or nil.
//
// Example
//  type Struct struct {
//    Field  string `json:"field" validate:"required"` // 'field' is required
//  }
//
func Required(ps *RuleParams) error {
	field, tag, fieldName := ps.Field, ps.Tag, ps.FieldName
	if hasValue(field) {
		return nil
	}
	return errorf(tag, "'%s' is required", fieldName)
}

// Empty returns an error if the field is not empty. It should be 'or'd together with
// other rules that require manditory input
//
// Example
//  type Struct struct {
//    Field  string `json:"field" validate:"empty | email"` // 'field' must be a valid email address or not set at all
//  }
//
func Empty(ps *RuleParams) error {
	field, tag, fieldName := ps.Field, ps.Tag, ps.FieldName
	if !hasValue(field) {
		return nil
	}
	return errorf(tag, "'%s' should position omitempty before other tags", fieldName)
}

// Name returns an error if the field doesn't contain a valid name
// I.e. no numbers or most special characters, excepting characters that may be in a name like a -
// and allowing foreign language letters with accent marks as well as spaces
// This prevents things like emails or phone numbers from being entered as a name.
//
// Example
//  type Struct struct {
//    Field  string `json:"field" validate:"name"` // 'field' must be a valid name
//  }
//
func Name(ps *RuleParams) error {
	if ps.Field.Kind() != reflect.String {
		panic("the name tag must be applied to a string")
	}

	if isValid, _ := regexp.Match(`^[^0-9_!¡?÷?¿/\\+=@#$%ˆ&*(){}|~<>;:[\]]{2,}$`, []byte(ps.Field.String())); isValid {
		return nil
	}
	if len(ps.Params) > 0 {
		return fmt.Errorf("%+v", ps.Params[0])
	}
	return errorf(ps.Tag, "'%s' must be a valid name", ps.FieldName)
}

// Email returns an error if the field doesn't contain a valid email address
//
// Example
//  type Struct struct {
//    Field  string `json:"field" validate:"email"` // 'field' must be a valid email address
//  }
//
func Email(ps *RuleParams) error {
	if ps.Field.Kind() != reflect.String {
		panic("the email tag must be applied to a string")
	}
	if isValid, _ := regexp.Match(`^(([^<>()[\]\\.,;:\s@"]+(\.[^<>()[\]\\.,;:\s@"]+)*)|(".+"))@((\[[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}])|(([a-zA-Z\-0-9]+\.)+[a-zA-Z]{2,}))$`, []byte(ps.Field.String())); isValid {
		return nil
	}
	return errorf(ps.Tag, "'%s' must be a valid email address", ps.FieldName)
}

// Password returns an error if the field doesn't contain a valid password
// Example
//  type Struct struct {
//    Field  string `json:"field" validate:"password"` // 'field' must be a valid password
//  }
//
func Password(ps *RuleParams) error {
	if ps.Field.Kind() != reflect.String {
		panic("the password tag must be applied to a string")
	}
	field := ps.Field.String()
	isLongEnough := len(field) >= 6
	hasSpecialCharacters, _ := regexp.Match(`[^a-zA-Z]+`, []byte(field))
	if isLongEnough && hasSpecialCharacters {
		return nil
	}
	return errorf(ps.Tag, "'%s' must be a at least 6 characters long and contain at least one number or special character (eg. @!#)", ps.FieldName)
}

// Number retuns an error if the field doesn't contain numbers only
//
// Example
//  type Struct struct {
//    Field   string `json:"field" validate:"number"`      // 'field' must contain only numbers
//    Field2  string `json:"field2" validate:"number:3,5"` // 'field2' must be 3 to 5 digits
//    Field3  uint   `json:"field3" validate:"number:3,5"` // 'field3' must be 3 to 5
//  }
//
func Number(ps *RuleParams) error {
	var min, max, i int
	var isMinSet, isMaxSet bool
	params, field, tag, fieldName := ps.Params, ps.Field, ps.Tag, ps.FieldName

	// parse min params
	if len(params) > 0 && len(params[0]) > 0 {
		var err error
		min, err = strconv.Atoi(params[0])
		isMinSet = err == nil
	}

	// parse max params
	if len(params) > 1 && len(params[1]) > 0 {
		var err error
		max, err = strconv.Atoi(params[1])
		isMaxSet = err == nil
	}

	switch field.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i = int(field.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		i = int(field.Uint())
	case reflect.Float32, reflect.Float64:
		i = int(field.Float())
	case reflect.String:
		str := field.String()
		if isValid, _ := regexp.Match("^[0-9]+$", []byte(str)); !isValid {
			return errorf(tag, "'%s' must contain only numbers", fieldName)
		} else if i := len(str); (!isMinSet || i >= min) && (!isMaxSet || i <= max) {
			return nil
		} else if isMaxSet && isMinSet {
			return errorf(tag, "'%s' must be %d to %d digits", fieldName, min, max)
		} else if isMaxSet {
			return errorf(tag, "'%s' must have %d or fewer digits", fieldName, max)
		} else if isMinSet {
			return errorf(tag, "'%s' must have %d or more digits", fieldName, min)
		}
	}

	if (!isMinSet || i >= min) && (!isMaxSet || i <= max) {
		return nil
	} else if isMaxSet && isMinSet {
		return errorf(tag, "'%s' must be %d to %d", fieldName, min, max)
	} else if isMaxSet {
		return errorf(tag, "'%s' must be %d or less", fieldName, max)
	} else if isMinSet {
		return errorf(tag, "'%s' must be %d or more", fieldName, min)
	}

	return nil
}

// Letters retuns an error if the field doesn't contain letters only
//
// Example
//  type Struct struct {
//    Field  string `json:"field" validate:"letters"` // 'field' can only take letters and spaces
//  }
//
func Letters(ps *RuleParams) error {
	field, tag, fieldName := ps.Field, ps.Tag, ps.FieldName
	if field.Kind() == reflect.String {
		if isLetters, _ := regexp.Match("^[A-Za-z ]+$", []byte(field.String())); isLetters {
			return nil
		}
	}
	return errorf(tag, "'%s' can only contain letters and spaces", fieldName)
}

// EQ returns an error if the field does not == one of the params passed in
//
// Example
//  type Struct struct {
//    Field  string `json:"field" validate:"eq:one,two,three"` // 'field' must equal either "one", "two", or "three"
//  }
//
func EQ(ps *RuleParams) error {
	params, field, tag, fieldName := ps.Params, ps.Field, ps.Tag, ps.FieldName
	psLen := len(params)
	if psLen == 0 {
		panic(fmt.Errorf("eq requires at least one parameter"))
	}

	// parse the params to match the kind of field and compare for equality
	switch field.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		for _, p := range params {
			if i, err := strconv.ParseInt(p, 10, 0); err == nil && field.Int() == i {
				return nil
			}
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		for _, p := range params {
			if i, err := strconv.ParseUint(p, 10, 0); err == nil && field.Uint() == i {
				return nil
			}
		}
	case reflect.Float32, reflect.Float64:
		for _, p := range params {
			i, err := strconv.ParseFloat(p, 64)
			if err == nil && field.Float() == i {
				return nil
			}
		}
	case reflect.String:
		for _, p := range params {
			if p == field.String() {
				return nil
			}
		}
	}

	// if the field implements encoding.TextMarshaler, check the text value of the field as well
	if marshaler, ok := field.Interface().(encoding.TextMarshaler); ok {
		if text, err := marshaler.MarshalText(); err == nil {
			for _, p := range params {
				if p == string(text) {
					return nil
				}
			}
		}
	}

	// construct the error message
	context := []string{fieldName}
	context = append(context, params...)
	return errorTemplate(tag, `{{$len := len .}}{{$last := minus $len 1}}{{range $i, $field := .}}{{if eq $i 1}} must equal {{else if eq $i $last}} or {{else if gt $i 0}}, {{end}}'{{$field}}'{{end}}`, context)
}

// XOR returns an error when more than one or zero of either the field that it is applied to or any of the field names passed as params are set to a non zero value
//
// Example
//  type Struct struct {
//    Field  string `json:"field" validate:" xor:Field2"` // either "field" or "Field2" must be set
//    Field2 string
//  }
//
func XOR(ps *RuleParams) error {
	params, parent, field, tag, fieldName := ps.Params, ps.Parent, ps.Field, ps.Tag, ps.FieldName
	fieldNames := []string{fieldName}
	pType := parent.Type()
	var populated int
	if hasValue(field) {
		populated++
	}
	for _, param := range params {
		fField, ok := pType.FieldByName(param)
		fValue := parent.FieldByName(param)
		if !ok || !fValue.IsValid() {
			panic(fmt.Errorf("'%s.%s' is not a valid field", parent.Type().Name(), param))
		}

		// count every field thatis populated
		if hasValue(fValue) {
			populated++
		}

		// write the json names of the other fields into the potential error message context
		if fieldName, ok := fField.Tag.Lookup("json"); ok {
			fieldNames = append(fieldNames, strings.Split(fieldName, ",")[0])
		} else {
			fieldNames = append(fieldNames, fField.Name)
		}
	}
	if populated == 1 {
		return nil
	}

	return errorTemplate(tag, `{{$len := len .}}{{$last := minus $len 1}}{{range $i, $field := .}}{{if eq $i 0}}either {{else if eq $i $last}} or {{else}}, {{end}}'{{$field}}'{{end}} must be set`, fieldNames)
}

// OR returns an error when neither the field that it is applied to nor any of the field names passed as params are set to a non zero value
//
// Example
//  type Struct struct {
//    Field  string `json:"field" validate:"or:Field2"` // either "field" or "Field2" or both must be set
//    Field2 string
//  }
//
func OR(ps *RuleParams) error {
	params, parent, field, tag, fieldName := ps.Params, ps.Parent, ps.Field, ps.Tag, ps.FieldName

	pType := parent.Type()
	if hasValue(field) {
		return nil
	}
	fieldNames := []string{fieldName}
	for _, param := range params {
		fField, ok := pType.FieldByName(param)
		fValue := parent.FieldByName(param)
		if !ok || !fValue.IsValid() {
			panic(fmt.Errorf("'%s.%s' is not a valid field", parent.Type().Name(), param))
		}
		if hasValue(fValue) {
			return nil
		}

		// write the json names of the other fields into the potential error message
		if fieldName, ok := fField.Tag.Lookup("json"); ok {
			fieldNames = append(fieldNames, strings.Split(fieldName, ",")[0])
		} else {
			fieldNames = append(fieldNames, fField.Name)
		}
	}

	return errorTemplate(tag, `{{$len := len .}}{{$last := minus $len 1}}{{range $i, $field := .}}{{if eq $i 0}}either {{else if eq $i $last}} and/or {{else}}, {{end}}'{{$field}}'{{end}} must be set`, fieldNames)
}

// AND returns an error when the field that it is applied to or any of the field names passed as params are set to the zero value
//
// Example
//  type Struct struct {
//    Field  string `json:"field" validate:"and:Field2"` // "field" and "Field2" must be set
//    Field2 string
//  }
//
func AND(ps *RuleParams) error {
	params, parent, field, tag, fieldName := ps.Params, ps.Parent, ps.Field, ps.Tag, ps.FieldName
	fieldNames := []string{fieldName}
	pType := parent.Type()
	isPopulated := hasValue(field)
	for _, param := range params {
		fField, ok := pType.FieldByName(param)
		fValue := parent.FieldByName(param)
		if !ok || !fValue.IsValid() {
			panic(fmt.Errorf("'%s.%s' is not a valid field", parent.Type().Name(), param))
		}
		isPopulated = isPopulated && hasValue(fValue)

		// write the json names of the other fields into the potential error message
		if fieldName, ok := fField.Tag.Lookup("json"); ok {
			fieldNames = append(fieldNames, strings.Split(fieldName, ",")[0])
		} else {
			fieldNames = append(fieldNames, fField.Name)
		}
	}
	if isPopulated {
		return nil
	}
	return errorTemplate(tag, `{{$len := len .}}{{$last := minus $len 1}}{{range $i, $field := .}}{{if eq $i $last}} and {{else if gt $i 0}}, {{end}}'{{$field}}'{{end}} must be set`, fieldNames)
}

// hasValue returns if the field is not nil or the golang devault/zero value
func hasValue(field reflect.Value) bool {
	fieldType := field.Type()
	fieldKind := fieldType.Kind()
	switch fieldKind {
	case reflect.Slice, reflect.Map, reflect.Ptr, reflect.Interface, reflect.Chan, reflect.Func:
		return !field.IsNil()
	default:
		return field.IsValid() && !reflect.DeepEqual(field.Interface(), reflect.Zero(fieldType).Interface())
	}
}
