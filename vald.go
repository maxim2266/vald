/*
Copyright (c) 2022 Maxim Konakov
All rights reserved.

Redistribution and use in source and binary forms, with or without modification,
are permitted provided that the following conditions are met:

1. Redistributions of source code must retain the above copyright notice,
   this list of conditions and the following disclaimer.
2. Redistributions in binary form must reproduce the above copyright notice,
   this list of conditions and the following disclaimer in the documentation
   and/or other materials provided with the distribution.
3. Neither the name of the copyright holder nor the names of its contributors
   may be used to endorse or promote products derived from this software without
   specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE DISCLAIMED.
IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT,
INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING,
BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY
OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING
NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE,
EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
*/

/*
Package vald encapsulates and drives validation logic for inputs like HTTP form data.

Consider a web server with many HTML pages, each containing a form which is then POSTed back to the server.
Validating each parameter of each form usually results in a number of functions with spaghetti code that is
hard to maintain and modify. This package provides a framework for defining such code in a much more compact
form, also taking care of all the low-level details of the validation process.
*/
package vald

import (
	"errors"
	"regexp"
	"strconv"
)

// Getter is the type of function that given a key returns its corresponding value, or an empty string
// if the key is not found.
// Typically, the getter is one of:
//   - http.Request.FormValue
//   - http.Request.PostFormValue
//   - url.Values.Get
//   - os.Getenv
type Getter = func(string) string

// Consumer is the type of callback function to be called with validated key/value pairs.
type Consumer = func(string, string) error

// Validator is the type of function that performs validation.
type Validator func(Getter, Consumer) error

// Map invokes the validator using the given getter function and returns a map of validated data,
// or an error.
func (validate Validator) Map(get Getter) (map[string]string, error) {
	r := make(map[string]string)

	cons := func(k, v string) error { r[k] = v; return nil }

	if err := validate(get, cons); err != nil {
		return nil, err
	}

	return r, nil
}

// Pack constructs a new validator that when called invokes the given validators one by one.
func Pack(validators ...Validator) Validator {
	if len(validators) == 0 {
		panic("empty validator list in vald.Pack()")
	}

	return func(get Getter, cons Consumer) (err error) {
		for _, validate := range validators {
			if err = validate(get, cons); err != nil {
				break
			}
		}

		return
	}
}

// Checker is the type of function that validates the given value.
type Checker = func(string) (string, error)

// Req constructs a validator that when called retrieves the parameter with the given key and
// validates its value with the given Checker function. The validator returns an error if the
// parameter is nor present.
func Req(key string, check Checker) Validator {
	return func(get Getter, cons Consumer) error {
		if val := get(key); len(val) > 0 {
			return doCheck(key, val, check, cons)
		}

		return errors.New("missing key: " + strconv.Quote(key))
	}
}

// Opt constructs a validator that when called retrieves the parameter with the given key and
// validates its value with the given Checker function. The validator does nothing if the
// parameter is nor present.
func Opt(key string, check Checker) Validator {
	return func(get Getter, cons Consumer) (err error) {
		if val := get(key); len(val) > 0 {
			err = doCheck(key, val, check, cons)
		}

		return
	}
}

// OptDef constructs a validator that when called retrieves the parameter with the given key and
// validates its value with the given Checker function. The validator calls Consumer function
// with the given default value if the parameter is not present. The default value is not validated
// with the Checker function.
func OptDef(key string, check Checker, deflt string) Validator {
	return func(get Getter, cons Consumer) error {
		if val := get(key); len(val) > 0 {
			return doCheck(key, val, check, cons)
		}

		return cons(key, deflt)
	}
}

// Cond constructs a validator that when called retrieves the parameter with the given key, and
// if the parameter is present it gets validated with the given Checker and then the control is
// passed over to the "yes" validator, othewise the function proceeds with the "no" validator.
func Cond(key string, check Checker, yes, no Validator) Validator {
	if yes == nil && no == nil {
		panic("nil validators in vald.OptCond()")
	}

	return func(get Getter, cons Consumer) (err error) {
		if val := get(key); len(val) > 0 {
			if err = doCheck(key, val, check, cons); err == nil && yes != nil {
				err = yes(get, cons)
			}
		} else if no != nil {
			err = no(get, cons)
		}

		return
	}
}

func doCheck(key, val string, check Checker, cons Consumer) (err error) {
	if val, err = check(val); err != nil {
		return errors.New("invalid value for key " + strconv.Quote(key) + ": " + err.Error())
	}

	return cons(key, val)
}

// OneOf constructs a Checker function that attempts to find its argument in the given list of
// string literals, and if no match is found the checker returns an error.
func OneOf(literals ...string) Checker {
	if len(literals) == 0 {
		panic("empty list of literals in vald.OneOf()")
	}

	m := make(map[string]string, len(literals))

	for i, lit := range literals {
		if len(lit) == 0 {
			panic("empty literal in vald.OneOf() at index " + strconv.Itoa(i))
		}

		m[lit] = lit // all literals interned
	}

	return func(val string) (s string, err error) {
		if s = m[val]; len(s) == 0 {
			err = errors.New(strconv.Quote(val))
		}

		return
	}
}

// Regex constructs a Checker function that matches its argument against the given regular expression.
func Regex(patt string) Checker {
	match := regexp.MustCompile(patt).MatchString

	return func(val string) (string, error) {
		if match(val) {
			return val, nil
		}

		return "", errors.New(strconv.Quote(val))
	}
}

// Bool is a Checker function that attempts to convert the given value to boolean using
// the same conversion rules as in strconv.ParseBool() function. Upon successful conversion
// the checker returns the value as either string "true" or "false".
func Bool(val string) (string, error) {
	flag, err := strconv.ParseBool(val)

	if err != nil {
		return "", mapNumErr(val, err)
	}

	return strconv.FormatBool(flag), nil
}

// map error returned from strconv.Parse* family of functions.
func mapNumErr(val string, err error) error {
	if e, ok := err.(*strconv.NumError); ok {
		err = e.Err
	}

	return errors.New(strconv.Quote(val) + ": " + err.Error())
}

// FromMap is a convenience function that constructs a Getter from the given Go map.
func FromMap(m map[string]string) Getter {
	return func(k string) string {
		return m[k]
	}
}
