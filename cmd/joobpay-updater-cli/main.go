package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/gerrandonea-joobpay/joobpay-go-updater/internal/utils"
)

// Manifest representa la estructura del archivo JSON de manifiesto
type Manifest struct {
	Version  string `json:"version"`
	Checksum string `json:"checksum"`
}

func main() {
	var zipFileName string

	// Definir flags
	appPath := flag.String("app-path", "", "Ruta al bundle .app (requerido)")
	version := flag.String("version", "", "Versión de la actualización (requerido)")
	outputName := flag.String("output-name", "", "Nombre base del archivo de salida (opcional)")
	keychainProfile := flag.String("keychain-profile", "", "Perfil de Keychain para notarización (opcional)")
	outputDir := flag.String("output-dir", ".", "Directorio donde guardar los archivos generados (opcional)")

	flag.Parse()

	// Validar flags requeridos
	if *appPath == "" || *version == "" {
		fmt.Println("Error: --app-path y --version son requeridos")
		flag.Usage()
		os.Exit(1)
	}

	if *outputName == "" {
		// Usar el nombre del .app sin la extensión
		baseName := filepath.Base(*appPath)
		*outputName = strings.TrimSuffix(baseName, ".app")
	}

	// Verificar que el .app existe
	if _, err := os.Stat(*appPath); os.IsNotExist(err) {
		fmt.Printf("Error: el bundle .app no existe: %s\n", *appPath)
		os.Exit(1)
	}

	// Verificar que es un directorio .app
	if !strings.HasSuffix(*appPath, ".app") {
		fmt.Println("Error: --app-path debe apuntar a un bundle .app")
		os.Exit(1)
	}

	// Crear directorio de salida si no existe
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		fmt.Printf("Error creando directorio de salida: %v\n", err)
		os.Exit(1)
	}

	// Detectar arquitectura
	arch := runtime.GOARCH
	fmt.Printf("Arquitectura detectada: %s\n", arch)

	// Paso 1: Crear ZIP con el bundle adentro
	zipFileName = fmt.Sprintf("%s.zip", *outputName)

	zipFilePath := filepath.Join(*outputDir, zipFileName)
	fmt.Printf("Creando archivo ZIP: %s\n", zipFilePath)

	if err := utils.ZipDirectory(*appPath, zipFilePath); err != nil {
		fmt.Printf("Error creando ZIP: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("ZIP creado exitosamente")

	// Paso 2: Notarizar el ZIP (solo si se proporciona keychain-profile)
	if *keychainProfile != "" {
		fmt.Println("Iniciando proceso de notarización del ZIP...")

		if err := notarizeZip(zipFilePath, *keychainProfile); err != nil {
			fmt.Printf("Error en notarización: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("Notarización completada exitosamente")
	}

	// Paso 3: Calcular SHA-256 del ZIP final
	fmt.Println("Calculando checksum SHA-256...")
	checksum, err := utils.CalculateSHA256(zipFilePath)
	if err != nil {
		fmt.Printf("Error calculando checksum: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Checksum: %s\n", checksum)

	// Paso 4: Generar manifiesto JSON
	manifest := Manifest{
		Version:  *version,
		Checksum: checksum,
	}

	manifestFileName := fmt.Sprintf("darwin-%s.json", arch)
	manifestFilePath := filepath.Join(*outputDir, manifestFileName)
	fmt.Printf("Generando manifiesto: %s\n", manifestFilePath)

	manifestData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		fmt.Printf("Error generando JSON: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(manifestFilePath, manifestData, 0644); err != nil {
		fmt.Printf("Error escribiendo manifiesto: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Manifiesto generado exitosamente")
	fmt.Println("")
	fmt.Println("========================================")
	fmt.Println("BUILD COMPLETADO")
	fmt.Println("========================================")
	fmt.Printf("   ZIP: %s\n", zipFilePath)
	fmt.Printf("   Manifiesto: %s\n", manifestFilePath)
	fmt.Printf("   Versión: %s\n", *version)
	fmt.Printf("   Arquitectura: darwin-%s\n", arch)
	if *keychainProfile != "" {
		fmt.Println("   Notarizado: [✓]")
	} else {
		fmt.Println("   Notarizado: [✗]")
	}
	fmt.Println("========================================")
}

// notarizeZip envía el ZIP a Apple para notarización
func notarizeZip(zipPath, keychainProfile string) error {
	fmt.Printf("   ⏳ Enviando %s a Apple para notarización (esto puede tomar varios minutos)...\n", zipPath)

	cmd := exec.Command("xcrun", "notarytool", "submit",
		zipPath,
		"--keychain-profile", keychainProfile,
		"--wait",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("notarización fallida: %w", err)
	}

	fmt.Println("   ✅ Notarización aprobada por Apple")

	return nil
}
