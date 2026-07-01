package jwtx

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/rs/xid"
)

type Auther interface {
	// Generate a JWT (JSON Web Token) with the provided subject.
	GenerateToken(ctx context.Context, subject string) (TokenInfo, error)
	// Generate a JWT with custom expiration time.
	GenerateTokenWithExpired(ctx context.Context, subject string, expired int) (TokenInfo, error)
	// Generate a refresh token with the provided subject and tenant.
	GenerateRefreshToken(ctx context.Context, subject string, tenant string) (RefreshTokenInfo, error)
	// Parse and validate a refresh token, returns subject, tenant, jti.
	ParseRefreshToken(ctx context.Context, refreshToken string) (string, string, string, error)
	// Invalidate a token by removing it from the token store.
	DestroyToken(ctx context.Context, accessToken string) error
	// Invalidate a refresh token by adding it to the blacklist.
	DestroyRefreshToken(ctx context.Context, refreshToken string) error
	// Parse the subject (or user identifier) from a given access token.
	ParseSubject(ctx context.Context, accessToken string) (string, error)
	// Release any resources held by the JWTAuth instance.
	Release(ctx context.Context) error
}

const defaultKey = "CG24SDVP8OHPK395GB5G"

var (
	ErrInvalidToken = errors.New("Invalid token")
	ErrTokenRevoked = errors.New("Token has been revoked")
)

type options struct {
	signingMethod  jwt.SigningMethod
	signingKey     []byte
	signingKey2    []byte
	keyFuncs       []func(*jwt.Token) (interface{}, error)
	expired        int
	refreshExpired int
	tokenType      string
}

type Option func(*options)

func SetSigningMethod(method jwt.SigningMethod) Option {
	return func(o *options) {
		o.signingMethod = method
	}
}

func SetSigningKey(key, oldKey string) Option {
	return func(o *options) {
		o.signingKey = []byte(key)
		if oldKey != "" && key != oldKey {
			o.signingKey2 = []byte(oldKey)
		}
	}
}

func SetExpired(expired int) Option {
	return func(o *options) {
		o.expired = expired
	}
}

func SetRefreshExpired(expired int) Option {
	return func(o *options) {
		o.refreshExpired = expired
	}
}

func New(store Storer, opts ...Option) Auther {
	o := options{
		tokenType:      "Bearer",
		expired:        7200,
		refreshExpired: 2592000, // 30 days
		signingMethod:  jwt.SigningMethodHS512,
		signingKey:     []byte(defaultKey),
	}

	for _, opt := range opts {
		opt(&o)
	}

	o.keyFuncs = append(o.keyFuncs, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return o.signingKey, nil
	})

	if o.signingKey2 != nil {
		o.keyFuncs = append(o.keyFuncs, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, ErrInvalidToken
			}
			return o.signingKey2, nil
		})
	}

	return &JWTAuth{
		opts:  &o,
		store: store,
	}
}

type JWTAuth struct {
	opts  *options
	store Storer
}

// CustomClaims includes standard claims plus custom fields for refresh token
type CustomClaims struct {
	jwt.StandardClaims
	Type   string `json:"type,omitempty"`   // "access" or "refresh"
	Tenant string `json:"tenant,omitempty"` // Tenant ID
}

func (a *JWTAuth) GenerateToken(ctx context.Context, subject string) (TokenInfo, error) {
	return a.GenerateTokenWithExpired(ctx, subject, a.opts.expired)
}

func (a *JWTAuth) GenerateTokenWithExpired(ctx context.Context, subject string, expired int) (TokenInfo, error) {
	now := time.Now()
	expiresAt := now.Add(time.Duration(expired) * time.Second).Unix()

	token := jwt.NewWithClaims(a.opts.signingMethod, &CustomClaims{
		StandardClaims: jwt.StandardClaims{
			IssuedAt:  now.Unix(),
			ExpiresAt: expiresAt,
			NotBefore: now.Unix(),
			Subject:   subject,
		},
		Type: "access",
	})

	tokenStr, err := token.SignedString(a.opts.signingKey)
	if err != nil {
		return nil, err
	}

	tokenInfo := &tokenInfo{
		ExpiresAt:   expiresAt,
		TokenType:   a.opts.tokenType,
		AccessToken: tokenStr,
	}
	return tokenInfo, nil
}

