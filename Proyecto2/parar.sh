#!/bin/bash
echo "Deteniendo el sistema..."

# Se detiene el servicio del daemon
sudo systemctl stop proyecto2-daemon.service

# Se elimina el archivo de los logs del daemon
LOG_FILE="/var/log/proyecto2-daemon.log"

if [ -f "$LOG_FILE" ]; then
    echo "Eliminando logs anteriores: $LOG_FILE"
    sudo rm -f "$LOG_FILE"
else
    echo "No había logs previos."
fi

# Se descarga el módulo del kernel
sudo rmmod modulo 2>/dev/null && echo "Módulo de kernel descargado."

# Para parar los contenedores de Grafana y Valkey
# docker stop grafana valkey

echo "El sistema se ha detenido correctamente"