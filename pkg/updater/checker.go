package updater

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"

	"golang.org/x/mod/semver"
)

// CheckForUpdate verifica si hay una actualización disponible
// Retorna:
//   - hasUpdate: true si hay una versión nueva disponible
//   - newVersion: string con la nueva versión (vacío si no hay actualización)
//   - error: error si hubo problemas descargando o parseando el manifiesto
func (u *Updater) CheckForUpdate() (bool, string, error) {
	// Construir URL del manifiesto según la arquitectura
	arch := runtime.GOARCH
	manifestURL := fmt.Sprintf("%sdarwin-%s.json", u.config.SourceURL, arch)

	// Descargar manifiesto
	resp, err := http.Get(manifestURL)
	if err != nil {
		return false, "", fmt.Errorf("error descargando manifiesto: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, "", fmt.Errorf("error HTTP %d descargando manifiesto", resp.StatusCode)
	}

	// Leer contenido
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, "", fmt.Errorf("error leyendo manifiesto: %w", err)
	}

	// Parsear JSON
	var manifest Manifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		return false, "", fmt.Errorf("error parseando manifiesto: %w", err)
	}

	// Guardar manifiesto para uso posterior
	u.manifest = &manifest

	// Comparar versiones
	hasUpdate, err := isNewerVersion(u.config.CurrentVersion, manifest.Version)
	if err != nil {
		// Si hay error en comparación semántica, comparar como strings
		hasUpdate = manifest.Version != u.config.CurrentVersion
	}

	if hasUpdate {
		return true, manifest.Version, nil
	}

	return false, "", nil
}

// isNewerVersion compara dos versiones semánticas
// Retorna true si newVersion es mayor que currentVersion
func isNewerVersion(currentVersion, newVersion string) (bool, error) {
	// Asegurar que las versiones tienen el prefijo "v" para semver
	current := currentVersion
	if current != "" && current[0] != 'v' {
		current = "v" + current
	}

	new := newVersion
	if new != "" && new[0] != 'v' {
		new = "v" + new
	}

	// Validar que ambas son versiones semánticas válidas
	if !semver.IsValid(current) {
		return false, fmt.Errorf("versión actual inválida: %s", currentVersion)
	}
	if !semver.IsValid(new) {
		return false, fmt.Errorf("versión nueva inválida: %s", newVersion)
	}

	// Comparar: retorna 1 si new > current, 0 si iguales, -1 si new < current
	comparison := semver.Compare(new, current)
	return comparison > 0, nil
}
