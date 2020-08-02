package validate

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"runtime"
	"strings"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

// Errors contains a slice of errors
type Errors interface {
	Errors() []error
}

// FieldErrors implemente Errors
type FieldErrors []error

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
	Message string `json:"message,omitempty"`
}

// Error implements errors.Error
func (fe *FieldError) Error() string {
	return fe.Message
}

// MarshalJSON implments the json.Marshaler interface
func (fe *FieldError) MarshalJSON() ([]byte, error) {
	// TODO: after we have a clean `Path` for each error,
	//       add a config boolean that renders these a json objects instead
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

var debug = false

// ierrorf formats the internal error messages related to parsing and executing within the framework
func ierrorf(v string, is ...interface{}) error {
	var tag string
	if debug {
		_, file, line, _ := runtime.Caller(1)
		pieces := strings.Split(file, "/")
		tag = fmt.Sprintf("%s:%d: ", pieces[len(pieces)-1], line)
	}
	return fmt.Errorf(tag+v, is...)
}