func (a *JWTAuth) GenerateRefreshToken(ctx context.Context, subject string, tenant string) (RefreshTokenInfo, error) {
	now := time.Now()
	expiresAt := now.Add(time.Duration(a.opts.refreshExpired) * time.Second).Unix()
	jti := xid.New().String() // Unique token ID for revocation

	token := jwt.NewWithClaims(a.opts.signingMethod, &CustomClaims{
		StandardClaims: jwt.StandardClaims{
			Id:        jti,
			IssuedAt:  now.Unix(),
			ExpiresAt: expiresAt,
			NotBefore: now.Unix(),
			Subject:   subject,
		},
		Type:   "refresh",
		Tenant: tenant,
	})

	tokenStr, err := token.SignedString(a.opts.signingKey)
	if err != nil {
		return nil, err
	}

	return &refreshTokenInfo{
		RefreshToken: tokenStr,
		ExpiresAt:    expiresAt,
	}, nil
}

func (a *JWTAuth) parseToken(tokenStr string) (*jwt.StandardClaims, error) {
	var (
		token *jwt.Token
		err   error
	)

	for _, keyFunc := range a.opts.keyFuncs {
		token, err = jwt.ParseWithClaims(tokenStr, &jwt.StandardClaims{}, keyFunc)
		if err != nil || token == nil || !token.Valid {
			continue
		}
		break
	}

	if err != nil || token == nil || !token.Valid {
		return nil, ErrInvalidToken
	}

	return token.Claims.(*jwt.StandardClaims), nil
}

func (a *JWTAuth) parseCustomClaims(tokenStr string) (*CustomClaims, error) {
	var (
		token *jwt.Token
		err   error
	)

	for _, keyFunc := range a.opts.keyFuncs {
		token, err = jwt.ParseWithClaims(tokenStr, &CustomClaims{}, keyFunc)
		if err != nil || token == nil || !token.Valid {
			continue
		}
		break
	}

	if err != nil || token == nil || !token.Valid {
		return nil, ErrInvalidToken
	}

	return token.Claims.(*CustomClaims), nil
}

func (a *JWTAuth) ParseRefreshToken(ctx context.Context, refreshToken string) (string, string, string, error) {
	if refreshToken == "" {
		return "", "", "", ErrInvalidToken
	}

	claims, err := a.parseCustomClaims(refreshToken)
	if err != nil {
		return "", "", "", err
	}

	// Check token type
	if claims.Type != "refresh" {
		return "", "", "", ErrInvalidToken
	}

	// Check if token is revoked
	err = a.callStore(func(store Storer) error {
		revokedKey := "revoked:refresh:" + claims.Id
		if exists, err := store.Check(ctx, revokedKey); err != nil {
			return err
		} else if exists {
			return ErrTokenRevoked
		}
		return nil
	})
	if err != nil {
		return "", "", "", err
	}

	return claims.Subject, claims.Tenant, claims.Id, nil
}

func (a *JWTAuth) callStore(fn func(Storer) error) error {
	if store := a.store; store != nil {
		return fn(store)
	}
	return nil
}

func (a *JWTAuth) DestroyToken(ctx context.Context, tokenStr string) error {
	claims, err := a.parseToken(tokenStr)
	if err != nil {
		return err
	}

	return a.callStore(func(store Storer) error {
		expired := time.Until(time.Unix(claims.ExpiresAt, 0))
		return store.Set(ctx, tokenStr, expired)
	})
}

func (a *JWTAuth) DestroyRefreshToken(ctx context.Context, refreshToken string) error {
	claims, err := a.parseCustomClaims(refreshToken)
	if err != nil {
		return err
	}

	return a.callStore(func(store Storer) error {
		revokedKey := "revoked:refresh:" + claims.Id
		expired := time.Until(time.Unix(claims.ExpiresAt, 0))
		return store.Set(ctx, revokedKey, expired)
	})
}

func (a *JWTAuth) ParseSubject(ctx context.Context, tokenStr string) (string, error) {
	if tokenStr == "" {
		return "", ErrInvalidToken
	}

	claims, err := a.parseToken(tokenStr)
	if err != nil {
		return "", err
	}

	err = a.callStore(func(store Storer) error {
		if exists, err := store.Check(ctx, tokenStr); err != nil {
			return err
		} else if exists {
			return ErrInvalidToken
		}
		return nil
	})
	if err != nil {
		return "", err
	}

	return claims.Subject, nil
}

func (a *JWTAuth) Release(ctx context.Context) error {
	return a.callStore(func(store Storer) error {
		return store.Close(ctx)
	})
}
