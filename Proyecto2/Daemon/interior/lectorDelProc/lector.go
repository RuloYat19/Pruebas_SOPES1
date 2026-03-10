package lectorDelProc

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"Daemon/interior/modelos"
)

const (
	rutaArchivoProc = "/proc/continfo_pr2_so1_202300722"
)

// Lector que maneja la lectura del archivo /proc
type Lector struct {
	rutaArchivo string
}

// Se crea un nuevo lector
func NuevoLector() *Lector {
	return &Lector{
		rutaArchivo: rutaArchivoProc,
	}
}

// Se lee todo el archivo donde retorna procesos e info del sistema
func (r *Lector) LeerTodo() ([]modelos.ProcessInfo, modelos.SystemInfo, error) {
	archivo, err := os.Open(r.rutaArchivo)

	if err != nil {
		return nil, modelos.SystemInfo{}, fmt.Errorf("Hubo problemas al abrir el archivo %s: %v", r.rutaArchivo, err)
	}

	defer archivo.Close()

	var procesos []modelos.ProcessInfo
	var informacionSistema modelos.SystemInfo
	scanner := bufio.NewScanner(archivo)

	seccion := "none" // Si esta vaina es system o processes

	for scanner.Scan() {
		linea := strings.TrimSpace(scanner.Text())

		if linea == "" {
			continue
		}

		// Se detectan secciones
		if linea == "MEMORIA RAM" {
			seccion = "system"
			continue
		} else if linea == "PROCESOS" {
			seccion = "processes"
			continue
		}

		// Se procesa según la sección
		switch seccion {
		case "system":
			r.parsearLineaSistema(linea, &informacionSistema)
		case "processes":
			if strings.HasPrefix(linea, "Total procesos identificados:") {
				// Se ignora la línea de total
				continue
			}

			proc := modelos.ProcessInfo{}
			if err := proc.ParseLine(linea); err == nil && proc.PID > 0 {
				procesos = append(procesos, proc)
			}
		}
	}

	return procesos, informacionSistema, scanner.Err()
}

// Se analizan las líneas de la sección Sistema
func (r *Lector) parsearLineaSistema(line string, info *modelos.SystemInfo) {
	// RAM Total
	if strings.Contains(line, "RAM en Total que tengo en la aguantadora xd:") {
		fmt.Sscanf(line, "RAM en Total que tengo en la aguantadora xd: %d KB", &info.TotalRAM)
	}
	// RAM Libre
	if strings.Contains(line, "RAM que esta Libre:") {
		fmt.Sscanf(line, "RAM que esta Libre: %d KB", &info.FreeRAM)
	}
	// RAM en Uso
	if strings.Contains(line, "RAM que esta en Uso:") {
		fmt.Sscanf(line, "RAM que esta en Uso: %d KB", &info.UsedRAM)
	}
}
