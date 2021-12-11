package validator

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

// Errors contains a slice of errors
type Errors interface {
	Errors() []error
}

// FieldErrors are slice of FieldError generate by the rules
type FieldErrors []error

// Is implements errors.Is
func (es FieldErrors) Is(err error) bool {
	switch err.(type) {
	case FieldErrors:
		return true
	case *FieldErrors:
		return true
	}
	for _, e := range es {
		if errors.Is(e, err) {
			return true
		}
	}
	return false
}

// As implements errors.As
func (es FieldErrors) As(err interface{}) bool {
	if ptr, ok := err.(*FieldErrors); ok {
		*ptr = es
		return true
	} else if fes, ok := err.(FieldErrors); ok {
		for i := range fes {
			if len(es) == i {
				return true
			}
			fes[i] = es[i]
		}
		return true
	}
	for _, e := range es {
		if errors.As(e, err) {
			return true
		}
	}
	return false
}

// Error implements errors.Error
func (es FieldErrors) Error() string {
	bs, err := json.Marshal(es)
	if err != nil {
		return err.Error()
	}
	return string(bs)
}

// Errors implements Errors
func (es FieldErrors) Errors() []error {
	return es
}

// Add merges FieldErrors together
func (es *FieldErrors) Add(errs ...error) {
	for _, err := range errs {
		if errs, ok := err.(Errors); ok {
			es.Add(errs.Errors()...)
		}
		*es = append(*es, err)
	}
}

// FieldError is the error returned when a field rule returns an error
type FieldError struct {
	Path    string `json:"path,omitempty"`
	Message error  `json:"message,omitempty"`
}

// Is implements errors.Is
func (fe *FieldError) Is(err error) bool {
	if _, ok := err.(*FieldError); ok {
		return true
	}
	return errors.Is(fe.Message, err)
}

// Is implements errors.As
func (fe *FieldError) As(i interface{}) bool {
	if e, ok := i.(*FieldError); ok {
		e.Path = fe.Path
		e.Message = fe.Message
		return true
	}
	return errors.As(fe.Message, i)
}

// Error implements errors.Error
func (fe *FieldError) Error() string {
	return fe.Message.Error()
}

// MarshalJSON implements the json.Marshaler interface
func (fe *FieldError) MarshalJSON() ([]byte, error) {
	// TODO: after we have a clean `Path` for each error,
	//       add a config boolean that renders these as json objects instead
	return []byte(fmt.Sprintf("\"%s\"", fe.Message)), nil
}

// errorf handles i18n errors
func errorf(tag language.Tag, str string, is ...interface{}) error {
	return errors.New(message.NewPrinter(tag).Sprintf(str, is...))
}

// errorTemplate handles i18n template based errors
func errorTemplate(tag language.Tag, str string, context interface{}) error {
	str = message.NewPrinter(tag).Sprint(str)
	var bs bytes.Buffer
	if t, err := template.New(str).Funcs(template.FuncMap{
		"minus": func(a, b int) int {
			return a - b
		},
	}).Parse(str); err != nil {
		return err
	} else if err := t.Execute(&bs, context); err != nil {
		return err
	}
	return errors.New(bs.String())
}
