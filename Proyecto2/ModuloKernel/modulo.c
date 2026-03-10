#include <linux/module.h>
#include <linux/kernel.h>
#include <linux/init.h>
#include <linux/proc_fs.h> 
#include <linux/seq_file.h>
#include <linux/sched.h>
#include <linux/mm.h>
#include <linux/sched/signal.h>
#include <linux/time.h>
#include <linux/ktime.h>
#include <linux/uaccess.h>
#include <linux/slab.h>

MODULE_LICENSE("GPL");
MODULE_AUTHOR("Raul Emanuel Yat Cancinos - 202300722");
MODULE_DESCRIPTION("Modulo de Kernel para Proyecto 2 - LSO1");
MODULE_VERSION("1.0");

#define PROC_NAME "continfo_pr2_so1_202300722"

// Estructura para almacenar información de un proceso
struct proc_info {
    unsigned int pid;
    unsigned int tgid;
    char comm[TASK_COMM_LEN];
    unsigned long vsz_kb;
    unsigned long rss_kb;
    unsigned long mem_percent;
    unsigned long cpu_percent;
    unsigned long long utime;
    unsigned long long stime;
    unsigned long long start_time;
};

static struct proc_dir_entry *proc_file;
static struct proc_info *g_processes;
static int g_num_procs;
static unsigned long total_ram;
static unsigned long free_ram;
static ktime_t last_cpu_calculation;

// Se obtiene la información de memoria RAM del sistema
static void obtenerInformacionDeMemoria(unsigned long *total, unsigned long *free)
{
    struct sysinfo si;
    si_meminfo(&si);
    
    *total = si.totalram * (si.mem_unit / 1024);
    *free = si.freeram * (si.mem_unit / 1024);
    
    pr_debug("RAM en Total: %lu KB, que esta Libre: %lu KB\n", *total, *free);
}

// Se calcula el porcentaje de CPU basado en tiempo de proceso y tiempo total
static unsigned long calcularPorcentajeCPU(struct task_struct *task)
{
    unsigned long long total_time, process_time;
    unsigned long cpu_percent = 0;
    static unsigned long long last_total_time = 0;
    static ktime_t last_time = 0;
    ktime_t current_time;
    s64 time_delta_ms;
    
    current_time = ktime_get();
    
    if (last_total_time == 0) {
        last_total_time = (unsigned long long)ktime_to_ms(current_time);
        last_time = current_time;
        return 0;
    }
    
    time_delta_ms = ktime_ms_delta(current_time, last_time);
    if (time_delta_ms == 0)
        return 0;
    
    process_time = jiffies_to_msecs(task->utime + task->stime);
    
    cpu_percent = (process_time * 100) / time_delta_ms;
    
    last_total_time = (unsigned long long)ktime_to_ms(current_time);
    last_time = current_time;
    
    return cpu_percent > 100 ? 100 : cpu_percent;
}

// Se obtiene VSZ (Virtual Memory Size) y RSS (Resident Set Size) en KB
static void obtenerProcesosEnMemoria(struct task_struct *task, unsigned long *vsz_kb, unsigned long *rss_kb)
{
    struct mm_struct *mm = task->mm;
    
    if (!mm) {
        *vsz_kb = 0;
        *rss_kb = 0;
        return;
    }
    
    *vsz_kb = (mm->total_vm * PAGE_SIZE) / 1024;
    
    *rss_kb = (get_mm_rss(mm) * PAGE_SIZE) / 1024;
    
    pr_debug("Proceso %d: VSZ=%lu KB, RSS=%lu KB\n", task->pid, *vsz_kb, *rss_kb);
}

