.PHONY: setup build test clean

LIBRAW_DIR := libraw/LibRaw

setup:
	git submodule add https://github.com/LibRaw/LibRaw.git $(LIBRAW_DIR) 2>/dev/null || true
	git submodule update --init --recursive
	cd $(LIBRAW_DIR) && make -f Makefile.devel -j$$(sysctl -n hw.ncpu 2>/dev/null || nproc 2>/dev/null || echo 4) library

build:
	CGO_ENABLED=1 go build -o bin/dng2jpg ./cmd/dng2jpg

test:
	CGO_ENABLED=1 go test -v ./...

clean:
	rm -rf bin/
