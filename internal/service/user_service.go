package service

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

func containsUppercase(s string) bool {
	for _, char := range s {
		if char >= 'A' && char <= 'Z' {
			return true
		}
	}
	return false
}

func HashPassword(password string) string {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		fmt.Println("failed to hash password:", err)
		return ""
	}
	return string(hash)

}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil // если пароли совпадают, вернется nil
}

func ValidatePassword(password string) bool {
	if len(password) < 8 {
		return false
	}

	if !containsUppercase(password) {
		return false
	}
	return true
}
