package service

import (
	"fmt"
	"math"
	"os/exec"
	"strconv"
	"strings"
	"unicode"

	"github.com/dyike/Void/internal/model"
)

// CalculatorService provides expression evaluation and unit conversion.
type CalculatorService struct{}

// NewCalculatorService creates a new calculator service.
func NewCalculatorService() *CalculatorService {
	return &CalculatorService{}
}

// Evaluate evaluates a math expression and returns the result.
func (s *CalculatorService) Evaluate(expr string) (*model.CalcResult, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return &model.CalcResult{Input: expr, IsValid: false}, nil
	}

	if res := tryUnitConversion(expr); res != nil {
		res.IsValid = true
		return res, nil
	}
	if res := tryTempConversion(expr); res != nil {
		res.IsValid = true
		return res, nil
	}
	if res, ok := evalMath(expr); ok {
		return &model.CalcResult{
			Input:       expr,
			InputLabel:  "Expression",
			Result:      formatNumber(res),
			ResultLabel: numberWords(res),
			IsValid:     true,
		}, nil
	}

	out, err := exec.Command("python3", "-c",
		fmt.Sprintf("import math; print(eval(%q))", expr)).Output()
	if err == nil {
		r := strings.TrimSpace(string(out))
		if r != "" {
			return &model.CalcResult{Input: expr, Result: r, IsValid: true}, nil
		}
	}
	return &model.CalcResult{Input: expr, IsValid: false}, nil
}

// IsLikelyExpression returns true if input looks like a math/conversion.
func (s *CalculatorService) IsLikelyExpression(input string) bool {
	input = strings.TrimSpace(input)
	if len(input) < 2 {
		return false
	}
	lc := strings.ToLower(input)
	for _, kw := range []string{" to ", " in ", " as "} {
		if strings.Contains(lc, kw) {
			return true
		}
	}
	for _, fn := range []string{"sqrt", "abs", "sin", "cos", "tan", "log", "exp", "pow"} {
		if strings.Contains(lc, fn+"(") {
			return true
		}
	}
	hasDigit, hasOp := false, false
	for _, r := range input {
		if unicode.IsDigit(r) {
			hasDigit = true
		}
		if r == '+' || r == '-' || r == '*' || r == '/' || r == '^' || r == '%' {
			hasOp = true
		}
	}
	return hasDigit && hasOp
}

// ─── Unit Conversion ─────────────────────────────────────────────────────

type unitDef struct {
	aliases []string
	label   string
	symbol  string
	toBase  float64
}

type unitCat struct {
	name  string
	units []unitDef
}

var unitCategories = []unitCat{
	{name: "Length", units: []unitDef{
		{[]string{"mm", "millimeter", "millimeters"}, "Millimeters", "mm", 1e-3},
		{[]string{"cm", "centimeter", "centimeters"}, "Centimeters", "cm", 1e-2},
		{[]string{"m", "meter", "meters", "metre", "metres"}, "Meters", "m", 1},
		{[]string{"km", "kilometer", "kilometers"}, "Kilometers", "km", 1000},
		{[]string{"in", "inch", "inches"}, "Inches", "in", 0.0254},
		{[]string{"ft", "foot", "feet"}, "Feet", "ft", 0.3048},
		{[]string{"yd", "yard", "yards"}, "Yards", "yd", 0.9144},
		{[]string{"mi", "mile", "miles"}, "Miles", "mi", 1609.344},
	}},
	{name: "Mass", units: []unitDef{
		{[]string{"mg", "milligram", "milligrams"}, "Milligrams", "mg", 1e-3},
		{[]string{"g", "gram", "grams"}, "Grams", "g", 1},
		{[]string{"kg", "kilogram", "kilograms"}, "Kilograms", "kg", 1000},
		{[]string{"lb", "lbs", "pound", "pounds"}, "Pounds", "lb", 453.59237},
		{[]string{"oz", "ounce", "ounces"}, "Ounces", "oz", 28.349523125},
		{[]string{"t", "tonne", "tonnes", "metric ton"}, "Metric Tonnes", "t", 1e6},
	}},
	{name: "Data", units: []unitDef{
		{[]string{"b", "byte", "bytes"}, "Bytes", "B", 1},
		{[]string{"kb", "kilobyte", "kilobytes"}, "Kilobytes", "KB", 1000},
		{[]string{"kib", "kibibyte"}, "Kibibytes", "KiB", 1024},
		{[]string{"mb", "megabyte", "megabytes"}, "Megabytes", "MB", 1e6},
		{[]string{"mib", "mebibyte"}, "Mebibytes", "MiB", 1024 * 1024},
		{[]string{"gb", "gigabyte", "gigabytes"}, "Gigabytes", "GB", 1e9},
		{[]string{"gib", "gibibyte"}, "Gibibytes", "GiB", 1024 * 1024 * 1024},
		{[]string{"tb", "terabyte", "terabytes"}, "Terabytes", "TB", 1e12},
	}},
	{name: "Time", units: []unitDef{
		{[]string{"ms", "millisecond", "milliseconds"}, "Milliseconds", "ms", 1e-3},
		{[]string{"s", "sec", "second", "seconds"}, "Seconds", "s", 1},
		{[]string{"min", "minute", "minutes"}, "Minutes", "min", 60},
		{[]string{"h", "hr", "hour", "hours"}, "Hours", "h", 3600},
		{[]string{"day", "days", "d"}, "Days", "day", 86400},
		{[]string{"week", "weeks", "wk"}, "Weeks", "week", 604800},
	}},
	{name: "Speed", units: []unitDef{
		{[]string{"m/s", "mps"}, "Meters/second", "m/s", 1},
		{[]string{"km/h", "kph", "kmh"}, "Km/hour", "km/h", 1.0 / 3.6},
		{[]string{"mph", "miles/hour"}, "Miles/hour", "mph", 0.44704},
	}},
	{name: "Area", units: []unitDef{
		{[]string{"m2", "sq m", "square meter", "square meters"}, "Square Meters", "m²", 1},
		{[]string{"km2", "sq km", "square kilometer"}, "Square Km", "km²", 1e6},
		{[]string{"cm2", "sq cm"}, "Square Cm", "cm²", 1e-4},
		{[]string{"ft2", "sq ft", "square foot", "square feet"}, "Square Feet", "ft²", 0.09290304},
		{[]string{"in2", "sq in", "square inch"}, "Square Inches", "in²", 0.00064516},
		{[]string{"acre", "acres"}, "Acres", "ac", 4046.856},
		{[]string{"ha", "hectare", "hectares"}, "Hectares", "ha", 10000},
	}},
}

