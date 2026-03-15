package security

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	domainauth "github.com/radhakrishna/archbattle/core/internal/domain/auth"
	"github.com/radhakrishna/archbattle/core/internal/domain/shared"
)

type JWTIssuer struct {
	secret []byte
}

type sessionClaims struct {
	Username string `json:"username"`
	Role     string `json:"role"`
	Tier     string `json:"tier"`
	ELO      int    `json:"elo"`
	jwt.RegisteredClaims
}

func NewJWTIssuer(secret string) *JWTIssuer {
	return &JWTIssuer{secret: []byte(secret)}
}

func (i *JWTIssuer) Issue(session *domainauth.Session, ttl time.Duration) (string, error) {
	claims := sessionClaims{
		Username: session.Username,
		Role:     session.Role,
		Tier:     string(session.Tier),
		ELO:      session.ELO,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   session.UserID.String(),
			ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(i.secret)
}

func (i *JWTIssuer) Parse(token string) (*domainauth.Session, error) {
	parsed, err := jwt.ParseWithClaims(token, &sessionClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return i.secret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := parsed.Claims.(*sessionClaims)
	if !ok || !parsed.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}
	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return nil, fmt.Errorf("parse token subject: %w", err)
	}
	return &domainauth.Session{UserID: userID, Username: claims.Username, Role: claims.Role, Tier: shared.Tier(claims.Tier), ELO: claims.ELO, ExpiresAt: claims.ExpiresAt.Time}, nil
}
