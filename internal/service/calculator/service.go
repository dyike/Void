package calculator

import (
	"fmt"
	"math"
	"os/exec"
	"strconv"
	"strings"
	"unicode"
)

// Service provides expression evaluation.
type Service struct{}

// NewService creates a new calculator service.
func NewService() *Service { return &Service{} }

// CalcResult holds the evaluation output.
type CalcResult struct {
	Input   string `json:"input"`
	Result  string `json:"result"`
	IsValid bool   `json:"isValid"`
}

// Evaluate evaluates a math expression and returns the result.
func (s *Service) Evaluate(expr string) (*CalcResult, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return &CalcResult{Input: expr, IsValid: false}, nil
	}

	// First try Go built-in evaluator
	if result, ok := evalExpr(expr); ok {
		formatted := formatNumber(result)
		return &CalcResult{Input: expr, Result: formatted, IsValid: true}, nil
	}

	// Fallback: try Python for complex expressions
	out, err := exec.Command("python3", "-c", fmt.Sprintf("import math; print(eval(%q))", expr)).Output()
	if err == nil {
		res := strings.TrimSpace(string(out))
		return &CalcResult{Input: expr, Result: res, IsValid: true}, nil
	}

	return &CalcResult{Input: expr, IsValid: false}, nil
}

// IsLikelyExpression returns true if the string looks like a math expression.
func (s *Service) IsLikelyExpression(input string) bool {
	input = strings.TrimSpace(input)
	if len(input) < 2 {
		return false
	}

	hasOperator := false
	hasDigit := false
	for _, r := range input {
		if unicode.IsDigit(r) {
			hasDigit = true
		}
		if r == '+' || r == '-' || r == '*' || r == '/' || r == '^' || r == '%' {
			hasOperator = true
		}
		// If it has letters that look like variable names (not math funcs), skip
		if unicode.IsLetter(r) {
			lc := strings.ToLower(input)
			for _, fn := range []string{"sin", "cos", "tan", "sqrt", "log", "pi", "e"} {
				if strings.Contains(lc, fn) {
					goto maybeMath
				}
			}
			return false
		}
	maybeMath:
	}
	return hasDigit && hasOperator
}

// ─── Simple expression evaluator ─────────────────────────────────────────

// evalExpr evaluates a simple infix expression.
func evalExpr(expr string) (float64, bool) {
	expr = strings.ReplaceAll(expr, " ", "")
	expr = strings.ReplaceAll(expr, "^", "**") // normalize
	val, pos, ok := parseExpr(expr, 0)
	if !ok || pos != len(expr) {
		return 0, false
	}
	return val, true
}

func parseExpr(expr string, pos int) (float64, int, bool) {
	return parseAddSub(expr, pos)
}

func parseAddSub(expr string, pos int) (float64, int, bool) {
	left, pos, ok := parseMulDiv(expr, pos)
	if !ok {
		return 0, pos, false
	}
	for pos < len(expr) && (expr[pos] == '+' || expr[pos] == '-') {
		op := expr[pos]
		pos++
		right, np, ok2 := parseMulDiv(expr, pos)
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

func parseMulDiv(expr string, pos int) (float64, int, bool) {
	left, pos, ok := parsePow(expr, pos)
	if !ok {
		return 0, pos, false
	}
	for pos < len(expr) && (expr[pos] == '*' || expr[pos] == '/' || expr[pos] == '%') {
		op := expr[pos]
		pos++
		// Check for ** (power)
		if op == '*' && pos < len(expr) && expr[pos] == '*' {
			pos++
			right, np, ok2 := parseUnary(expr, pos)
			if !ok2 {
				return 0, np, false
			}
			pos = np
			left = math.Pow(left, right)
			continue
		}
		right, np, ok2 := parsePow(expr, pos)
		if !ok2 {
			return 0, np, false
		}
		pos = np
		switch op {
		case '*':
			left *= right
		case '/':
			if right == 0 {
				return 0, pos, false
			}
			left /= right
		case '%':
			left = math.Mod(left, right)
		}
	}
	return left, pos, true
}

func parsePow(expr string, pos int) (float64, int, bool) {
	return parseUnary(expr, pos)
}

func parseUnary(expr string, pos int) (float64, int, bool) {
	if pos < len(expr) && expr[pos] == '-' {
		val, np, ok := parsePrimary(expr, pos+1)
		return -val, np, ok
	}
	if pos < len(expr) && expr[pos] == '+' {
		return parsePrimary(expr, pos+1)
	}
	return parsePrimary(expr, pos)
}

func parsePrimary(expr string, pos int) (float64, int, bool) {
	if pos >= len(expr) {
		return 0, pos, false
	}
	// Parentheses
	if expr[pos] == '(' {
		val, np, ok := parseExpr(expr, pos+1)
		if !ok || np >= len(expr) || expr[np] != ')' {
			return 0, np, false
		}
		return val, np + 1, true
	}
	// Number
	end := pos
	hasDot := false
	for end < len(expr) && (expr[end] >= '0' && expr[end] <= '9' || (!hasDot && expr[end] == '.')) {
		if expr[end] == '.' {
			hasDot = true
		}
		end++
	}
	if end == pos {
		return 0, pos, false
	}
	val, err := strconv.ParseFloat(expr[pos:end], 64)
	if err != nil {
		return 0, pos, false
	}
	return val, end, true
}

func formatNumber(f float64) string {
	if f == math.Trunc(f) && math.Abs(f) < 1e15 {
		return strconv.FormatInt(int64(f), 10)
	}
	s := strconv.FormatFloat(f, 'f', 10, 64)
	// Trim trailing zeros
	if strings.Contains(s, ".") {
		s = strings.TrimRight(s, "0")
		s = strings.TrimRight(s, ".")
	}
	return s
}