// Se escanean todos los procesos y construye la información
static void escanearProcesos(struct seq_file *m)
{
    struct task_struct *task;
    unsigned long total_mem, free_mem;
    unsigned long vsz, rss;
    unsigned long cpu_perc;
    int count = 0;
    
    obtenerInformacionDeMemoria(&total_ram, &free_ram);
    
    seq_printf(m, "\n=== MEMORIA RAM EN MI COMPAÑERA DE BATALLA XD ===\n");
    seq_printf(m, "RAM en Total que tengo en la aguantadora xd: %lu KB\n", total_ram);
    seq_printf(m, "RAM que esta Libre: %lu KB\n", free_ram);
    seq_printf(m, "RAM que esta en Uso: %lu KB\n", total_ram - free_ram);
    seq_printf(m, "\n=== PROCESOS QUE HAY EN MI COMPAÑERA DE BATALLA XD ===\n");
    
    rcu_read_lock();
    
    for_each_process(task) {
        obtenerProcesosEnMemoria(task, &vsz, &rss);
        
        cpu_perc = calcularPorcentajeCPU(task);
        
        unsigned long mem_perc = (rss * 10000) / (total_ram ? total_ram : 1);
        
        seq_printf(m, "%d|%s|", task->pid, task->comm);
        
        if (strstr(task->comm, "docker") || strstr(task->comm, "containerd")) {
            seq_printf(m, "CONTENEDOR");
        } else {
            seq_printf(m, "%.50s", task->comm);
        }
        
        seq_printf(m, "|%lu|%lu|%lu.%02lu|%lu.%02lu\n",
                  vsz,
                  rss,
                  mem_perc / 100,
                  mem_perc % 100,
                  cpu_perc / 100,
                  cpu_perc % 100);
        
        count++;
    }
    
    rcu_read_unlock();
    
    seq_printf(m, "\nTotal procesos identificados: %d\n", count);
}

// Función que se llama cuando se inicia la lectura del archivo
static void *proc_seq_start(struct seq_file *m, loff_t *pos)
{
    if (*pos == 0) {
        return (void *)1;
    }
    return NULL;
}

// Función que se lama para obtener el siguiente elemento
static void *proc_seq_next(struct seq_file *m, void *v, loff_t *pos)
{
    (*pos)++;
    if (*pos == 1) {
        return v;
    }
    return NULL;
}

// Función que se llama al finalizar
static void proc_seq_stop(struct seq_file *m, void *v)
{

}

// Función que muestra el contenido
static int proc_seq_show(struct seq_file *m, void *v)
{
    escanearProcesos(m);
    return 0;
}

// Se definen las operaciones del archivo /proc
static const struct seq_operations proc_seq_ops = {
    .start = proc_seq_start,
    .next  = proc_seq_next,
    .stop  = proc_seq_stop,
    .show  = proc_seq_show
};

// Función que se llama cuando se abre el archivo
static int proc_open(struct inode *inode, struct file *file)
{
    return seq_open(file, &proc_seq_ops);
}

// Se definen las operaciones del archivo para proc_create
static const struct proc_ops proc_file_ops = {
    .proc_open = proc_open,
    .proc_read = seq_read,
    .proc_lseek = seq_lseek,
    .proc_release = seq_release
};

// Función que se llama al cargar el módulo usando "insmod"
static int __init iniciarModulo(void)
{
    int ret = 0;
    
    pr_info("Sonda de Kernel: Inicializando módulo...\n");
    
    proc_file = proc_create(PROC_NAME, 0444, NULL, &proc_file_ops);
    if (!proc_file) {
        pr_err("Sonda de Kernel: Error al crear /proc/%s\n", PROC_NAME);
        return -ENOMEM;
    }
    
    last_cpu_calculation = ktime_get();
    
    pr_info("Sonda de Kernel: Módulo cargado exitosamente\n");
    pr_info("Sonda de Kernel: Archivo creado en /proc/%s\n", PROC_NAME);
    
    return 0;
}

// Función que se llama al descargar el módulo (rmmod)
static void __exit salirModulo(void)
{
    proc_remove(proc_file);
    
    pr_info("Sonda de Kernel: Módulo descargado, /proc/%s eliminado\n", PROC_NAME);
}

// Se registran las funciones de inicialización y salida
module_init(iniciarModulo);
module_exit(salirModulo);