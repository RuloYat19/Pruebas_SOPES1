# Nombre del módulo (debe coincidir con el archivo .rs)
NAME := modulo

ifneq ($(KERNELRELEASE),)
obj-m += $(NAME).o
$(NAME)-y := $(NAME).o
KBUILD_RUSTFLAGS += -Cpanic=abort -Zemit-stack-sizes
else
KERNELDIR ?= /lib/modules/$(shell uname -r)/build
PWD := $(shell pwd)

# Variables de Rust (ajustadas para Ubuntu Server)
RUSTC_VERSION ?= 1.80
RUSTC ?= rustc-$(RUSTC_VERSION)
BINDGEN ?= bindgen-0.65
RUSTFMT ?= rustfmt-$(RUSTC_VERSION)
RUST_LIB_SRC ?= /usr/lib/rust-1.80/rustlib/src/rust/library
LIBCLANG_PATH ?= /usr/lib/llvm-17/lib

default:
	@echo "=== Compilando módulo Rust ==="
	@echo "RUSTC: $(RUSTC)"
	@echo "BINDGEN: $(BINDGEN)"
	@echo "RUST_LIB_SRC: $(RUST_LIB_SRC)"
	@echo "============================="
	@$(MAKE) \
		HOSTRUSTC=$(RUSTC) \
		RUSTC=$(RUSTC) \
		BINDGEN=$(BINDGEN) \
		RUSTFMT=$(RUSTFMT) \
		RUST_LIB_SRC=$(RUST_LIB_SRC) \
		LIBCLANG_PATH=$(LIBCLANG_PATH) \
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