package docker

import (
	"Daemon/interior/modelos"
	"context"
	"fmt"
	"log"
	"math/rand"
	"os/exec"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

type Manager struct {
	cli *client.Client
	ctx context.Context
}

func NuevoManager() (*Manager, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("Hubo problemas creando el cliente de Docker: %v", err)
	}

	return &Manager{
		cli: cli,
		ctx: context.Background(),
	}, nil
}

// Se obtiene el mapa de PID a ContainerID para contenedores activos
func (m *Manager) ObtenerMapaPIDContainerID() (map[int]string, error) {
	contenedores, err := m.cli.ContainerList(m.ctx, container.ListOptions{
		All: false, // Solo activos
	})
	if err != nil {
		return nil, fmt.Errorf("error listando contenedores: %v", err)
	}

	mapa := make(map[int]string)
	for _, c := range contenedores {
		if c.State == "running" {
			inspect, err := m.cli.ContainerInspect(m.ctx, c.ID)
			if err == nil && inspect.State.Pid > 0 {
				mapa[inspect.State.Pid] = c.ID
			}
		}
	}
	return mapa, nil
}

// Se obtiene el contenedor por PID
func (m *Manager) ObtenerIDPorPID(pid int) (string, error) {
	contenedores, err := m.cli.ContainerList(m.ctx, container.ListOptions{
		All: true,
	})
	if err != nil {
		return "", err
	}

	for _, c := range contenedores {
		inspect, err := m.cli.ContainerInspect(m.ctx, c.ID)
		if err == nil && inspect.State.Pid == pid {
			return c.ID, nil
		}
	}
	return "", fmt.Errorf("no se encontró contenedor con PID %d", pid)
}

// Se listan todos los contenedores
func (m *Manager) ObtenerContenedores() ([]container.Summary, error) {
	containers, err := m.cli.ContainerList(m.ctx, container.ListOptions{All: true})
	if err != nil {
		return nil, fmt.Errorf("Hubo problemas al listar los contenedores: %v", err)
	}
	return containers, nil
}

// Se detiene y elimina un contenedor
func (m *Manager) PararyRemoverContenedor(containerID string) error {
	// Se detiene el contenedor usando StopOptions
	stopOpts := container.StopOptions{}

	if err := m.cli.ContainerStop(m.ctx, containerID, stopOpts); err != nil {
		return fmt.Errorf("Hubo problemas al detener el contenedor %s: %v", containerID, err)
	}

	// Se elimina el contenedor usando RemoveOptions
	removeOpts := container.RemoveOptions{
		RemoveVolumes: true,
		Force:         true,
	}

	if err := m.cli.ContainerRemove(m.ctx, containerID, removeOpts); err != nil {
		return fmt.Errorf("Hubo problemas al eliminar el contenedor %s: %v", containerID, err)
	}

	return nil
}

