package nacos

import "testing"

type testConversion struct {
	time     string
	expected string
}

var tests = map[string][]testConversion{
	"Shanghai": {
		{"01:00", "17:00"},
		{"23:59", "15:59"},
	},
	// Others
}

func runConversionTests(t *testing.T, tests []testConversion, conversionFunc func(string) (string, error)) {
	for _, test := range tests {
		actual, err := conversionFunc(test.time)
		if err != nil {
			t.Errorf("Unexpected error for %s: %v", test.time, err)
			continue
		}
		if actual != test.expected {
			t.Errorf("Expected %s for %s but got %s", test.expected, test.time, actual)
		}
	}
}
