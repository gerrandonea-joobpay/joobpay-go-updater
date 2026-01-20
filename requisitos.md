# SPECIFICATION: joobpay-go-updater

## 1. GOAL
Desarrollar un sistema de actualización (Updater) en Go capaz de reemplazar un **Application Bundle (.app)** completo en macOS, superando las limitaciones de `go-selfupdate` que solo gestiona binarios.

## 2. CONTEXT
La librería se llamará `joobpay-go-updater`. Debe gestionar el ciclo de vida completo: empaquetado (Builder), verificación, descarga segura, reemplazo atómico y reinicio de la aplicación (Updater Client).

---

## 3. FUNCTIONAL REQUIREMENTS

### A. CLI Builder (Herramienta de Empaquetado)
La herramienta de línea de comandos debe permitir generar los artefactos de distribución.

**Inputs (Flags):**
* `--app-path`: Ruta al bundle `.app` original.
* `--version`: String de la nueva versión (ej: "1.0.1").
* `--output-name`: Nombre base del archivo de salida (ej: "joobpay-app").
* `--keychain-profile`: (Opcional) Nombre del perfil de credenciales para `xcrun notarytool`.

**Proceso de Build:**
1.  **Detección de Arquitectura:** Determinar si el target es `arm64` o `amd64`.
2.  **Manejo de Notarización (Si `--keychain-profile` no está vacío):**
    * Comprimir el bundle actual en un zip con sufijo: `{OutputName}.zip`.
    * Ejecutar envío a Apple: 
        `xcrun notarytool submit {OutputName}-no-notarized.zip --keychain-profile {Profile} --wait`
    * Si la notarización falla: Abortar y retornar error.
3.  **Empaquetado Final:**
    * Comprimir el bundle (`.app`) final (ya estampado si aplicó) en un archivo `.zip`.
    * Nombre del archivo: `{OutputName}.zip`.
4.  **Generación de Manifiesto (JSON):**
    * Calcular **SHA-256** del `{OutputName}.zip` final.
    * Generar archivo `darwin-arm64.json` o `darwin-amd64.json` con la estructura:
        ```json
        {
          "version": "1.0.1",
          "checksum": "a3b9c...", 
        }
        ```

### B. Updater Library (Cliente Integrado)
La librería que vive dentro de la aplicación.

**Configuración Inicial:**
* `CurrentVersion`: Versión actual de la app.
* `SourceURL`: URL base donde se alojan los archivos (ej: "https://s3.../miapp.zip").
* `DownloadPath`: Ruta local para descargas temporales (ej: `~/Library/Caches/...`).

**Funciones Expuestas:**

1.  **CheckForUpdate() -> (bool, error)**
    * Descarga el JSON correspondiente a la arquitectura del sistema.
    * Compara `json.version` vs `CurrentVersion`.
    * Retorna `true` si `json.version` es diferente (y mayor), `false` si es igual.

2.  **DownloadUpdate() -> (bool, error)**
    * **Trigger:** Función invocada manualmente por el desarrollador.
    * Verifica `CheckForUpdate` internamente.
    * Descarga el `.zip` desde la URL construida.
    * **Validación de Integridad (Crítico):**
        * Calcula el SHA-256 del archivo descargado en disco.
        * Compara contra `json.checksum`.
        * **Si difieren:** Borra el archivo inmediatamente, retorna `false` y un error de "Checksum Mismatch".
        * **Si coinciden:** Retorna `true`.

3.  **ApplyUpdate() -> (bool, error)**
    * **Trigger:** Función invocada manualmente por el desarrollador.
    * **Pre-validaciones:**
        * Verificar que `DownloadPath` existe.
        * Verificar que el `.zip` descargado existe.
    * **Preparación:**
        * Descomprimir el `.zip` en el mismo directorio de descarga.
        * Obtener la ruta absoluta de la App en ejecución (`executablePath`).
        * Obtener el PID del proceso actual (`os.Getpid()`).
    * **Ejecución:**
        * Construir el script de Shell dinámico.
        * Ejecutar el script como un proceso independiente (detached process) para que sobreviva al cierre de la App principal.

---

## 4. SHELL SCRIPT LOGIC (Update Application)

El script generado debe realizar las siguientes acciones de forma secuencial y atómica.

**Variables inyectadas al script:**
* `PID`: ID del proceso de la app vieja.
* `NEW_APP_PATH`: Ruta donde se descomprimió la nueva app.
* `CURRENT_APP_PATH`: Ruta donde está instalada la app actual.
* `OLD_APP_BACKUP`: Ruta temporal para mover la app vieja.

**Flujo del Script:**

1.  **Espera de Terminación:**
    * Loop que verifica si el `PID` sigue vivo. Esperar hasta que el proceso muera para evitar errores de "Archivo en uso".

2.  **Sanitización (Gatekeeper):**
    * Ejecutar `xattr -d -r com.apple.quarantine "$NEW_APP_PATH"` para limpiar atributos de seguridad de descarga.

3.  **Swap Atómico (Reemplazo):**
    * Mover la app actual a backup: `mv "$CURRENT_APP_PATH" "$OLD_APP_BACKUP"`
    * Mover la nueva app al destino: `mv "$NEW_APP_PATH" "$CURRENT_APP_PATH"`

4.  **Verificación y Rollback:**
    * Si el paso anterior falla, intentar restaurar: `mv "$OLD_APP_BACKUP" "$CURRENT_APP_PATH"`.

5.  **Limpieza:**
    * Si todo salió bien, eliminar el backup: `rm -rf "$OLD_APP_BACKUP"`
    * Eliminar el `.zip` descargado.

6.  **Reinicio (Launch):**
    * Ejecutar la nueva aplicación: `open -n "$CURRENT_APP_PATH"`