package updater

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gerrandonea-joobpay/joobpay-go-updater/internal/utils"
)

// DownloadUpdate descarga la actualización y valida su integridad
// Retorna error si:
//   - No se ha llamado a CheckForUpdate() previamente
//   - La descarga falla
//   - El checksum no coincide (el archivo se elimina automáticamente)
func (u *Updater) DownloadUpdate() error {
	// Verificar que tenemos un manifiesto
	if u.manifest == nil {
		// Intentar obtener el manifiesto primero
		hasUpdate, _, err := u.CheckForUpdate()
		if err != nil {
			return fmt.Errorf("error verificando actualización: %w", err)
		}
		if !hasUpdate {
			return fmt.Errorf("no hay actualización disponible")
		}
	}

	// Asegurar que existe el directorio de descarga
	if err := u.ensureDownloadPath(); err != nil {
		return err
	}

	// Determinar nombre del archivo y ruta de descarga
	zipPath := filepath.Join(u.config.DownloadPath, u.config.ZipFileName)

	// Construir URL de descarga: SourceURL + ZipFileName
	downloadURL := u.config.SourceURL + u.config.ZipFileName

	// Descargar el archivo
	fmt.Printf("Descargando actualización desde: %s\n", downloadURL)
	if err := downloadFile(downloadURL, zipPath); err != nil {
		return fmt.Errorf("error descargando actualización: %w", err)
	}

	// Validar checksum SHA-256
	fmt.Println("Validando integridad del archivo...")
	valid, err := utils.VerifyChecksum(zipPath, u.manifest.Checksum)
	if err != nil {
		// Error calculando checksum, eliminar archivo
		os.Remove(zipPath)
		return fmt.Errorf("error validando checksum: %w", err)
	}

	if !valid {
		// Checksum no coincide, eliminar archivo inmediatamente
		os.Remove(zipPath)
		return fmt.Errorf("checksum mismatch: el archivo descargado está corrupto o fue manipulado")
	}

	fmt.Println("Checksum validado correctamente")

	return nil
}

// downloadFile descarga un archivo desde una URL y lo guarda en disco
func downloadFile(url, destPath string) error {
	// Crear archivo de destino
	out, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("error creando archivo de destino: %w", err)
	}
	defer out.Close()

	// Realizar petición HTTP
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("error en petición HTTP: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error HTTP %d descargando archivo", resp.StatusCode)
	}

	// Copiar contenido al archivo
	written, err := io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("error escribiendo archivo: %w", err)
	}

	fmt.Printf("Descargados %.2f MB\n", float64(written)/(1024*1024))

	return nil
}

// IsDownloaded verifica si ya existe una actualización descargada
func (u *Updater) IsDownloaded() bool {
	zipPath := u.GetZipPath()
	if zipPath == "" {
		return false
	}

	_, err := os.Stat(zipPath)
	return err == nil
}

// CleanDownload elimina el archivo de actualización descargado
func (u *Updater) CleanDownload() error {
	zipPath := u.GetZipPath()
	if zipPath == "" {
		return nil
	}

	if err := os.Remove(zipPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("error eliminando archivo de actualización: %w", err)
	}

	return nil
}