type unitMatch struct {
	catName string
	toBase  float64
	symbol  string
	label   string
}

var unitLookup map[string]unitMatch

func init() {
	unitLookup = make(map[string]unitMatch)
	for _, cat := range unitCategories {
		for _, u := range cat.units {
			for _, alias := range u.aliases {
				k := strings.ToLower(strings.TrimSpace(alias))
				if _, exists := unitLookup[k]; !exists {
					unitLookup[k] = unitMatch{cat.name, u.toBase, u.symbol, u.label}
				}
			}
		}
	}
}

func parseConversionQuery(q string) (value float64, from, to string, ok bool) {
	q = strings.TrimSuffix(strings.TrimSpace(q), "?")
	for _, kw := range []string{" to ", " in ", " as ", "="} {
		idx := strings.Index(strings.ToLower(q), kw)
		if idx < 0 {
			continue
		}
		left := strings.TrimSpace(q[:idx])
		right := strings.TrimSpace(q[idx+len(kw):])
		if left == "" || right == "" {
			continue
		}
		numEnd := 0
		for numEnd < len(left) {
			c := left[numEnd]
			if c >= '0' && c <= '9' || c == '.' || c == '-' || (c == ',' && numEnd > 0) {
				numEnd++
			} else {
				break
			}
		}
		if numEnd == 0 {
			continue
		}
		val, err := strconv.ParseFloat(strings.ReplaceAll(left[:numEnd], ",", ""), 64)
		if err != nil {
			continue
		}
		return val, strings.ToLower(strings.TrimSpace(left[numEnd:])), strings.ToLower(right), true
	}
	return 0, "", "", false
}

func tryUnitConversion(q string) *model.CalcResult {
	value, from, to, ok := parseConversionQuery(q)
	if !ok {
		return nil
	}
	fromM, fOk := unitLookup[from]
	toM, tOk := unitLookup[to]
	if !fOk || !tOk || fromM.catName != toM.catName {
		return nil
	}
	result := value * fromM.toBase / toM.toBase
	return &model.CalcResult{
		Input:       fmt.Sprintf("%s %s", formatNumber(value), fromM.symbol),
		InputLabel:  fromM.label,
		Result:      fmt.Sprintf("%s %s", formatNumber(result), toM.symbol),
		ResultLabel: toM.label,
	}
}

// ─── Temperature ─────────────────────────────────────────────────────────

type tempKey int

const (
	tempC tempKey = iota
	tempF
	tempK
)

var tempAliases = map[string]tempKey{
	"c": tempC, "celsius": tempC, "°c": tempC,
	"f": tempF, "fahrenheit": tempF, "°f": tempF,
	"k": tempK, "kelvin": tempK, "kelvins": tempK,
}

var tempSymbols = map[tempKey]string{tempC: "°C", tempF: "°F", tempK: "K"}
var tempLabels = map[tempKey]string{tempC: "Celsius", tempF: "Fahrenheit", tempK: "Kelvin"}

