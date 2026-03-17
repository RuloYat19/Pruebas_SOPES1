package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"Daemon/interior/modelos"

	"github.com/go-redis/redis/v8"
)

type ClienteValkey struct {
	cliente *redis.Client
	ctx     context.Context
}

func NuevoClienteValkey(addr string) *ClienteValkey {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: "",
		DB:       0,
	})

	return &ClienteValkey{
		cliente: rdb,
		ctx:     context.Background(),
	}
}

// Se guardan las métricas del sistema
func (v *ClienteValkey) GuardarMetricasDelSistema(informacion modelos.SystemInfo) error {
	marcaTiempo := time.Now().UnixMilli()

	// Se usa TimeSeries o simples claves
	pipe := v.cliente.Pipeline()

	pipe.HSet(v.ctx, "system:latest", map[string]interface{}{
		"total_ram": informacion.TotalRAM,
		"free_ram":  informacion.FreeRAM,
		"used_ram":  informacion.UsedRAM,
		"timestamp": marcaTiempo,
	})

	// Se guarda el histórico para gráficas de línea
	pipe.ZAdd(v.ctx, "system:ram:history", &redis.Z{
		Score:  float64(marcaTiempo),
		Member: informacion.UsedRAM,
	})

	_, err := pipe.Exec(v.ctx)
	return err
}

// Se guardan las métricas de procesos
func (v *ClienteValkey) GuardarMetricasDeProceso(procesos []modelos.ProcessInfo) error {
	marcaTiempo := time.Now().UnixMilli()

	pipe := v.cliente.Pipeline()

	// Top 5 por RAM
	for i, proceso := range v.obtenerTopN(procesos, "ram", 5) {
		key := fmt.Sprintf("top:ram:%d", i+1)
		data, _ := json.Marshal(proceso)
		pipe.Set(v.ctx, key, data, 0)
	}

	// Top 5 por CPU
	for i, proceso := range v.obtenerTopN(procesos, "cpu", 5) {
		key := fmt.Sprintf("top:cpu:%d", i+1)
		data, _ := json.Marshal(proceso)
		pipe.Set(v.ctx, key, data, 0)
	}

	// Contenedores eliminados para la gráfica de barras
	contenedores := v.filtrarContenedores(procesos)
	pipe.ZAdd(v.ctx, "containers:history", &redis.Z{
		Score:  float64(marcaTiempo),
		Member: len(contenedores),
	})

	// Se guardan todos los procesos para el análisis
	for _, proc := range procesos {
		if proc.IsContainer {
			data, _ := json.Marshal(proc)
			key := fmt.Sprintf("container:%d:%d", proc.PID, marcaTiempo)
			pipe.Set(v.ctx, key, data, 24*time.Hour)
		}
	}

	_, err := pipe.Exec(v.ctx)
	return err
}

// Se obtienen los N procesos con mayor consumo
func (v *ClienteValkey) obtenerTopN(procesos []modelos.ProcessInfo, metrica string, n int) []modelos.ProcessInfo {
	// Se crea una copia para no modificar original
	procs := make([]modelos.ProcessInfo, len(procesos))
	copy(procs, procesos)

	// Se ordena según métrica
	switch metrica {
	case "ram":
		for i := 0; i < len(procs)-1; i++ {
			for j := i + 1; j < len(procs); j++ {
				if procs[i].RSS_KB < procs[j].RSS_KB {
					procs[i], procs[j] = procs[j], procs[i]
				}
			}
		}
	case "cpu":
		for i := 0; i < len(procs)-1; i++ {
			for j := i + 1; j < len(procs); j++ {
				if procs[i].CPUPercent < procs[j].CPUPercent {
					procs[i], procs[j] = procs[j], procs[i]
				}
			}
		}
	}

	// Se retorna el top N
	if len(procs) > n {
		return procs[:n]
	}
	return procs
}

// Se filtran solo los procesos que son contenedores
func (v *ClienteValkey) filtrarContenedores(procesos []modelos.ProcessInfo) []modelos.ProcessInfo {
	var contenedores []modelos.ProcessInfo
	for _, p := range procesos {
		if p.IsContainer {
			contenedores = append(contenedores, p)
		}
	}
	return contenedores
}
