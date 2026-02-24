package auth

import (
	"fmt"
	"os"
	"time"

	"energy-monitoring-system/internal/db"

	"github.com/golang-jwt/jwt/v5"
)

func GenerateJWT(user *db.User) (string, error) {
	jwtSecret := []byte(os.Getenv("JWT_SECRET"))
	claim := jwt.MapClaims{
		"user_id":      user.ID,
		"meter_number": user.MeterNumber,
		"exp":          time.Now().Add(time.Hour * 24 * 7).Unix(),
		"iss":          "energy-monitoring-system",
		"sub":          "access",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claim)
	return token.SignedString(jwtSecret)
}

func ValidateJWT(tokenString string) (*jwt.Token, error) {
	jwtSecret := []byte(os.Getenv("JWT_SECRET"))
	return jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtSecret, nil
	})
}
