# joobpay-go-updater

Sistema de actualización para macOS Application Bundles (.app) en Go.

## Características

- Actualización de bundles `.app` completos (no solo binarios)
- Notarización integrada con Apple
- Verificación de integridad SHA-256
- Reemplazo atómico con rollback automático
- Reinicio automático de la aplicación

## Instalación

```bash
go get github.com/gerrandonea-joobpay/joobpay-go-updater
```

## CLI Builder

Herramienta para empaquetar y preparar actualizaciones:

```bash
# Instalar CLI
go install github.com/gerrandonea-joobpay/joobpay-go-updater/cmd/joobpay-updater-cli@latest

# Uso básico (sin notarización)
joobpay-updater-cli \
  --app-path ./MyApp.app \
  --version 1.0.1

# Con notarización
joobpay-updater-cli \
  --app-path ./MyApp.app \
  --version 1.0.1 \
  --output-name myapp \
  --output-dir ./dist \
  --keychain-profile mac-dev
```

### Flags

| Flag | Descripción | Requerido |
|------|-------------|-----------|
| `--app-path` | Ruta al bundle `.app` | Sí |
| `--version` | Versión de la actualización | Sí |
| `--output-name` | Nombre base del archivo de salida | No (usa nombre del .app) |
| `--output-dir` | Directorio donde guardar los archivos | No (default: `.`) |
| `--keychain-profile` | Perfil de Keychain para notarización | No |

### Flujo del CLI

1. Crea el ZIP con el bundle `.app` adentro
2. Si `--keychain-profile` está presente, notariza el ZIP con Apple
3. Calcula el SHA-256 del ZIP
4. Genera el manifiesto JSON

### Salida

El CLI genera en el directorio especificado:
- `{output-name}.zip` - Bundle comprimido (notarizado si se especificó)
- `darwin-arm64.json` o `darwin-amd64.json` - Manifiesto con versión y checksum

## Updater Library

Librería para integrar en tu aplicación:

```go
package main

import (
    "fmt"
    "os"
    
    "github.com/gerrandonea-joobpay/joobpay-go-updater/pkg/updater"
)

func main() {
    // Configuración
    upd := updater.New(updater.Config{
        CurrentVersion: "1.0.0",
        SourceURL:      "https://my-bucket.s3.amazonaws.com/updates/",
        ZipFileName:    "myapp.zip",
        DownloadPath:   "~/Library/Caches/myapp/updates/",
    })

    // Verificar actualizaciones
    hasUpdate, newVersion, err := upd.CheckForUpdate()
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }

    if !hasUpdate {
        fmt.Println("No hay actualizaciones disponibles")
        return
    }

    fmt.Printf("Nueva versión disponible: %s\n", newVersion)

    // Descargar actualización
    if err := upd.DownloadUpdate(); err != nil {
        fmt.Printf("Error descargando: %v\n", err)
        return
    }

    // Aplicar actualización (ejecuta script detached y sale)
    if err := upd.ApplyUpdate(); err != nil {
        fmt.Printf("Error aplicando: %v\n", err)
        return
    }

    os.Exit(0)
}
```

## Estructura del Manifiesto JSON

```json
{
  "version": "1.0.1",
  "checksum": "a3b9c..."
}
```

## Proceso de Actualización

1. **CheckForUpdate**: Descarga `SourceURL/darwin-{arch}.json` y compara versiones
2. **DownloadUpdate**: Descarga `SourceURL/{ZipFileName}` y valida el checksum SHA-256
3. **ApplyUpdate**: Ejecuta un script detached que:
   - Espera que el proceso actual termine
   - Limpia atributos de cuarentena (Gatekeeper)
   - Realiza swap atómico del bundle
   - Hace rollback si hay errores
   - Reinicia la aplicación

## Workflow de Distribución

1. Compilar y firmar tu `.app`
2. Ejecutar el CLI para generar ZIP y manifiesto:
   ```bash
   joobpay-updater-cli \
     --app-path ./MyApp.app \
     --version 1.0.1 \
     --output-dir ./dist \
     --keychain-profile mac-dev
   ```
3. Subir `dist/MyApp.zip` y `dist/darwin-arm64.json` a tu S3/CDN
4. Los clientes detectarán y descargarán la actualización automáticamente

## Licencia

MIT
