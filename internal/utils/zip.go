package utils

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ZipDirectory comprime un directorio (como un .app bundle) en un archivo ZIP
func ZipDirectory(sourcePath, destPath string) error {
	// Crear archivo ZIP
	zipFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("error creando archivo zip: %w", err)
	}
	defer zipFile.Close()

	writer := zip.NewWriter(zipFile)
	defer writer.Close()

	// Obtener el nombre base del directorio para preservar la estructura
	baseName := filepath.Base(sourcePath)

	// Recorrer el directorio y agregar archivos
	err = filepath.Walk(sourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Crear header del archivo
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return fmt.Errorf("error creando header: %w", err)
		}

		// Calcular la ruta relativa dentro del ZIP
		relPath, err := filepath.Rel(filepath.Dir(sourcePath), path)
		if err != nil {
			return fmt.Errorf("error calculando ruta relativa: %w", err)
		}
		header.Name = relPath

		// Si es directorio, agregar slash al final
		if info.IsDir() {
			header.Name += "/"
		} else {
			// Usar método de compresión Deflate para archivos
			header.Method = zip.Deflate
		}

		// Preservar permisos de ejecución
		header.SetMode(info.Mode())

		// Crear entrada en el ZIP
		entryWriter, err := writer.CreateHeader(header)
		if err != nil {
			return fmt.Errorf("error creando entrada zip: %w", err)
		}

		// Si es directorio, no hay contenido que escribir
		if info.IsDir() {
			return nil
		}

		// Copiar contenido del archivo
		file, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("error abriendo archivo: %w", err)
		}
		defer file.Close()

		_, err = io.Copy(entryWriter, file)
		if err != nil {
			return fmt.Errorf("error copiando contenido: %w", err)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("error recorriendo directorio: %w", err)
	}

	// Ignoramos baseName ya que lo usamos en relPath
	_ = baseName

	return nil
}

// UnzipFile descomprime un archivo ZIP en un directorio destino
func UnzipFile(zipPath, destPath string) error {
	// Abrir archivo ZIP
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("error abriendo zip: %w", err)
	}
	defer reader.Close()

	// Crear directorio destino si no existe
	if err := os.MkdirAll(destPath, 0755); err != nil {
		return fmt.Errorf("error creando directorio destino: %w", err)
	}

	// Extraer cada archivo
	for _, file := range reader.File {
		err := extractZipFile(file, destPath)
		if err != nil {
			return err
		}
	}

	return nil
}

// extractZipFile extrae un archivo individual del ZIP
func extractZipFile(file *zip.File, destPath string) error {
	// Construir ruta destino
	filePath := filepath.Join(destPath, file.Name)

	// Validar que no haya path traversal
	if !strings.HasPrefix(filePath, filepath.Clean(destPath)+string(os.PathSeparator)) {
		return fmt.Errorf("ruta inválida en zip: %s", file.Name)
	}

	// Si es directorio, crearlo
	if file.FileInfo().IsDir() {
		if err := os.MkdirAll(filePath, file.Mode()); err != nil {
			return fmt.Errorf("error creando directorio: %w", err)
		}
		return nil
	}

	// Crear directorios padre si no existen
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return fmt.Errorf("error creando directorios padre: %w", err)
	}

	// Crear archivo destino
	destFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
	if err != nil {
		return fmt.Errorf("error creando archivo: %w", err)
	}
	defer destFile.Close()

	// Abrir archivo del ZIP
	srcFile, err := file.Open()
	if err != nil {
		return fmt.Errorf("error abriendo archivo en zip: %w", err)
	}
	defer srcFile.Close()

	// Copiar contenido
	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		return fmt.Errorf("error extrayendo archivo: %w", err)
	}

	return nil
}
