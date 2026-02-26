package service

import (
	"github.com/dyike/Void/internal/model"
	"github.com/dyike/Void/internal/repo"
)

// CalculatorService provides expression evaluation and unit conversion.
type CalculatorService struct {
	repo *repo.CalculatorRepo
}

// NewCalculatorService creates a new calculator service.
func NewCalculatorService(r *repo.CalculatorRepo) *CalculatorService {
	return &CalculatorService{repo: r}
}

// Evaluate evaluates a math expression and returns the result.
func (s *CalculatorService) Evaluate(expr string) (*model.CalcResult, error) {
	return s.repo.Evaluate(expr)
}

// IsLikelyExpression returns true if input looks like a math/conversion.
func (s *CalculatorService) IsLikelyExpression(input string) bool {
	return s.repo.IsLikelyExpression(input)
}
