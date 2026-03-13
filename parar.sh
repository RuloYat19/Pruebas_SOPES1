#!/bin/bash
echo "Deteniendo el sistema..."

# Se detiene el servicio del daemon
sudo systemctl stop proyecto2-daemon.service

# Se descarga el módulo del kernel
sudo rmmod modulo 2>/dev/null && echo "Módulo de kernel descargado."

# Para parar los contenedores de Grafana y Valkey
# docker stop grafana valkey

echo "El sistema se ha detenido correctamente"