#!/bin/bash
set -e

echo "Limpiando datos antes de comenzar con el Proyecto 2"

# Verificar que Docker está instalado
if ! command -v docker &> /dev/null; then
    echo "❌ Docker no está instalado. Instálalo primero."
    exit 1
fi

# 0. Limpieza de datos
cd ModuloKernel
sudo make clean 2>/dev/null || true
sudo rmmod modulo 2>/dev/null || true
cd ..

pkill -f daemon 2>/dev/null || true
echo "Limpieza completada, ahora vamos con las demás vainas xd"

# 1. Se cargar el módulo del kernel
echo "Cargando módulo del kernel..."
cd ModuloKernel
sudo make
sudo insmod modulo.ko
if [ $? -eq 0 ]; then
    echo "El módulo se ha cargado correctamente"
else
    echo "Hubo problemas al cargar el módulo"
    exit 1
fi
cd ..

# 2. Se verifica que el archivo /proc existe en la computadora
echo "Verificando si el archivo /proc/continfo_pr2_so1_202300722 existe..."
if [ -f /proc/continfo_pr2_so1_202300722 ]; then
    echo "El archivo /proc fue creado con éxito"
else
    echo "El archivo /proc no fue encontrado en la computadora"
    exit 1
fi

# 3. Iniciar Valkey
echo "Iniciando Valkey..."
if docker inspect valkey >/dev/null 2>&1; then
    docker start valkey
    echo "Se ha iniciado Valkey con el contenedor existente"
else
    docker run -d --name valkey -p 6379:6379 valkey/valkey
    echo "Valkey creado e iniciado"
fi

# 4. Iniciar Grafana
echo "Iniciando Grafana..."
if docker inspect grafana >/dev/null 2>&1; then
    docker start grafana
    echo "Se ha iniciado Grafana con el contenedor existente"
else
    docker run -d --name grafana -p 3000:3000 grafana/grafana
    echo "Grafana creado e iniciado"
fi

# 5. Compilar y ejecutar daemon
echo "Ejecutando el daemon..."
cd Daemon
go mod download 2>/dev/null || true
go build -o bin/daemon cmd/daemon/main.go
sudo ./bin/daemon
