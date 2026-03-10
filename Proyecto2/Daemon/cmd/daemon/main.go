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

	// 1. Se separan los contenedores por categoría
	var containers []modelos.ProcessInfo
	for _, p := range procesos {
		if p.IsContainer {
			// Ignorar procesos del sistema Docker
			if strings.Contains(p.Name, "docker-proxy") ||
				strings.Contains(p.Name, "containerd-shim") ||
				strings.Contains(p.Name, "containerd") ||
				strings.Contains(p.Name, "dockerd") {
				continue
			}
			containers = append(containers, p)
		}
	}

	log.Printf("El total de contenedores detectados son %d", len(containers))

	// 2. Se obtiene la lista de  los contenedores de Docker reales
	contenedoresDocker, err := dockerMgr.ObtenerContenedores()
	if err != nil {
		log.Printf("Hubo problemas al obtener los contenedores de Docker: %v", err)
		return
	}
	log.Printf("EL total contenedores activos en Docker es de %d", len(contenedoresDocker))

	// 3. Se clasifican los contenedores por consumo
	var altoConsumo, bajoConsumo, otros []modelos.ProcessInfo

	for _, proc := range containers {
		// Se identifican si es Grafana para no eliminarla
		if strings.Contains(strings.ToLower(proc.Name), "grafana") ||
			strings.Contains(strings.ToLower(proc.Command), "grafana") {
			log.Printf("El contenedor de Grafana se ha mantenido: PID=%d, %s", proc.PID, proc.Name)
			continue
		}

		// Se clasifica según consumo usando el RSS como métrica principal
		if proc.RSS_KB > 50000 { // Más de 50MB de RAM
			altoConsumo = append(altoConsumo, proc)
		} else if proc.RSS_KB < 10000 { // Menos de 10MB de RAM
			bajoConsumo = append(bajoConsumo, proc)
		} else {
			otros = append(otros, proc)
		}
	}

	log.Printf("De alto consumo hay %d, de bajo consumo hay %d y otros hay %d", len(altoConsumo), len(bajoConsumo), len(otros))

	// 4. Se aplica la lógica de negocio del proyecto
	contenedoresAEliminar := decidirContenedoresAEliminar(altoConsumo, bajoConsumo, otros)

	if len(contenedoresAEliminar) == 0 {
		log.Println("No hay contenedores que eliminar en esta ocasión")
		return
	}

	log.Printf("Eliminando %d contenedores...", len(contenedoresAEliminar))

	// 5. Se eliminarán los contenedores
	for _, proceso := range contenedoresAEliminar {
		// Se busca el ID del contenedor en Docker
		containerID := dockerMgr.EncontrarIDContenedor(proceso, contenedoresDocker)
		if containerID == "" {
			// Se intenta con el método más preciso por si falla el primero por si las moscas
			id, err := dockerMgr.EncontrarIDContenedorPorPID(proceso)

			if err == nil {
				containerID = id
			}
		}

		if containerID == "" {
			log.Printf("Hubo problemas al encontrar ID para el contenedor PID=%d (%s). Por lo tanto, se pasa a omitir", proceso.PID, proceso.Name)
			continue
		}

		log.Printf("Eliminando contenedor %s con PID = %d, %s", containerID[:12], proceso.PID, proceso.Name)
		if err := dockerMgr.PararyRemoverContenedor(containerID); err != nil {
			log.Printf("Hubo problemas al eliminar el contenedor %s: %v", containerID[:12], err)
		} else {
			log.Printf("El Contenedor %s se ha eliminado correctamente.", containerID[:12])
		}
		time.Sleep(1 * time.Second) // Pequeña pausa entre eliminaciones para que no se buguee o pasen vainas raras xd
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
		// Se eliminan los que son de mayor consumo para ello se ordena de forma ascendente para que los primeros sean los de menor consumo
		sort.Slice(bajoConsumo, func(i, j int) bool {
			return bajoConsumo[i].RSS_KB < bajoConsumo[j].RSS_KB
		})

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
