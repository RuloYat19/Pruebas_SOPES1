obj-m := modulo.o

KDIR := /lib/modules/$(shell uname -r)/build
PWD := $(shell pwd)

ccflags-y := -Wall

all:
	$(MAKE) -C $(KDIR) M=$(PWD) modules

clean:
	$(MAKE) -C $(KDIR) M=$(PWD) clean

install:
	sudo insmod modulo.ko

remove:
	sudo rmmod modulo

test:
	@echo "=== Leyendo /proc/continfo_pr2_so1_202300722 ==="
	@cat /proc/continfo_pr2_so1_202300722

.PHONY: all clean install remove test