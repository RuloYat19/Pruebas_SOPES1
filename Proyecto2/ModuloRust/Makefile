# Makefile final para módulo Rust
NAME := modulo
KERNELDIR := /lib/modules/$(shell uname -r)/build
PWD := $(shell pwd)

all:
	@echo "=== Compilando módulo Rust: $(NAME) ==="
	@$(MAKE) -C $(KERNELDIR) M=$(PWD) modules \
		HOSTRUSTC=rustc-1.80 \
		RUSTC=rustc-1.80 \
		BINDGEN=bindgen-0.65 \
		RUSTFMT=rustfmt-1.80 \
		RUST_LIB_SRC=/usr/lib/rust-1.80/rustlib/src/rust/library \
		LIBCLANG_PATH=/usr/lib/llvm-17/lib \
		KBUILD_EXTRA_SYMBOLS=

clean:
	@$(MAKE) -C $(KERNELDIR) M=$(PWD) clean
	@rm -f *.o *.ko *.mod* .*.cmd *.d Module.symvers modules.order
	@rm -rf .tmp_versions

# Le decimos al kernel que esto es un módulo Rust
obj-m += $(NAME).o
$(NAME)-y := $(NAME).rs