package util

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// SignMessage generates a SHA‑256 HMAC for the given message using the provided secret.
// The resulting signature is returned as a hex‑encoded string.
// It never returns a nil signature; an error is returned only if writing to the HMAC fails
// (which should never happen with the standard library implementation).
func SignMessage(secret, message []byte) (string, error) {
	mac := hmac.New(sha256.New, secret)
	if _, err := mac.Write(message); err != nil {
		return "", fmt.Errorf("failed to write message to HMAC: %w", err)
	}
	sum := mac.Sum(nil)
	return hex.EncodeToString(sum), nil
}

// VerifySignature checks that the supplied hex‑encoded signature matches the HMAC
// computed over the message with the given secret. The comparison is performed in
// constant time using hmac.Equal to mitigate timing attacks.
// It returns true when the signature is valid, false otherwise.
// An error is returned only if the supplied signature cannot be decoded from hex.
func VerifySignature(secret, message []byte, signatureHex string) (bool, error) {
	expected, err := hex.DecodeString(signatureHex)
	if err != nil {
		return false, fmt.Errorf("invalid hex signature: %w", err)
	}

	mac := hmac.New(sha256.New, secret)
	if _, err := mac.Write(message); err != nil {
		return false, fmt.Errorf("failed to write message to HMAC: %w", err)
	}
	actual := mac.Sum(nil)

	// hmac.Equal performs a constant‑time comparison.
	return hmac.Equal(actual, expected), nil
}

// VerifySignatureBytes performs the same verification as VerifySignature but
// accepts the signature already decoded as a byte slice. This is useful when the
// signature is transmitted in binary form (e.g., base64) and decoded by the caller.
func VerifySignatureBytes(secret, message, signature []byte) (bool, error) {
	if len(signature) == 0 {
		return false, fmt.Errorf("empty signature")
	}

	mac := hmac.New(sha256.New, secret)
	if _, err := mac.Write(message); err != nil {
		return false, fmt.Errorf("failed to write message to HMAC: %w", err)
	}
	actual := mac.Sum(nil)

	return hmac.Equal(actual, signature), nil
}