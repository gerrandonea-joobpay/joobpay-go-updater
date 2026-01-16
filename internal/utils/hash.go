package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

// CalculateSHA256 calcula el hash SHA-256 de un archivo
func CalculateSHA256(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("error abriendo archivo para hash: %w", err)
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("error calculando hash: %w", err)
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// VerifyChecksum verifica que el checksum de un archivo coincida con el esperado
func VerifyChecksum(filePath, expectedChecksum string) (bool, error) {
	actualChecksum, err := CalculateSHA256(filePath)
	if err != nil {
		return false, err
	}

	return actualChecksum == expectedChecksum, nil
}
