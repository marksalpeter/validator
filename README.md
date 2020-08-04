[![GoDoc](https://godoc.org/github.com/golang/gddo?status.svg)](http://godoc.org/github.com/marksalpeter/validator)

This is a validation package for golang that returns human readable error messages. Its ideal for validating input data for public facing restful api's. 

It has the following features
* Combination of validators with logical operators (e.g. `&`, `|`, `()`)
* Cross field and cross struct validation (e.g. `firstName and lastName must be set`)
* Custom validators, e.g. `validator.AddRule("name", func(ps..) error)`
* Customizable i18n aware error messages using the `golang.org/x/text/message` package

## How it works
`Validator` uses `struct` tags to verify data passed in to apis. Use the `validate` tag to apply various `Rule`s that the field must follow (e.g. `validate:"email"`). You can add custom validation rules as necessary by implementing your own `validator.Rule` functions. This package also comes with [several common rules referenced below](#Validation-Rules) such as `number:min,max`, `email`, `password`, etc.

### Example
```go
package main

import "github.com/marksalpeter/validator"

type User struct {
	FirstName    string `json:"firstName,omitempty" validate:"name"`           // FirstName cannot contain numbers or special characters
	LastName     string `json:"lastName,omitempty" validate:"name"`            // LastName cannot contain numbers or special characters
	EmailAddress string `json:"emailAddress,omitempty" validate:"email"`       // EmailAddress must be a valid email address
	PhoneNumber  string `json:"phoneNumber,omitempty" validate:"number:11,15"` // PhoneNumber must be 11-15 numbers e.g 15551234567
}

func main() {

	// displays all validation errors
	var user User
	v := validator.New()
	if err := v.Validate(&user); err != nil {
		fmt.Println(err) // prints ["'firstName' must be a valid name","'lastName' must be a valid name","'emailAddress' must be a valid email address","'phoneNumber' must contain only numbers"]
	}

	// set the properties
	user.FirstName = "First"
	user.LastName = "Last"
	user.EmailAddress = "email@address.com"
	user.PhoneNumber = "15551234567"
	if err := v.Validate(); err != nil {
		panic(err)
	}
	fmt.Println("user data is valid!")
}
```

### Advanced Example
```go
package main

import "github.com/marksalpeter/validator"

// User can either set name or firstName and lastName
type User struct {
	Name      string `json:"name,omitempty" validate:"name | or:FirstName,LastName"`
	FirstName string `json:"firstName,omitempty" validate:"empty | (name & and:LastName)"`
	LastName  string `json:"lastName,omitempty" validate:"empty | (name & and:FirstName)"`
}

func main() {

	var user User
	v := validator.New()

	// empty struct fails on `or:FirstName,LastName`
	if err := v.Validate(&user); err != nil {
		fmt.Println(err) // prints ["either 'name', 'firstName' and/or 'lastName' must be set"]
	}

	// only first name fails on `(name & and:LastName)`
	user.FirstName = "First"
	user.LastName = ""
	if err := v.Validate(&user); err != nil {
		fmt.Println(err) // prints ["'firstName' and 'lastName' must be set"]
	}

	// only last name fails on (name & and:FirstName)
	user.FirstName = ""
	user.LastName = "Last"
	if err := v.Validate(&user); err != nil {
		fmt.Println(err) // prints ["'lastName' and 'firstName' must be set"]
	}

	// first and last name are set first name passes
	user.FirstName = "First"
	user.LastName = "Last"
	if err := v.Validate(&user); err != nil {
		fmt.Println(err) // never reached
	}

	// name passes
	user.Name = "First Last"
	user.FirstName = ""
	user.LastName = ""
	if err := v.Validate(&user); err != nil {
		fmt.Println(err) // never reached
	}

}
```

## Validation Rules
This package comes with several default validation rules:

| Rule  | Description  |
| :--- | :--- |
| [required](#required-) | `required` returns an error if the filed contains the zero value of the type or nil |
| [empty](#empty-) | `empty` returns an error if the field is not empty |
| [name](#name-) | `name` returns an error if the field doesn't contain a valid name |
| [email](#email-) | `email` returns an error if the field doesn't contain a valid email address |
| [password](#password-) | `password` returns an error if the field doesn't contain a valid password |
| [number](#number-) | `number` retuns an error if the field doesn't contain numbers only |
| [letters](#letters-) | `letters` retuns an error if the field doesn't contain letters only |
| [eq](#eq-) | `eq` returns an error if the field does not == one of the params passed in |
| [xor](#xor-) | `xor` returns an error when more than one or zero of either the field that it is applied to or any of the field names passed as params are set to a non zero value |
| [or](#or-) | `or` returns an error when neither the field that it is applied to nor any of the field names passed as params are set to a non zero value |
| [and](#and-) | `and` returns an error when the field that it is applied to or any of the field names passed as params are set to the zero value |


### Required [^](#Validation-Rules)
Required returns an error if the filed contains the zero value of the type or nil.
#### Example
```go
type Struct struct {
	Field  string `json:"field" validate:"required"` // 'field' is required
}
```

### Empty [^](#Validation-Rules)
Empty returns an error if the field is not empty. It should be 'or'd together with
other rules that require manditory input
#### Example
```go
type Struct struct {
	Field  string `json:"field" validate:"empty | email"` // 'field' must be a valid email address or not set at all
}
```

### Name [^](#Validation-Rules)
Name returns an error if the field doesn't contain a valid name
i.e. no numbers or most special characters, excepting characters that may be in a name like a -
and allowing foreign language letters with accent marks as well as spaces
This prevents things like emails or phone numbers from being entered as a name.
#### Example
```go
type Struct struct {
	Field  string `json:"field" validate:"name"` // 'field' must be a valid name
}
```

### Email [^](#Validation-Rules)
Email returns an error if the field doesn't contain a valid email address
#### Example
```go
type Struct struct {
	Field  string `json:"field" validate:"email"` // 'field' must be a valid email address
}
```

### Password [^](#Validation-Rules)
Password returns an error if the field doesn't contain a valid password
#### Example
```go
type Struct struct {
	Field  string `json:"field" validate:"password"` // 'field' must be a valid password
}
```

### Number [^](#Validation-Rules)
Number retuns an error if the field doesn't contain numbers only
#### Example
```go
type Struct struct {
	Field   string `json:"field" validate:"number"`      // 'field' must contain only numbers
	Field2  string `json:"field2" validate:"number:3,5"` // 'field2' must be 3 to 5 digits
	Field3  uint   `json:"field3" validate:"number:3,5"` // 'field3' must be 3 to 5
}
```

### Letters [^](#Validation-Rules)
Letters retuns an error if the field doesn't contain letters only
#### Example
```go
type Struct struct {
	Field  string `json:"field" validate:"letters"` // 'field' can only take letters and spaces
}
```

### EQ [^](#Validation-Rules)
EQ returns an error if the field does not == one of the params passed in
#### Example
```go
type Struct struct {
	Field  string `json:"field" validate:"eq:one,two,three"` // 'field' must equal either "one", "two", or "three"
}
```

### XOR [^](#Validation-Rules)
XOR returns an error when more than one or zero of either the field that it is applied to or any of the field names passed as params are set to a non zero value
#### Example
```go
type Struct struct {
	Field  string `json:"field" validate:" xor:Field2"` // either "field" or "Field2" must be set
	Field2 string
}
```

### OR [^](#Validation-Rules)
OR returns an error when neither the field that it is applied to nor any of the field names passed as params are set to a non zero value
#### Example
```go
type Struct struct {
	Field  string `json:"field" validate:"or:Field2"` // either "field" or "Field2" or both must be set
	Field2 string
}
```

### AND [^](#Validation-Rules)
AND returns an error when the field that it is applied to or any of the field names passed as params are set to the zero value
#### Example
```go
type Struct struct {
	Field  string `json:"field" validate:"and:Field2"` // "field" and "Field2" must be set
	Field2 string
}
```

## Special Mentions
This package was heavily inspired by the [go-playground validator](https://github.com/go-playground/validator) and was originally envisioned as a more flexible, powerful version of the same basic concept.

## How to Contribute
Open a pull request 😁

## License
Distributed under MIT License, please see license file within the code for more details.
