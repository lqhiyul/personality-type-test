package main

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

var errPasswordTooLong = errors.New("password must be 72 bytes or fewer")

func HashPassword(password string) (string, error) {
	if len([]byte(password)) > 72 {
		return "", errPasswordTooLong
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func CheckPasswordHash(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
