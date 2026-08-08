// Package auth issues and verifies the bearer tokens the API is protected with.
package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/sovanna2011/farm-area/backend/internal/domain"
)

type Tokens struct {
	secret []byte
	ttl    time.Duration
	issuer string
}

func NewTokens(secret string, ttl time.Duration) *Tokens {
	return &Tokens{secret: []byte(secret), ttl: ttl, issuer: "farm-area"}
}

type Claims struct {
	jwt.RegisteredClaims
	UserID   int    `json:"uid"`
	FullName string `json:"name"`
	Role     string `json:"role"`
}

func (t *Tokens) Issue(u domain.User) (string, time.Time, error) {
	expires := time.Now().Add(t.ttl)
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   u.Username,
			Issuer:    t.issuer,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(expires),
		},
		UserID:   u.ID,
		FullName: u.FullName,
		Role:     u.Role,
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(t.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign token: %w", err)
	}
	return signed, expires, nil
}

// Parse verifies the signature, the issuer and the expiry. The signing method is pinned: without
// that check a token with alg "none", or one signed with the public half of an RSA key, would be
// accepted as valid.
func (t *Tokens) Parse(raw string) (domain.User, error) {
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(raw, claims, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method %v", token.Header["alg"])
		}
		return t.secret, nil
	}, jwt.WithIssuer(t.issuer), jwt.WithExpirationRequired())
	if err != nil {
		return domain.User{}, err
	}
	if !domain.IsOneOf(claims.Role, []string{domain.RoleAdmin, domain.RoleManager, domain.RolePlanner, domain.RoleViewer}) {
		return domain.User{}, fmt.Errorf("token carries an unknown role %q", claims.Role)
	}
	return domain.User{
		ID:       claims.UserID,
		Username: claims.Subject,
		FullName: claims.FullName,
		Role:     claims.Role,
	}, nil
}

func HashPassword(plain string) (string, error) {
	h, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	return string(h), err
}

func CheckPassword(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}