func tryTempConversion(q string) *model.CalcResult {
	value, from, to, ok := parseConversionQuery(q)
	if !ok {
		return nil
	}
	fromT, fOk := tempAliases[from]
	toT, tOk := tempAliases[to]
	if !fOk || !tOk {
		return nil
	}
	var celsius float64
	switch fromT {
	case tempC:
		celsius = value
	case tempF:
		celsius = (value - 32) * 5 / 9
	case tempK:
		celsius = value - 273.15
	}
	var result float64
	switch toT {
	case tempC:
		result = celsius
	case tempF:
		result = celsius*9/5 + 32
	case tempK:
		result = celsius + 273.15
	}
	return &model.CalcResult{
		Input:       fmt.Sprintf("%s %s", formatNumber(value), tempSymbols[fromT]),
		InputLabel:  tempLabels[fromT],
		Result:      fmt.Sprintf("%s %s", formatNumber(result), tempSymbols[toT]),
		ResultLabel: tempLabels[toT],
	}
}

// ─── Math Parser ──────────────────────────────────────────────────────────

func evalMath(expr string) (float64, bool) {
	e := strings.ReplaceAll(strings.TrimSpace(expr), " ", "")
	val, pos, ok := parseMath(e, 0)
	return val, ok && pos == len(e)
}

func parseMath(e string, pos int) (float64, int, bool) { return parseAS(e, pos) }

func parseAS(e string, pos int) (float64, int, bool) {
	left, pos, ok := parseMD(e, pos)
	if !ok {
		return 0, pos, false
	}
	for pos < len(e) && (e[pos] == '+' || e[pos] == '-') {
		op := e[pos]
		pos++
		right, np, ok2 := parseMD(e, pos)
		if !ok2 {
			return 0, np, false
		}
		pos = np
		if op == '+' {
			left += right
		} else {
			left -= right
		}
	}
	return left, pos, true
}

func parseMD(e string, pos int) (float64, int, bool) {
	left, pos, ok := parsePow(e, pos)
	if !ok {
		return 0, pos, false
	}
	for pos < len(e) {
		op := e[pos]
		if op != '*' && op != '/' && op != '%' {
			break
		}
		pos++
		if op == '*' && pos < len(e) && e[pos] == '*' {
			pos++
			right, np, ok2 := parseUnary(e, pos)
			if !ok2 {
				return 0, np, false
			}
			pos = np
			left = math.Pow(left, right)
			continue
		}
		right, np, ok2 := parsePow(e, pos)
		if !ok2 {
			return 0, np, false
		}
		pos = np
		switch op {
		case '*':
			left *= right
		case '/':
			if right == 0 {
				left = math.NaN()
			} else {
				left /= right
			}
		case '%':
			left = math.Mod(left, right)
		}
	}
	return left, pos, true
}

func parsePow(e string, pos int) (float64, int, bool) {
	left, pos, ok := parseUnary(e, pos)
	if !ok {
		return 0, pos, false
	}
	if pos < len(e) && e[pos] == '^' {
		pos++
		right, np, ok2 := parseUnary(e, pos)
		if !ok2 {
			return 0, np, false
		}
		pos = np
		left = math.Pow(left, right)
	}
	return left, pos, true
}

func parseUnary(e string, pos int) (float64, int, bool) {
	if pos < len(e) && e[pos] == '-' {
		v, np, ok := parsePrimary(e, pos+1)
		return -v, np, ok
	}
	if pos < len(e) && e[pos] == '+' {
		return parsePrimary(e, pos+1)
	}
	return parsePrimary(e, pos)
}

func parsePrimary(e string, pos int) (float64, int, bool) {
	if pos >= len(e) {
		return 0, pos, false
	}
	if e[pos] == '(' {
		v, np, ok := parseMath(e, pos+1)
		if !ok || np >= len(e) || e[np] != ')' {
			return 0, np, false
		}
		return v, np + 1, true
	}
	end, hasDot := pos, false
	for end < len(e) && (e[end] >= '0' && e[end] <= '9' || (!hasDot && e[end] == '.')) {
		if e[end] == '.' {
			hasDot = true
		}
		end++
	}
	if end == pos {
		return 0, pos, false
	}
	v, err := strconv.ParseFloat(e[pos:end], 64)
	if err != nil {
		return 0, pos, false
	}
	return v, end, true
}

// ─── Formatting ───────────────────────────────────────────────────────────

func formatNumber(f float64) string {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return "Error"
	}
	if f == math.Trunc(f) && math.Abs(f) < 1e15 {
		return strconv.FormatInt(int64(f), 10)
	}
	s := strconv.FormatFloat(f, 'f', 10, 64)
	if strings.Contains(s, ".") {
		s = strings.TrimRight(s, "0")
		s = strings.TrimRight(s, ".")
	}
	return s
}

func numberWords(f float64) string {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return ""
	}
	abs := math.Abs(f)
	switch {
	case abs >= 1e12:
		return fmt.Sprintf("%.3g trillion", f/1e12)
	case abs >= 1e9:
		return fmt.Sprintf("%.3g billion", f/1e9)
	case abs >= 1e6:
		return fmt.Sprintf("%.3g million", f/1e6)
	case abs >= 1e3:
		return fmt.Sprintf("%.3g thousand", f/1e3)
	default:
		return ""
	}
}
