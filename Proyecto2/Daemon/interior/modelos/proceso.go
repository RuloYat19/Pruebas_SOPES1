package modelos

import (
	"strconv"
	"strings"
)

// Esto representa un proceso leído desde /proc
type ProcessInfo struct {
	PID           int     `json:"pid"`
	Name          string  `json:"name"`
	Command       string  `json:"command"`
	VSZ_KB        int64   `json:"vsz_kb"`      // Tamaño virtual en KB
	RSS_KB        int64   `json:"rss_kb"`      // Resident Set Size en KB
	MemoryPercent float64 `json:"mem_percent"` // Porcentaje de memoria
	CPUPercent    float64 `json:"cpu_percent"` // Porcentaje de CPU
	IsContainer   bool    `json:"is_container"`
}

// Esto representa información del sistema
type SystemInfo struct {
	TotalRAM int64 `json:"total_ram_kb"`
	FreeRAM  int64 `json:"free_ram_kb"`
	UsedRAM  int64 `json:"used_ram_kb"`
}

// Se convierte en una línea del archivo /proc en la estructura de ProcessInfo
func (p *ProcessInfo) ParseLine(line string) error {
	// El formato que se maneja es PID | Nombre | Comando | VSZ | RSS | %MEM | %CPU
	parts := strings.Split(strings.TrimSpace(line), "|")
	if len(parts) < 7 {
		return nil // Línea de encabezado o vacía
	}

	// PID
	pid, err := strconv.Atoi(parts[0])

	if err != nil {
		return err
	}

	p.PID = pid

	// Nombre
	p.Name = parts[1]

	// Comando/Contenedor
	p.Command = parts[2]
	p.IsContainer = (parts[2] == "CONTENEDOR")

	// VSZ en KB
	vsz, err := strconv.ParseInt(parts[3], 10, 64)

	if err == nil {
		p.VSZ_KB = vsz
	}

	// RSS en KB
	rss, err := strconv.ParseInt(parts[4], 10, 64)

	if err == nil {
		p.RSS_KB = rss
	}

	// %MEM
	memParts := strings.Split(parts[5], ".")

	if len(memParts) == 2 {
		entero, _ := strconv.ParseFloat(memParts[0], 64)
		decimal, _ := strconv.ParseFloat(memParts[1], 64)
		p.MemoryPercent = entero + decimal/100
	}

	// %CPU
	cpuParts := strings.Split(parts[6], ".")

	if len(cpuParts) == 2 {
		entero, _ := strconv.ParseFloat(cpuParts[0], 64)
		decimal, _ := strconv.ParseFloat(cpuParts[1], 64)
		p.CPUPercent = entero + decimal/100
	}

	return nil
}
