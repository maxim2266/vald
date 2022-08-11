# vald

[![GoDoc](https://godoc.org/github.com/maxim2266/vald?status.svg)](https://godoc.org/github.com/maxim2266/vald)
[![Go Report Card](https://goreportcard.com/badge/github.com/maxim2266/vald)](https://goreportcard.com/report/github.com/maxim2266/vald)
[![License: BSD 3 Clause](https://img.shields.io/badge/License-BSD_3--Clause-yellow.svg)](https://opensource.org/licenses/BSD-3-Clause)

Package `vald` encapsulates and drives validation logic for inputs like HTTP form data.

### Purpose
Consider a web server with many HTML pages, each containing a form which is then POSTed back to the server.
Validating each parameter of each form usually results in a number of functions with spaghetti code that is
hard to maintain and modify. This package provides a framework for defining such code in a much more compact
form, also taking care of all the low-level details of the validation process. Example:
```Go
// statically construct a validator function with a list of parameters to process
var validateForm1 = vald.Pack(
	// required parameter "id" with a value that must match the given regular expression
	vald.Req("id", vald.Regex(`^[0-9]{6}$`)),
	// optional parameter "name"
	vald.Opt("name", vald.Regex(`^[a-zA-Z]+$`)),
	// required parameter "amount"
	vald.Req("amount", vald.Regex(`^[1-9][0-9]*\.[0-9]{2}$`)),
	// optional parameter "currency" with the default value "GBP"
	vald.OptDef("currency", vald.OneOf("USD", "GBP", "EUR"), "GBP"),
)
```
This function can later be used in an HTTP POST handler:
```Go
func handleForm1(w http.ResponseWriter, r *http.Request) {
	err := validateForm1(r.FormValue, func(k, v string) error {
		// process key "k" with validated value "v"
		return nil
	})

	if err != nil {
		// some parameter is invalid or missing
	}

	// ...
}
```
Alternatively, the validator can be used to build a Go map of validated parameters:
```Go
func handleForm2(w http.ResponseWriter, r *http.Request) {
	params, err := validateForm1.Map(r.FormValue)

	if err != nil {
		// some parameter is invalid or missing
	}

	// here "params" is a map[string]string, filled with validated parameters
}
```

### How it works
All data types used in this package are functions, and they can always be substituted with other
user-provided functions with matching signature and suitable behaviour.

The main type is `Validator`, which is a function `func(Getter, Consumer) error`, and it is the one
that does the actual validation of one or more parameters. The package provides a number of constructors
for the functions of this type, in particular (see `go doc` for more details):
- `Pack` takes a variable list of other validators and constructs a single validator that when called invokes
	its arguments one by one in the order of specification;
- `Req` constructs a validator for a parameter with the given name and checker function that returns
	an error if the parameter is not present;
- `Opt` constructs a validator for a parameter with the given name and checker function that does nothing
	if the parameter is not present;
- `OptDef` constructs a validator for a parameter with the given name and checker function that supplies
	the given default value if the parameter is not present;
- `Cond` constructs a validator for a parameter with the given name and checker function that acts
	like the one produced by `Opt` constructor, but also proceeds with the first given validator if the
	parameter is present, or with the second validator otherwise.

All the above constructors (except `Pack`) take at least two parameters: parameter name as a string, and
a function of type `Checker` with the signature `func(string) (string, error)`.
The function receives a value of a parameter and returns the same (or overwritten) value, or an error. There
is a number of `Checker` constructors provided, like `Regex` or `OneOf`, see documentation for more details.

Each `Validator` function takes two parameters. The first one is `Getter` of type `func(string) string`. This
function is expected to return either the value associated with the given parameter name, or an empty
string. A number of functions from standard library fit this pattern, for example:
- `http.Request.FormValue`
- `http.Request.PostFormValue`
- `url.Values.Get`
- `os.Getenv`

The package also provides a convenience constructor `FromMap` that makes a `Getter` from the given
`map[string]string`.

The second parameter is a callback function `Consumer` of type `func(string, string) error`. It is typically
provided by the user. The callback is invoked each time a parameter passes validation.

The validation process stops at the first error encountered.

### Status
The package has been tested on Linux Mint 20.3. Required Go version: 1.17
