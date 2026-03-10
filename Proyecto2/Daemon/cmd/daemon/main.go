package main

import (
	"log"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"

	"Daemon/interior/docker"
	lectorDelProc "Daemon/interior/lectorDelProc"
	"Daemon/interior/modelos"
	"Daemon/interior/storage"

	"github.com/robfig/cron/v3"
)

const (
	// Configuración
	ProcFile       = "/proc/continfo_pr2_so1_202300722"
	ValkeyAddr     = "localhost:6379"
	intervaloBucle = 30 * time.Second // Cada 30 segundos
)

func main() {
	log.Println("Iniciando Daemon...")

	// 1. Se inicializa la conexión a Valkey
	valkey := storage.NuevoClienteValkey(ValkeyAddr)
	log.Println("Se ha conectado correctamente a Valkey")

	// 2. Se inicializa el lector de /proc
	reader := lectorDelProc.NuevoLector()
	log.Printf("El lector de %s se ha inicializado correctamente", ProcFile)

	// 3. Se inicializa el gestor de Docker
	dockerMgr, err := docker.NuevoManager()
	if err != nil {
		log.Printf("El Docker no está disponible: %v", err)
	} else {
		log.Println("El gestor de Docker se ha inicializado correctamente")
	}

	// 4. Se configura el cronjob para los contenedores de prueba que son a cada 2 minutos
	c := cron.New()
	_, err = c.AddFunc("@every 2m", func() {
		log.Println("Ejecutando cronjob donde se están creando los contenedores de prueba...")
		if dockerMgr != nil {
			if err := dockerMgr.CrearContenedoresDePrueba(); err != nil {
				log.Printf("Hubieron problemas al crear los contenedores: %v", err)
			}
		}
	})

	if err != nil {
		log.Fatalf("Hubo problemas al crear el cronjob: %v", err)
	}

	c.Start()
	log.Println("El cronjob se ha iniciado, ésto cada 2 minutos")

	// 5. Bucle principal
	ticker := time.NewTicker(intervaloBucle)
	go func() {
		for range ticker.C {
			log.Println("Ejecutando el bucle...")

			// Se leen los datos del módulo del kernel
			procesos, informacionSistema, err := reader.LeerTodo()
			if err != nil {
				log.Printf("Hubieron problemas al leer el directorio /proc: %v", err)
				continue
			}

			log.Printf("Se han leído %d procesos, el RAM total es de %d KB", len(procesos), informacionSistema.TotalRAM)

			// Se guarda la info en Valkey
			if err := valkey.GuardarMetricasDelSistema(informacionSistema); err != nil {
				log.Printf("Hubieron problemas al guardar las métricas del sistema: %v", err)
			}

			if err := valkey.GuardarMetricasDeProceso(procesos); err != nil {
				log.Printf("Hubieron problemas al guardar las métricas de los procesos: %v", err)
			}

			// Se analiza y se decide que contenedores se van a eliminar
			if dockerMgr != nil {
				go analizaryEliminarContenedores(procesos, dockerMgr)
			}
		}
	}()

	log.Printf("El Daemon se ha iniciado. el bucle es cada %v", intervaloBucle)
	log.Println("Para detener ello presione las teclas Ctrl+C")

	// Se maneja la señal de terminación
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Deteniendo Daemon...")
	ticker.Stop()
	c.Stop()
	log.Println("El Daemon se ha detenido correctamente")
}

