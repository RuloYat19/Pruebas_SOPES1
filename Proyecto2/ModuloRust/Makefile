# Versión simplificada para depuración
NAME := modulo
KERNELDIR := /lib/modules/$(shell uname -r)/build
PWD := $(shell pwd)

all:
	$(MAKE) -C $(KERNELDIR) M=$(PWD) modules \
		HOSTRUSTC=rustc-1.80 \
		RUSTC=rustc-1.80 \
		BINDGEN=bindgen-0.65 \
		RUSTFMT=rustfmt-1.80 \
		RUST_LIB_SRC=/usr/lib/rust-1.80/rustlib/src/rust/library \
		LIBCLANG_PATH=/usr/lib/llvm-17/lib

clean:
	$(MAKE) -C $(KERNELDIR) M=$(PWD) clean
	rm -f *.o *.ko *.mod*