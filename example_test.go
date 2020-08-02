package validator_test

import (
	"fmt"

	"github.com/marksalpeter/validator"
)

func ExampleValidator() {
	type User struct {
		FirstName    string `json:"firstName,omitempty" validate:"name"`           // firstName and lastName must be between 1 and 20 letters each
		LastName     string `json:"lastName,omitempty" validate:"name"`            // numbers and special characters will fail validation
		EmailAddress string `json:"emailAddress,omitempty" validate:"email"`       // EmailAddress must be a valid email address
		PhoneNumber  string `json:"phoneNumber,omitempty" validate:"number:11,15"` // PhoneNumber must be 11-15 numbers e.g 15551234567
	}

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
	if err := v.Validate(&user); err != nil {
		panic(err)
	}
	fmt.Println("user data is valid!")

	// Output:
	// ["'firstName' must be a valid name","'lastName' must be a valid name","'emailAddress' must be a valid email address","'phoneNumber' must contain only numbers"]
	// user data is valid!
}

func ExampleXOR() {
	// User can either set name or firstName and lastName
	type User struct {
		Name      string `json:"name,omitempty" validate:"name | or:FirstName,LastName"`
		FirstName string `json:"firstName,omitempty" validate:"empty | (name & and:LastName)"`
		LastName  string `json:"lastName,omitempty" validate:"empty | (name & and:FirstName)"`
	}

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

	// Output:
	// ["either 'name', 'firstName' and/or 'lastName' must be set"]
	// ["'firstName' and 'lastName' must be set"]
	// ["'lastName' and 'firstName' must be set"]
}