// Se analizan los procesos y decide qué contenedores se van a eliminar
func analizaryEliminarContenedores(procesos []modelos.ProcessInfo, dockerMgr *docker.Manager) {
	log.Println("Analizando contenedores para decidir cuáles eliminar...")

	// 1. Obtener mapa de PID a ContainerID usando el NUEVO MÉTODO
	pidAContainerID, err := dockerMgr.ObtenerMapaPIDContainerID()
	if err != nil {
		log.Printf("Error obteniendo mapa de contenedores: %v", err)
		return
	}
	log.Printf("Total contenedores activos con PID: %d", len(pidAContainerID))

	// 2. Obtener contenedores Docker para referencia (opcional)
	contenedoresDocker, err := dockerMgr.ObtenerContenedores()
	if err != nil {
		log.Printf("Error obteniendo contenedores Docker: %v", err)
	} else {
		// Usar la variable para algo, aunque sea solo un log
		log.Printf("Contenedores Docker encontrados: %d", len(contenedoresDocker))
	}

	// 3. DEPURACIÓN: Ver qué PIDs tenemos de Docker y del Kernel
	log.Println("--- INICIO DIAGNÓSTICO ---")

	// Mostrar los primeros 10 PIDs de Docker (para no saturar el log)
	dockerPIDs := make([]int, 0, len(pidAContainerID))
	for pid := range pidAContainerID {
		dockerPIDs = append(dockerPIDs, pid)
	}
	if len(dockerPIDs) > 10 {
		log.Printf("PIDs en Docker (primeros 10): %v", dockerPIDs[:10])
	} else {
		log.Printf("PIDs en Docker: %v", dockerPIDs)
	}

	// Mostrar los primeros 10 procesos que el kernel MARCA como contenedores
	kernelContainerPIDs := make([]int, 0)
	kernelContainerNames := make([]string, 0)
	for _, p := range procesos {
		if p.IsContainer {
			kernelContainerPIDs = append(kernelContainerPIDs, p.PID)
			kernelContainerNames = append(kernelContainerNames, p.Name)
		}
	}
	if len(kernelContainerPIDs) > 10 {
		log.Printf("PIDs del Kernel (marcados como contenedor, primeros 10): %v", kernelContainerPIDs[:10])
		log.Printf("Nombres del Kernel (primeros 10): %v", kernelContainerNames[:10])
	} else {
		log.Printf("PIDs del Kernel (marcados como contenedor): %v", kernelContainerPIDs)
		log.Printf("Nombres del Kernel: %v", kernelContainerNames)
	}
	log.Println("--- FIN DIAGNÓSTICO ---")

	// 3. Separar contenedores (solo los ACTIVOS)
	var containers []modelos.ProcessInfo
	for _, p := range procesos {
		// Por ahora, consideramos TODO lo que tenga PID en el mapa
		if _, existe := pidAContainerID[p.PID]; existe {
			// Se ignoran los procesos del sistema Docker
			if strings.Contains(p.Name, "docker-proxy") ||
				strings.Contains(p.Name, "containerd-shim") {
				continue
			}
			containers = append(containers, p)
			log.Printf("DEBUG - Contenedor detectado: PID=%d, Name=%s, RSS=%d KB",
				p.PID, p.Name, p.RSS_KB)
		}
	}

	log.Printf("Contenedores detectados (activos): %d", len(containers))

	// 4. Clasificar por consumo (con umbrales ajustados)
	var altoConsumo, bajoConsumo, otros []modelos.ProcessInfo

	for _, proc := range containers {
		// Mantener Grafana
		if strings.Contains(strings.ToLower(proc.Name), "grafana") ||
			strings.Contains(strings.ToLower(proc.Command), "grafana") {
			log.Printf("Grafana mantenido: PID=%d", proc.PID)
			continue
		}

		// UMBRALES AJUSTADOS para mejor clasificación
		if proc.RSS_KB > 30000 { // Más de 30MB
			altoConsumo = append(altoConsumo, proc)
		} else if proc.RSS_KB < 5000 { // Menos de 5MB
			bajoConsumo = append(bajoConsumo, proc)
		} else {
			otros = append(otros, proc)
		}
	}

	log.Printf("Clasificación - Alto: %d, Bajo: %d, Otros: %d",
		len(altoConsumo), len(bajoConsumo), len(otros))

	// 5. Aplicar lógica de negocio
	contenedoresAEliminar := decidirContenedoresAEliminar(altoConsumo, bajoConsumo, otros)

	if len(contenedoresAEliminar) == 0 {
		log.Println("No hay contenedores para eliminar")
		return
	}

	log.Printf("Eliminando %d contenedores...", len(contenedoresAEliminar))

	// 6. Eliminar contenedores usando el mapa
	for _, proceso := range contenedoresAEliminar {
		containerID := pidAContainerID[proceso.PID]

		if containerID == "" {
			// Intentar con el método alternativo
			id, err := dockerMgr.ObtenerIDPorPID(proceso.PID)
			if err == nil {
				containerID = id
			}
		}

		if containerID == "" {
			log.Printf("No se encontró ID para PID=%d (%s)", proceso.PID, proceso.Name)
			continue
		}

		log.Printf("Eliminando %s (PID=%d)", containerID[:12], proceso.PID)
		if err := dockerMgr.PararyRemoverContenedor(containerID); err != nil {
			log.Printf("Error eliminando %s: %v", containerID[:12], err)
		} else {
			log.Printf("Contenedor %s eliminado", containerID[:12])
		}
		time.Sleep(1 * time.Second)
	}
}

