package repos

import "math/rand/v2"

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func generateRandomString(length int) string {
	b := make([]byte, length)
	for i := range b {
		// rand.IntN safely retrieves a random index within the charset bounds
		b[i] = charset[rand.IntN(len(charset))]
	}
	return string(b)
}
