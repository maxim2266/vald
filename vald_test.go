package vald

import (
	"fmt"
	"testing"
)

func TestSimple(t *testing.T) {
	// test data
	src := map[string]string{
		"aaa":  "zzz",
		"bbb":  "yyy",
		"ccc":  "xxx",
		"isOK": "1",
	}

	// expected result
	exp := map[string]string{
		"aaa":  "zzz",
		"bbb":  "yyy",
		"ccc":  "xxx",
		"AAA":  "XXX",
		"isOK": "true",
	}

	// validator
	validate := Pack(
		Opt("aaa", OneOf("xxx", "yyy", "zzz")),
		Req("bbb", Regex(`^[a-z]{3}$`)),
		Req("ccc", OneOf("xxx", "yyy")),
		OptDef("AAA", Regex(`^[A-Z]{3}$`), "XXX"),
		Req("isOK", Bool),
	)

	// do the validation
	count := 0

	err := validate(FromMap(src), func(k, v string) error {
		ev, ok := exp[k]

		if !ok {
			return fmt.Errorf("unexpected key %q with value %q", k, v)
		}

		if v != ev {
			return fmt.Errorf("unexpected value for key %q: %q instead of %q", k, v, ev)
		}

		t.Log(k, v)
		count++

		return nil
	})

	if err != nil {
		t.Error(err)
		return
	}

	if count != len(exp) {
		t.Errorf("unexpected result size: %d instead of %d", count, len(exp))
		return
	}
}

func TestErrors(t *testing.T) {
	// test cases
	type testCase struct {
		validator  Validator
		value, err string
	}

	cases := map[string]testCase{
		"aaa": {
			Req("aaa", OneOf("xxx", "yyy", "zzz")),
			"XXX",
			`parameter "aaa": invalid value: "XXX"`,
		},
		"bbb": {
			Req("bbb", Regex(`^[a-z]{3}$`)),
			"XXX",
			`parameter "bbb": invalid value: "XXX"`,
		},
		"isOK": {
			Req("isOK", Bool),
			"XXX",
			`parameter "isOK": invalid syntax: "XXX"`,
		},
		"missing": {
			Req("ddd", Bool),
			"xxx",
			`parameter "ddd": missing value`,
		},
	}

	// getter
	get := func(k string) string {
		if tc, ok := cases[k]; ok {
			return tc.value
		}

		return ""
	}

	// run test
	for k, tc := range cases {
		m, err := tc.validator.Map(get)

		if err == nil {
			t.Errorf("missing error for key %q", k)
			return
		}

		if len(m) > 0 {
			t.Errorf("got %d results instead of 0", len(m))
			return
		}

		if err.Error() != tc.err {
			t.Errorf("unexpected error message for key %q: %q instead of %q", k, err, tc.err)
			return
		}
	}
}

func TestCond(t *testing.T) {
	type pair struct {
		key, value string
	}

	type testCase struct {
		validate Validator
		input    map[string]string
		output   []pair
	}

	cases := []testCase{
		{
			Pack(Cond("aaa", Bool, Req("bbb", Bool), Req("ccc", Bool))),
			map[string]string{"aaa": "1", "bbb": "0"},
			[]pair{{"aaa", "true"}, {"bbb", "false"}},
		},
		{
			Pack(Cond("aaa", Bool, Req("bbb", Bool), Req("ccc", Bool))),
			map[string]string{"ccc": "0"},
			[]pair{{"ccc", "false"}},
		},
		{
			Pack(Cond("aaa", Bool, Req("bbb", Bool), nil)),
			map[string]string{"aaa": "1", "bbb": "0"},
			[]pair{{"aaa", "true"}, {"bbb", "false"}},
		},
		{
			Pack(Cond("aaa", Bool, nil, Req("ccc", Bool))),
			map[string]string{"ccc": "0"},
			[]pair{{"ccc", "false"}},
		},
	}

	for i, tc := range cases {
		j := 0
		err := tc.validate(FromMap(tc.input), func(k, v string) error {
			if j >= len(tc.output) {
				return fmt.Errorf("[%d] overflow with %d", i, j)
			}

			if k != tc.output[j].key {
				return fmt.Errorf("[%d] unexpected key: %q instead of %q", i, k, tc.output[j].key)
			}

			if v != tc.output[j].value {
				return fmt.Errorf("[%d] unexpected value with key %q: %q instead of %q",
					i, k, v, tc.output[j].value)
			}

			j++
			return nil
		})

		if err != nil {
			t.Error(err)
			return
		}

		if j != len(tc.output) {
			t.Errorf("[%d] unexpected output length: %d instead of %d", i, j, len(tc.output))
			return
		}
	}
}
