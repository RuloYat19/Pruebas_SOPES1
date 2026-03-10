### Para revisar si Valkey lo está haciendo bien xd
# Entrar al contenedor de Valkey
docker exec -it valkey valkey-cli

# Dentro de valkey-cli, ver qué datos hay
KEYS *
HGETALL system:latest
ZRANGE system:ram:history 0 -1 WITHSCORES
exit

### Comandos del flujo de trabajo
# 1. Primera vez (crear estructura de carpetas)
mkdir -p grafana/provisioning/{datasources,dashboards}

# 2. Crear los archivos YAML y JSON (copiar el contenido de arriba)

# 3. Iniciar todo (servicios + daemon)
make dev

# 4. En otra terminal, ver logs
make docker-logs

# 5. Acceder a Grafana
# Abre http://localhost:3000
# Usuario: admin
# Contraseña: admin
# El dashboard "Proyecto 2" ya debería estar disponible

# 6. Cuando termines
make docker-down
make clean

### Manera rápida
# Verificar que los servicios están corriendo
docker-compose ps

# Probar conexión a Valkey
redis-cli -h localhost -p 6379 ping
# Debería responder: PONG

# Ver logs de Grafana (para asegurar que el plugin se instaló)
docker-compose logs grafana | grep -i plugin

### Ejecutar script con los comandos
cd /home/ruloyat/Escritorio/SistemasOperativos1/202300722_LAB_SO1_1S2026/Proyecto2
bash run.sh

### Comandos para ejecutar el módulo del Kernel
# 1. Cargar el módulo
sudo insmod modulo.ko

# 2. Verificar que está cargado
lsmod | grep modulo

# 3. Ver los mensajes del kernel
dmesg | tail

# 4. Ver el archivo creado en /proc
ls -la /proc/continfo_pr2_so1_202300722

# 5. LEER EL CONTENIDO (¡el momento más esperado!)
cat /proc/continfo_pr2_so1_202300722

# 6. Descargar el módulo (cuando termines)
sudo rmmod modulo

# Encabezados de lo que se observa
PID|Nombre|Comando/CONTAINER|VSZ(KB)|RSS(KB)|%MEM|%CPU
7770|cat  |              cat|5920   |1700   |0.01|0.00