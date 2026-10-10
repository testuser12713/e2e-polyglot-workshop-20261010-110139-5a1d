package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"workshop/backend/internal/model"
)

// tokenTTL is how long an issued token stays valid. A token whose exp has
// passed is rejected at every workshop route (AC-23).
const tokenTTL = 12 * time.Hour

// errNoSecret is returned when JWT_SECRET is not configured. The secret is read
// lazily from the environment so the process never dies at import time; the
// API's startup validation names the variable (AC-26).
var errNoSecret = errors.New("JWT_SECRET is not set (see RUN.json)")

// jwtSecret reads the signing key from the environment.
func jwtSecret() ([]byte, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return nil, errNoSecret
	}
	return []byte(secret), nil
}

// jwtHeader is the fixed protected header of the issued tokens.
type jwtHeader struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
}

// encodeSegment is the base64url encoding the JWT format uses (no padding).
func encodeSegment(raw []byte) string {
	return base64.RawURLEncoding.EncodeToString(raw)
}

// signToken produces an HS256 JWT for the given claims. It is the single
// signing path shared by IssueToken and the tests.
func signToken(claims model.Claims) (string, error) {
	secret, err := jwtSecret()
	if err != nil {
		return "", err
	}
	header, err := json.Marshal(jwtHeader{Alg: "HS256", Typ: "JWT"})
	if err != nil {
		return "", fmt.Errorf("encode token header: %w", err)
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("encode token claims: %w", err)
	}
	signingInput := encodeSegment(header) + "." + encodeSegment(payload)
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(signingInput))
	return signingInput + "." + encodeSegment(mac.Sum(nil)), nil
}

// IssueToken signs a JWT for the employee. The claims are exactly the
// contract's shape: sub carries the employee id as a string, plus email and
// exp in unix seconds.
func IssueToken(employeeID int, email string) (string, error) {
	claims := model.Claims{
		Sub:   strconv.Itoa(employeeID),
		Email: email,
		Exp:   time.Now().Add(tokenTTL).Unix(),
	}
	return signToken(claims)
}

// VerifyToken validates a token's signature and expiry and returns its claims.
// A malformed token, a bad signature or an expired token is an error, so
// RequireAuth rejects it with 401 (AC-15, AC-23).
func VerifyToken(token string) (model.Claims, error) {
	secret, err := jwtSecret()
	if err != nil {
		return model.Claims{}, err
	}

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return model.Claims{}, errors.New("malformed token")
	}

	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return model.Claims{}, errors.New("malformed token signature")
	}
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(parts[0] + "." + parts[1]))
	if !hmac.Equal(sig, mac.Sum(nil)) {
		return model.Claims{}, errors.New("invalid token signature")
	}

	headerRaw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return model.Claims{}, errors.New("malformed token header")
	}
	var header jwtHeader
	if err := json.Unmarshal(headerRaw, &header); err != nil || header.Alg != "HS256" {
		return model.Claims{}, errors.New("unsupported token algorithm")
	}

	payloadRaw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return model.Claims{}, errors.New("malformed token payload")
	}
	var claims model.Claims
	if err := json.Unmarshal(payloadRaw, &claims); err != nil {
		return model.Claims{}, errors.New("malformed token claims")
	}

	if claims.Exp <= time.Now().Unix() {
		return model.Claims{}, errors.New("token expired")
	}
	return claims, nil
}
