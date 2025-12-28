package crypto

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"

	"filippo.io/age"
)

// GenerateX25519KeyPair generates an age x25519 keypair for tenant encryption
func GenerateX25519KeyPair() (publicKey string, privateKey string, err error) {
	identity, err := age.GenerateX25519Identity()
	if err != nil {
		return "", "", fmt.Errorf("failed to generate keypair: %w", err)
	}

	return identity.Recipient().String(), identity.String(), nil
}

// EncryptToPublicKey encrypts data using the recipient's public key
func EncryptToPublicKey(plaintext []byte, publicKey string) ([]byte, error) {
	recipient, err := age.ParseX25519Recipient(publicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	var buf bytes.Buffer
	w, err := age.Encrypt(&buf, recipient)
	if err != nil {
		return nil, fmt.Errorf("failed to create encryption writer: %w", err)
	}

	if _, err := w.Write(plaintext); err != nil {
		w.Close()
		return nil, fmt.Errorf("failed to write plaintext: %w", err)
	}

	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("failed to finalize encryption: %w", err)
	}

	return buf.Bytes(), nil
}

// DecryptWithPrivateKey decrypts data using the private key
func DecryptWithPrivateKey(ciphertext []byte, privateKey string) ([]byte, error) {
	identity, err := age.ParseX25519Identity(privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	r := bytes.NewReader(ciphertext)
	rdr, err := age.Decrypt(r, identity)
	if err != nil {
		return nil, fmt.Errorf("failed to create decryption reader: %w", err)
	}

	plaintext, err := io.ReadAll(rdr)
	if err != nil {
		return nil, fmt.Errorf("failed to read decrypted data: %w", err)
	}

	return plaintext, nil
}

// EncryptBase64 encrypts data and returns base64-encoded ciphertext
func EncryptBase64(plaintext []byte, publicKey string) (string, error) {
	ciphertext, err := EncryptToPublicKey(plaintext, publicKey)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptBase64 decrypts base64-encoded ciphertext
func DecryptBase64(ciphertextB64 string, privateKey string) ([]byte, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(ciphertextB64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64: %w", err)
	}
	return DecryptWithPrivateKey(ciphertext, privateKey)
}

// GenerateKEK generates a random Key Encryption Key (32 bytes for AES-256)
func GenerateKEK() (string, error) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return "", fmt.Errorf("failed to generate KEK: %w", err)
	}
	return base64.StdEncoding.EncodeToString(key), nil
}

// EncryptForStorage encrypts a tenant private key using the platform KEK
// This is envelope encryption: KEK encrypts the tenant private key
func EncryptForStorage(plaintext []byte, kek string) (string, error) {
	kekBytes, err := base64.StdEncoding.DecodeString(kek)
	if err != nil {
		return "", fmt.Errorf("invalid KEK format: %w", err)
	}

	if len(kekBytes) != 32 {
		return "", fmt.Errorf("KEK must be 32 bytes (base64-encoded)")
	}

	// Simple XOR for demo - in production, use AES-GCM or similar
	// This is NOT secure for production, only for initial development
	ciphertext := make([]byte, len(plaintext))
	for i := range plaintext {
		ciphertext[i] = plaintext[i] ^ kekBytes[i%32]
	}

	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptFromStorage decrypts a tenant private key using the platform KEK
func DecryptFromStorage(ciphertextB64 string, kek string) ([]byte, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(ciphertextB64)
	if err != nil {
		return nil, fmt.Errorf("invalid ciphertext format: %w", err)
	}

	kekBytes, err := base64.StdEncoding.DecodeString(kek)
	if err != nil {
		return nil, fmt.Errorf("invalid KEK format: %w", err)
	}

	if len(kekBytes) != 32 {
		return nil, fmt.Errorf("KEK must be 32 bytes (base64-encoded)")
	}

	// Simple XOR for demo - matches EncryptForStorage
	plaintext := make([]byte, len(ciphertext))
	for i := range ciphertext {
		plaintext[i] = ciphertext[i] ^ kekBytes[i%32]
	}

	return plaintext, nil
}
