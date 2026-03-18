# Nombre del módulo (debe coincidir con el archivo .rs)
NAME := modulo

ifneq ($(KERNELRELEASE),)
# Esta parte se ejecuta desde el sistema de compilación del kernel
obj-m += $(NAME).o
$(NAME)-y := $(NAME).o

# Flags necesarios para Rust
KBUILD_RUSTFLAGS += -Cpanic=abort -Zemit-stack-sizes

else
# Esta parte se ejecuta cuando llamamos a 'make' directamente
KERNELDIR ?= /lib/modules/$(shell uname -r)/build
PWD := $(shell pwd)

default:
	@$(MAKE) \
		HOSTRUSTC=$(RUSTC) \
		RUSTC=$(RUSTC) \
		BINDGEN=$(BINDGEN) \
		RUSTFMT=$(RUSTFMT) \
		RUST_LIB_SRC=$(RUST_LIB_SRC) \
		-C $(KERNELDIR) M=$(PWD) modules

clean:
	@$(MAKE) -C $(KERNELDIR) M=$(PWD) clean
	@rm -f *.o *.ko *.mod* .*.cmd *.d Module.symvers modules.order
	@rm -rf .tmp_versions

install:
	sudo insmod $(NAME).ko

remove:
	sudo rmmod $(NAME)
endif