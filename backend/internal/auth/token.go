package auth

import (
	"workshop/backend/internal/model"
)

// IssueToken signs a JWT for the employee. This ticket ships the signature
// only; ticket #9 fills the body.
func IssueToken(employeeID int, email string) (string, error) {
	return "", nil
}

// VerifyToken validates a token and returns its claims. This ticket ships the
// signature only; ticket #9 fills the body.
func VerifyToken(token string) (model.Claims, error) {
	return model.Claims{}, nil
}