// Se implementa la lógica de negocio del proyecto
func decidirContenedoresAEliminar(altoConsumo, bajoConsumo, otros []modelos.ProcessInfo) []modelos.ProcessInfo {
	var aEliminar []modelos.ProcessInfo

	// 1. Se ordenan los contenedores por consumo de mayor a menor
	ordenarPorConsumo(altoConsumo)
	ordenarPorConsumo(bajoConsumo)
	ordenarPorConsumo(otros)

	// 2. Se decide para los contenedores para alto consumo (queremos mantener 2)
	if len(altoConsumo) > 2 {
		// Se eliminan los que son de alto consumo
		for i := 2; i < len(altoConsumo); i++ {
			aEliminar = append(aEliminar, altoConsumo[i])
		}

		log.Printf("Hay %d contenedores de alto consumo, se eliminarán %d contenedores", len(altoConsumo), len(altoConsumo)-2)
	} else if len(altoConsumo) < 2 {
		log.Printf("Se precisan 2 contenedores de alto consumo, por el momento solo hay %d contenedor/es", len(altoConsumo))
	}

	// 3. Se decide los que son de bajo consumo
	if len(bajoConsumo) > 3 {
		// Se ordena de mayor a menor
		ordenarPorConsumo(bajoConsumo)

		for i := 3; i < len(bajoConsumo); i++ {
			aEliminar = append(aEliminar, bajoConsumo[i])
		}

		log.Printf("Hay %d contenedores de bajo consumo, se eliminaran %d contenedores", len(bajoConsumo), len(bajoConsumo)-3)
	} else if len(bajoConsumo) < 3 {
		log.Printf("Se precisan 3 contenedores de bajo consumo, por el momento solo hay %d contenedor/es", len(bajoConsumo))
	}

	// 4. Los "otros" contenedores que son de consumo medio se eliminaran para mantener solo alto/bajo
	if len(otros) > 0 {
		log.Printf("%d contenedores de consumo medio serán eliminados", len(otros))
		aEliminar = append(aEliminar, otros...)
	}

	// 5. Se eliminarán duplicados para evitar problemas
	return eliminarDuplicados(aEliminar)
}

// Se ordenan los contenedores por RSS de mayor a menor
func ordenarPorConsumo(contenedores []modelos.ProcessInfo) {
	sort.Slice(contenedores, func(i, j int) bool {
		return contenedores[i].RSS_KB > contenedores[j].RSS_KB
	})
}

// Se remueven las entradas repetidas
func eliminarDuplicados(contenedores []modelos.ProcessInfo) []modelos.ProcessInfo {
	visto := make(map[int]bool)
	var resultado []modelos.ProcessInfo

	for _, c := range contenedores {
		if !visto[c.PID] {
			visto[c.PID] = true
			resultado = append(resultado, c)
		}
	}

	return resultado
}
