package updater

// updateScriptTemplate es el template del script de actualización
// Variables que se inyectan:
//   - PID: ID del proceso actual
//   - NEW_APP_PATH: Ruta del nuevo bundle .app descomprimido
//   - CURRENT_APP_PATH: Ruta del bundle .app actual (a reemplazar)
//   - OLD_APP_BACKUP: Ruta temporal para backup del bundle viejo
//   - ZIP_PATH: Ruta del archivo ZIP descargado
const updateScriptTemplate = `#!/bin/bash
set -e

# Variables inyectadas
PID="{{PID}}"
NEW_APP_PATH="{{NEW_APP_PATH}}"
CURRENT_APP_PATH="{{CURRENT_APP_PATH}}"
OLD_APP_BACKUP="{{OLD_APP_BACKUP}}"
ZIP_PATH="{{ZIP_PATH}}"

# Función de logging
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1"
}

# Función de rollback
rollback() {
    log "ERROR: Realizando rollback..."
    if [ -d "$OLD_APP_BACKUP" ]; then
        mv "$OLD_APP_BACKUP" "$CURRENT_APP_PATH" 2>/dev/null || true
        log "Rollback completado"
    fi
    exit 1
}

# Capturar errores y hacer rollback
trap rollback ERR

log "=== Iniciando actualización ==="
log "PID a esperar: $PID"
log "Nueva app: $NEW_APP_PATH"
log "App actual: $CURRENT_APP_PATH"

# 1. Esperar a que el proceso antiguo termine
log "Esperando a que el proceso $PID termine..."
while kill -0 "$PID" 2>/dev/null; do
    sleep 0.5
done
log "Proceso $PID terminado"

# Pequeña pausa adicional para asegurar que los recursos se liberaron
sleep 1

# 2. Sanitización - Limpiar atributos de cuarentena (Gatekeeper)
log "Limpiando atributos de cuarentena..."
xattr -d -r com.apple.quarantine "$NEW_APP_PATH" 2>/dev/null || true
log "Atributos de cuarentena limpiados"

# 3. Verificar que la nueva app existe
if [ ! -d "$NEW_APP_PATH" ]; then
    log "ERROR: La nueva app no existe en $NEW_APP_PATH"
    exit 1
fi

# 4. Swap Atómico
log "Realizando swap atómico..."

# Mover app actual a backup
if [ -d "$CURRENT_APP_PATH" ]; then
    log "Moviendo app actual a backup..."
    mv "$CURRENT_APP_PATH" "$OLD_APP_BACKUP"
fi

# Mover nueva app al destino
log "Instalando nueva app..."
mv "$NEW_APP_PATH" "$CURRENT_APP_PATH"

log "Swap completado exitosamente"

# 5. Limpieza
log "Limpiando archivos temporales..."

# Eliminar backup
if [ -d "$OLD_APP_BACKUP" ]; then
    rm -rf "$OLD_APP_BACKUP"
    log "Backup eliminado"
fi

# Eliminar ZIP descargado
if [ -f "$ZIP_PATH" ]; then
    rm -f "$ZIP_PATH"
    log "ZIP eliminado"
fi

# 6. Reiniciar la aplicación
log "Iniciando nueva versión de la aplicación..."
sleep 1
open -n "$CURRENT_APP_PATH"

log "=== Actualización completada exitosamente ==="
`
