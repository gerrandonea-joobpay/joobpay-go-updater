package updater

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/joobpay/joobpay-go-updater/internal/utils"
)

// ApplyUpdate aplica la actualización descargada
// Este método:
//  1. Descomprime el ZIP en el directorio de descarga
//  2. Genera un script de shell para el reemplazo atómico
//  3. Ejecuta el script como proceso detached
//
// Después de llamar a este método, la aplicación debe salir con os.Exit(0)
// para permitir que el script complete el reemplazo.
func (u *Updater) ApplyUpdate() error {
	// Validar pre-condiciones
	if err := u.validateForApply(); err != nil {
		return err
	}

	// Descomprimir el ZIP
	extractPath := filepath.Join(u.config.DownloadPath, "extracted")
	fmt.Printf("Descomprimiendo actualización en: %s\n", extractPath)

	// Limpiar directorio de extracción si existe
	os.RemoveAll(extractPath)
	zipPath := u.GetZipPath()
	if err := utils.UnzipFile(zipPath, extractPath); err != nil {
		return fmt.Errorf("error descomprimiendo actualización: %w", err)
	}

	// Encontrar el .app dentro del directorio extraído
	newAppPath, err := findAppBundle(extractPath)
	if err != nil {
		return err
	}

	fmt.Printf("Bundle encontrado: %s\n", newAppPath)

	// Obtener información del proceso actual
	pid := os.Getpid()
	currentAppPath := u.config.AppPath

	// Generar ruta para backup
	backupPath := filepath.Join(u.config.DownloadPath, fmt.Sprintf("backup-%d.app", time.Now().Unix()))

	// Generar script
	script := generateUpdateScript(pid, newAppPath, currentAppPath, backupPath, zipPath)

	// Escribir script a archivo temporal
	scriptPath := filepath.Join(u.config.DownloadPath, "update-script.sh")
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		return fmt.Errorf("error escribiendo script de actualización: %w", err)
	}

	fmt.Printf("Script de actualización creado: %s\n", scriptPath)

	// Ejecutar script como proceso detached
	if err := executeDetached(scriptPath); err != nil {
		return fmt.Errorf("error ejecutando script de actualización: %w", err)
	}

	fmt.Println("Script de actualización iniciado. La aplicación debe cerrarse ahora.")

	return nil
}

// validateForApply valida que se puede aplicar la actualización
func (u *Updater) validateForApply() error {
	// Verificar que existe el directorio de descarga
	if _, err := os.Stat(u.config.DownloadPath); os.IsNotExist(err) {
		return fmt.Errorf("directorio de descarga no existe: %s", u.config.DownloadPath)
	}

	// Verificar que existe el ZIP descargado
	zipPath := u.GetZipPath()
	if zipPath == "" {
		return fmt.Errorf("no hay actualización descargada")
	}

	if _, err := os.Stat(zipPath); os.IsNotExist(err) {
		return fmt.Errorf("archivo de actualización no existe: %s", zipPath)
	}

	// Verificar que existe la app actual
	if _, err := os.Stat(u.config.AppPath); os.IsNotExist(err) {
		return fmt.Errorf("aplicación actual no existe: %s", u.config.AppPath)
	}

	return nil
}

// findAppBundle busca el bundle .app dentro de un directorio
func findAppBundle(searchPath string) (string, error) {
	var appPath string

	entries, err := os.ReadDir(searchPath)
	if err != nil {
		return "", fmt.Errorf("error leyendo directorio: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() && strings.HasSuffix(entry.Name(), ".app") {
			appPath = filepath.Join(searchPath, entry.Name())
			break
		}
	}

	if appPath == "" {
		return "", fmt.Errorf("no se encontró bundle .app en: %s", searchPath)
	}

	return appPath, nil
}

// generateUpdateScript genera el script de actualización con las variables inyectadas
func generateUpdateScript(pid int, newAppPath, currentAppPath, backupPath, zipPath string) string {
	script := updateScriptTemplate

	// Reemplazar variables
	script = strings.ReplaceAll(script, "{{PID}}", fmt.Sprintf("%d", pid))
	script = strings.ReplaceAll(script, "{{NEW_APP_PATH}}", newAppPath)
	script = strings.ReplaceAll(script, "{{CURRENT_APP_PATH}}", currentAppPath)
	script = strings.ReplaceAll(script, "{{OLD_APP_BACKUP}}", backupPath)
	script = strings.ReplaceAll(script, "{{ZIP_PATH}}", zipPath)

	return script
}

// executeDetached ejecuta un script como proceso completamente independiente
func executeDetached(scriptPath string) error {
	cmd := exec.Command("/bin/bash", scriptPath)

	// Configurar para que el proceso sea completamente independiente
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true, // Crear nueva sesión para que sobreviva al padre
	}

	// Redirigir salida a un archivo de log
	logPath := filepath.Join(filepath.Dir(scriptPath), "update.log")
	logFile, err := os.Create(logPath)
	if err != nil {
		return fmt.Errorf("error creando archivo de log: %w", err)
	}

	cmd.Stdout = logFile
	cmd.Stderr = logFile

	// Iniciar el proceso (no esperamos a que termine)
	if err := cmd.Start(); err != nil {
		logFile.Close()
		return fmt.Errorf("error iniciando script: %w", err)
	}

	// Liberar el proceso para que continúe independientemente
	if err := cmd.Process.Release(); err != nil {
		return fmt.Errorf("error liberando proceso: %w", err)
	}

	fmt.Printf("Script iniciado con PID: %d\n", cmd.Process.Pid)
	fmt.Printf("Log de actualización: %s\n", logPath)

	return nil
}
