// Package token issues and verifies the JWTs used for dashboard auth and
// subscriber WebSocket auth.
package token

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims is the JWT payload for an authenticated user/workspace.
type Claims struct {
	UserID      string `json:"uid"`
	WorkspaceID string `json:"wid"`
	jwt.RegisteredClaims
}

// SubscriberClaims authenticates a subscriber's WebSocket connection.
type SubscriberClaims struct {
	WorkspaceID  string `json:"wid"`
	SubscriberID string `json:"sid"`
	jwt.RegisteredClaims
}

// Issue creates a signed JWT for a user/workspace.
func Issue(secret, userID, workspaceID string, ttl time.Duration) (string, error) {
	claims := Claims{
		UserID:      userID,
		WorkspaceID: workspaceID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := t.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return s, nil
}

// Parse validates a user JWT and returns its claims.
func Parse(secret, tokenStr string) (*Claims, error) {
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(tokenStr, claims, keyFunc(secret))
	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}
	return claims, nil
}

// IssueSubscriber creates a signed JWT scoped to a single subscriber.
func IssueSubscriber(secret, workspaceID, subscriberID string, ttl time.Duration) (string, error) {
	claims := SubscriberClaims{
		WorkspaceID:  workspaceID,
		SubscriberID: subscriberID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   subscriberID,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := t.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("sign subscriber token: %w", err)
	}
	return s, nil
}

// ParseSubscriber validates a subscriber JWT and returns its claims.
func ParseSubscriber(secret, tokenStr string) (*SubscriberClaims, error) {
	claims := &SubscriberClaims{}
	_, err := jwt.ParseWithClaims(tokenStr, claims, keyFunc(secret))
	if err != nil {
		return nil, fmt.Errorf("parse subscriber token: %w", err)
	}
	return claims, nil
}

func keyFunc(secret string) jwt.Keyfunc {
	return func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	}
}
