BIN_DIR := ./bin
BIN_PREFIX := gexcels

all: windows_x64 windows_arm64 darwin_x64 darwin_arm64

windows_x64:
	make build build_os=windows build_arch=amd64

windows_arm64:
	make build build_os=windows build_arch=arm64

darwin_x64:
	make build build_os=darwin build_arch=amd64

darwin_arm64:
	make build build_os=darwin build_arch=arm64

build: build_os := darwin
build: build_arch := arm64
build:
	GOOS=$(build_os) GOARCH=$(build_arch) go build -o $(BIN_DIR)/$(BIN_PREFIX)_$(build_os)_$(build_arch) cmd.go