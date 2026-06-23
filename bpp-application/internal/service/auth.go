package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	db        *pgxpool.Pool
	jwtSecret []byte
}

func NewAuthService(db *pgxpool.Pool) *AuthService {
	secret := os.Getenv("BPP_JWT_SECRET")
	if secret == "" {
		secret = "change-me-in-production"
	}
	return &AuthService{db: db, jwtSecret: []byte(secret)}
}

type ProviderAccount struct {
	ID          int64
	CompanyName string
	OJKLicense  string
	Email       string
}

func (s *AuthService) Register(ctx context.Context, companyName, ojkLicense, email, password string) (*ProviderAccount, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	var acc ProviderAccount
	err = s.db.QueryRow(ctx,
		`INSERT INTO provider_accounts (company_name, ojk_license, email, password_hash)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, company_name, ojk_license, email`,
		companyName, ojkLicense, email, string(hash),
	).Scan(&acc.ID, &acc.CompanyName, &acc.OJKLicense, &acc.Email)
	if err != nil {
		return nil, fmt.Errorf("insert account: %w", err)
	}
	return &acc, nil
}

func (s *AuthService) Login(ctx context.Context, email, password string) (string, error) {
	var id int64
	var companyName, hash string
	err := s.db.QueryRow(ctx,
		`SELECT id, company_name, password_hash FROM provider_accounts WHERE email = $1`,
		email,
	).Scan(&id, &companyName, &hash)
	if err != nil {
		return "", errors.New("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return "", errors.New("invalid credentials")
	}

	claims := jwt.MapClaims{
		"sub":          fmt.Sprintf("%d", id),
		"company_name": companyName,
		"email":        email,
		"exp":          time.Now().Add(24 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return signed, nil
}

func (s *AuthService) ValidateToken(tokenStr string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return nil, errors.New("invalid token")
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}
	return claims, nil
}
