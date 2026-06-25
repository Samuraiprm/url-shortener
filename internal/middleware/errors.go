package middleware

import "errors"

var (
	ErrURLTooLong  = errors.New("url too long (max 2048 characters)")
	ErrInvalidURL  = errors.New("url contains blocked scheme")
)
