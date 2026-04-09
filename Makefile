APP_NAME := peyote-designer
APP_DISPLAY_NAME := Peyote Designer
CMD := ./cmd/peyote-designer
DIST := dist
MACOS_APP := $(DIST)/$(APP_DISPLAY_NAME).app
MACOS_DMG := $(DIST)/$(APP_NAME)-macos-arm64.dmg
WINDOWS_BIN := $(DIST)/$(APP_NAME)-windows-amd64.exe
WINDOWS_ZIP := $(DIST)/$(APP_NAME)-windows-amd64.zip

.PHONY: run build build-macos build-windows test lint clean package

run:
	go run $(CMD)

build: build-macos

build-macos:
	sh ./scripts/build_macos_app.sh

build-windows:
	mkdir -p $(DIST)
	@if command -v x86_64-w64-mingw32-gcc >/dev/null 2>&1; then \
		CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc GOOS=windows GOARCH=amd64 go build -o $(WINDOWS_BIN) $(CMD); \
	else \
		echo "Windows build requires x86_64-w64-mingw32-gcc or a native Windows runner."; \
		exit 1; \
	fi

test:
	go test ./...

lint:
	golangci-lint run ./...

clean:
	rm -rf $(DIST)

package: clean build-macos build-windows
	sh ./scripts/package_macos_dmg.sh
	cd $(DIST) && zip -j $(notdir $(WINDOWS_ZIP)) $(notdir $(WINDOWS_BIN))
