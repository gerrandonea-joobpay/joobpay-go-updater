package utils

import (
	"fmt"
	"runtime"
)

// GetArchitecture retorna la arquitectura del sistema actual
// Retorna "arm64" o "amd64"
func GetArchitecture() string {
	return runtime.GOARCH
}

// GetManifestFilename retorna el nombre del archivo de manifiesto
// basado en la arquitectura actual (darwin-arm64.json o darwin-amd64.json)
func GetManifestFilename() string {
	return fmt.Sprintf("darwin-%s.json", GetArchitecture())
}

// GetZipFilename retorna el nombre del archivo ZIP
// basado en la arquitectura actual
func GetZipFilename(baseName string) string {
	return fmt.Sprintf("%s-darwin-%s.zip", baseName, GetArchitecture())
}
