# Nombre del módulo
NAME = modulo

ifneq ($(KERNELRELEASE),)

# Se compila el archivo .rs como un módulo
obj-m += $(NAME).o

# Se especifica que el archivo objeto se compila a partir de la fuente Rust
$(NAME)-y := $(NAME).o

KBUILD_RUSTFLAGS += -Cpanic=abort -Zemit-stack-sizes

else

KERNELDIR ?= /lib/modules/$(shell uname -r)/build
PWD := $(shell pwd)

default:
	$(MAKE) -C $(KERNELDIR) M=$(PWD) modules

clean:
	$(MAKE) -C $(KERNELDIR) M=$(PWD) clean

endif