package format

import "testing"

func TestFormValues(t *testing.T) {
	type Bar struct {
		Age int64 `form:"age"`
	}
	type foo struct {
		Name  string `form:"foo[name]"`
		Baz   *Bar   `form:"foo[bar]"`
		Empty string `form:"empty"`
	}

	type testCase struct {
		f        foo
		expected map[string][]string
	}

	tests := []testCase{
		testCase{
			f: foo{Name: "oscar", Baz: &Bar{Age: 10}},
			expected: map[string][]string{
				"foo[name]":     []string{"oscar"},
				"foo[bar][age]": []string{"10"},
			},
		},
		testCase{
			f: foo{Name: "oscar", Baz: &Bar{Age: 10}, Empty: "notempty"},
			expected: map[string][]string{
				"foo[name]":     []string{"oscar"},
				"foo[bar][age]": []string{"10"},
				"empty":         []string{"notempty"},
			},
		},
	}
	for _, test := range tests {
		values := FormValues(&test.f)

		if len(values) != len(test.expected) {
			t.Fatalf("invalid length: %+v", values)
		}

		for k, v := range test.expected {
			realVal, ok := values[k]
			if !ok {
				t.Errorf("missing key: %+v", k)
			}

			if len(realVal) != 1 {
				t.Errorf("more than one element in form.Values")
			}

			if realVal[0] != v[0] {
				t.Errorf("expected %+v, got %+v", v[0], realVal[0])
			}
		}
	}
}