// Se crean 5 contenedores aleatorios según el cronjob
func (m *Manager) CrearContenedoresDePrueba() error {
	log.Println("Iniciando con la creación de los 5 contenedores de prueba...")

	// Se definen las imágenes disponibles como indica el enunciado
	imagenes := []struct {
		nombre      string
		comando     string
		categoria   string
		descripcion string
	}{
		{
			nombre:      "roldyoran/go-client",
			comando:     "",
			categoria:   "alto",
			descripcion: "Alto consumo de la RAM",
		},
		{
			nombre:  "alpine",
			comando: "sh -c 'while true; do echo '2^20' | bc > /dev/null; sleep 2; done'",
			//comando:     "sh -c 'apk add --no-cache bc > /dev/null 2>&1 && while true; do echo \"2^20\" | bc > /dev/null; sleep 2; done'",
			categoria:   "alto",
			descripcion: "Alto consumo del CPU",
		},
		{
			nombre:      "alpine",
			comando:     "sleep 240",
			categoria:   "bajo",
			descripcion: "Bajo consumo",
		},
	}

	// Se crea un generador aleatorio nuevo para cada ejecución
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	contenedoresCreados := 0
	var errores []string

	// Se crean 5 contenedores aleatorios
	for i := 0; i < 5; i++ {
		// Se selecciona la imagen aleatoria usando el generador
		idx := r.Intn(len(imagenes))
		img := imagenes[idx]

		// Se genera el nombre único usando timestamp
		timestamp := time.Now().UnixNano()
		nombreContenedor := fmt.Sprintf("test-%s-%d-%d", img.categoria, timestamp, i)

		// Se prepara el comando
		args := []string{"run", "-d"}
		args = append(args, "--label", fmt.Sprintf("proyecto2.categoria=%s", img.categoria))
		args = append(args, "--label", fmt.Sprintf("proyecto2.timestamp=%d", time.Now().Unix()))
		args = append(args, "--name", nombreContenedor)
		args = append(args, "--restart", "no")

		// Se añade la imagen
		args = append(args, img.nombre)

		// Se añade el comando si existe
		if img.comando != "" {
			cmdParts := strings.Fields(img.comando)
			args = append(args, cmdParts...)
		}

		// Se ejecuta el comando "docker run"
		log.Printf("Creando el contenedor %d/%d: %s (%s)", i+1, 5, img.descripcion, img.categoria)

		output, err := exec.Command("docker", args...).CombinedOutput()

		if err != nil {
			errorMsg := fmt.Sprintf("Hubo problemas al crear el contenedor %s: %v - %s",
				img.nombre, err, string(output))
			log.Println(errorMsg)
			errores = append(errores, errorMsg)
		} else {
			log.Printf("Contenedor creado correctamente: %s (%s)",
				strings.TrimSpace(string(output)), nombreContenedor)
			contenedoresCreados++
		}

		time.Sleep(500 * time.Millisecond)
	}

	log.Printf("Fueron creados con éxito los %d/5 contenedores", contenedoresCreados)

	if len(errores) > 0 {
		return fmt.Errorf("se completó todo con %d errores", len(errores))
	}

	return nil
}

// Se busca el ID de Docker correspondiente a un proceso (versión mejorada)
func (m *Manager) EncontrarIDContenedor(proc modelos.ProcessInfo, contenedores []container.Summary) string {
	for _, c := range contenedores {
		// Búsqueda por nombre de imagen (sin tag)
		imagenLimpia := strings.Split(c.Image, ":")[0] // quita tag si existe
		nombreProc := strings.ToLower(proc.Name)

		if strings.Contains(strings.ToLower(imagenLimpia), nombreProc) {
			return c.ID
		}

		// Búsqueda por nombres del contenedor
		for _, name := range c.Names {
			// Docker agrega '/' al inicio del nombre
			nombreLimpio := strings.TrimPrefix(name, "/")
			if strings.Contains(strings.ToLower(nombreLimpio), nombreProc) {
				return c.ID
			}
		}

		// Búsqueda por labels (útil para contenedores de prueba)
		if categoria, ok := c.Labels["proyecto2.categoria"]; ok {
			if strings.Contains(strings.ToLower(proc.Name), strings.ToLower(categoria)) {
				return c.ID
			}
		}
	}
	return ""
}

// Se inspecciona el contenedor para más precisión
func (m *Manager) EncontrarIDContenedorPorPID(proc modelos.ProcessInfo) (string, error) {
	contenedores, err := m.cli.ContainerList(m.ctx, container.ListOptions{All: true})
	if err != nil {
		return "", fmt.Errorf("error listando contenedores: %v", err)
	}

	for _, c := range contenedores {
		// Se inspecciona el contenedor para obtener detalles
		inspect, err := m.cli.ContainerInspect(m.ctx, c.ID)
		if err != nil {
			continue
		}

		if inspect.State != nil && inspect.State.Pid == proc.PID {
			return c.ID, nil
		}

		// Se revisa si el nombre del proceso coincide
		if inspect.Config != nil {
			if strings.Contains(strings.ToLower(inspect.Config.Image), strings.ToLower(proc.Name)) {
				return c.ID, nil
			}
		}
	}
	return "", fmt.Errorf("no se encontró contenedor para PID %d", proc.PID)
}
