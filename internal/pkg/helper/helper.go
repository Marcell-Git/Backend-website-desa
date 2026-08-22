package helper

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"math"
	"net/http"
	"strconv"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash), err
}

func CheckPassword(hashed, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hashed), []byte(plain)) == nil
}

func HashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

func RandomTokenID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

type PageQuery struct {
	Page  int
	Limit int
	Sort  string
}

func ParsePagination(r *http.Request, defaultLimit int) PageQuery {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 {
		limit = defaultLimit
	}
	if limit > 100 {
		limit = 100
	}
	return PageQuery{Page: page, Limit: limit, Sort: r.URL.Query().Get("sort")}
}

func (p PageQuery) Offset() int {
	return (p.Page - 1) * p.Limit
}

func (p PageQuery) TotalPage(totalData int64) int {
	return int(math.Ceil(float64(totalData) / float64(p.Limit)))
}

func ParseIntDefault(s string, fallback int) int {
	if s == "" {
		return fallback
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return fallback
	}
	return n
}

func CleanSearch(s string) string {
	return strings.TrimSpace(s)
}

func IsDuplicateErrorGORM(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "23505")
}
