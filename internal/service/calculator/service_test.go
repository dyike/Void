package calculator_test

import (
	"testing"

	"github.com/dyike/Void/internal/service/calculator"
)

func newSvc() *calculator.Service { return calculator.NewService() }

func TestMathExpressions(t *testing.T) {
	svc := newSvc()

	cases := []struct {
		expr   string
		result string
	}{
		{"2 + 2", "4"},
		{"10 * 5 + 3", "53"},
		{"100 / 4", "25"},
		{"2 ^ 10", "1024"},
		{"(3 + 4) * 2", "14"},
		{"100 % 7", "2"},
		{"1000000 + 1", "1000001"},
	}

	for _, tc := range cases {
		t.Run(tc.expr, func(t *testing.T) {
			r, err := svc.Evaluate(tc.expr)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if !r.IsValid {
				t.Fatal("expected IsValid=true")
			}
			if r.Result != tc.result {
				t.Errorf("got %q want %q", r.Result, tc.result)
			}
		})
	}
}

func TestUnitConversion(t *testing.T) {
	svc := newSvc()

	cases := []struct {
		expr   string
		result string
	}{
		// Length
		{"1 km to m", "1000 m"},
		{"100 cm to m", "1 m"},
		{"1 mile to km", "1.60934 km"},
		// Mass
		{"1 kg to lb", "2.20462 lb"},
		{"1 lb to kg", "0.453592 kg"},
		// Data
		{"1 gb to mb", "1000 MB"},
		{"1 gib to mib", "1024 MiB"},
		// Time
		{"1 day to h", "24 h"},
		{"1 hour to min", "60 min"},
		// Speed
		{"100 km/h to m/s", "27.7778 m/s"},
	}

	for _, tc := range cases {
		t.Run(tc.expr, func(t *testing.T) {
			r, err := svc.Evaluate(tc.expr)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if !r.IsValid {
				t.Fatalf("expected IsValid=true for %q, got result=%q", tc.expr, r.Result)
			}
		})
	}
}

func TestTemperatureConversion(t *testing.T) {
	svc := newSvc()

	cases := []struct {
		expr       string
		resultWant string
	}{
		{"100 c to f", "212 °F"},
		{"32 f to c", "0 °C"},
		{"0 c to k", "273.15 K"},
		{"273.15 k to c", "0 °C"},
	}

	for _, tc := range cases {
		t.Run(tc.expr, func(t *testing.T) {
			r, err := svc.Evaluate(tc.expr)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if !r.IsValid {
				t.Fatalf("expected IsValid=true for %q", tc.expr)
			}
			if r.Result != tc.resultWant {
				t.Errorf("got %q want %q", r.Result, tc.resultWant)
			}
		})
	}
}

func TestIsLikelyExpression(t *testing.T) {
	svc := newSvc()

	trues := []string{
		"2 + 2", "100 * 5", "sqrt(16)", "100 km to m", "32 f to c",
	}
	falses := []string{
		"hello", "wails", "123", "", "a",
	}

	for _, s := range trues {
		if !svc.IsLikelyExpression(s) {
			t.Errorf("expected IsLikelyExpression(%q) = true", s)
		}
	}
	for _, s := range falses {
		if svc.IsLikelyExpression(s) {
			t.Errorf("expected IsLikelyExpression(%q) = false", s)
		}
	}
}
