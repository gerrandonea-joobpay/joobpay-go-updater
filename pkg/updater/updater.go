package updater

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Config contiene la configuración del Updater
type Config struct {
	// CurrentVersion es la versión actual de la aplicación
	CurrentVersion string

	// SourceURL es la URL base donde se alojan los archivos de actualización
	// Ejemplo: "https://s3.amazonaws.com/mybucket/updates/"
	SourceURL string

	// ZipFileName es el nombre del archivo ZIP a descargar
	// Ejemplo: "myapp.zip"``
	ZipFileName string

	// DownloadPath es la ruta local para descargas temporales
	// Ejemplo: "~/Library/Caches/myapp/updates/"
	DownloadPath string

	// StartAutomatically es un flag que indica si se debe iniciar la aplicación automáticamente
	StartAutomatically bool

	// AfterUpdateCommand es el comando que se ejecuta después de la actualización
	AfterUpdateCommand string

	// BeforeUpdateCommand es el comando que se ejecuta antes de la actualización
	BeforeUpdateCommand string
}

// Manifest representa la estructura del archivo JSON de manifiesto
type Manifest struct {
	Version  string `json:"version"`
	Checksum string `json:"checksum"`
}

// Updater gestiona el ciclo de vida de las actualizaciones
type Updater struct {
	config   Config
	manifest *Manifest
}

// New crea una nueva instancia del Updater
func New(config Config) *Updater {
	// Expandir ~ en DownloadPath
	if strings.HasPrefix(config.DownloadPath, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			config.DownloadPath = filepath.Join(home, config.DownloadPath[2:])
		}
	}

	// Asegurar que SourceURL termina con /
	if !strings.HasSuffix(config.SourceURL, "/") {
		config.SourceURL += "/"
	}

	return &Updater{
		config: config,
	}
}

// GetConfig retorna la configuración actual
func (u *Updater) GetConfig() Config {
	return u.config
}

// GetManifest retorna el manifiesto descargado (nil si no se ha verificado)
func (u *Updater) GetManifest() *Manifest {
	return u.manifest
}

// GetZipPath retorna la ruta del ZIP descargado
func (u *Updater) GetZipPath() string {
	return filepath.Join(u.config.DownloadPath, u.config.ZipFileName)
}

// ensureDownloadPath crea el directorio de descarga si no existe
func (u *Updater) ensureDownloadPath() error {
	if err := os.MkdirAll(u.config.DownloadPath, 0755); err != nil {
		return fmt.Errorf("error creando directorio de descarga: %w", err)
	}
	return nil
}
