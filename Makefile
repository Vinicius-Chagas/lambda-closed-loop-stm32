# Atalhos do projeto. Requisitos: arm-none-eabi-gcc, cmake, Docker (Renode), Go.
RENODE_IMAGE ?= antmicro/renode@sha256:b30df3a8ec4eb6c793cca6ed62733c3619363de52ffd9f52de98a11f3500471b
DOCKER_RUN   = docker run --rm -v $(CURDIR):/work -w /work --user $(shell id -u):$(shell id -g) -e HOME=/tmp
FW_BUILD     = firmware/build
TOOLCHAIN_IMAGE = lambda-toolchain
# Alvos que precisam do .elf usam o toolchain do host se houver; senão, o container.
FW_TARGET    = $(if $(shell command -v arm-none-eabi-gcc),firmware,firmware-docker)

.PHONY: all firmware firmware-docker test-unit test-unit-c test-unit-go test-sil test-integration test-renode test-renode-go test-e2e run painel painel-demo painel-captura grafico renode clean

all: firmware

firmware:
	@command -v arm-none-eabi-gcc >/dev/null || { \
		echo "arm-none-eabi-gcc não encontrado. Instale o toolchain ARM:"; \
		echo "  Arch:          sudo pacman -S arm-none-eabi-gcc arm-none-eabi-newlib"; \
		echo "  Fedora:        sudo dnf install arm-none-eabi-gcc-cs arm-none-eabi-newlib"; \
		echo "  Debian/Ubuntu: sudo apt install gcc-arm-none-eabi libnewlib-arm-none-eabi"; \
		echo "ou compile num container: make firmware-docker"; \
		exit 1; }
	cmake -S firmware -B $(FW_BUILD) -DCMAKE_TOOLCHAIN_FILE=cmake/arm-none-eabi.cmake
	cmake --build $(FW_BUILD) -j

# Mesmo firmware, compilado num container (só precisa do Docker). O build fica
# em firmware/build-docker e o .elf é copiado para $(FW_BUILD)/ecu.elf.
firmware-docker:
	docker build -q -t $(TOOLCHAIN_IMAGE) -f docker/Dockerfile.toolchain docker >/dev/null
	$(DOCKER_RUN) $(TOOLCHAIN_IMAGE) sh -c '\
		cmake -S firmware -B firmware/build-docker -DCMAKE_TOOLCHAIN_FILE=cmake/arm-none-eabi.cmake && \
		cmake --build firmware/build-docker -j'
	mkdir -p $(FW_BUILD)
	cp firmware/build-docker/ecu.elf $(FW_BUILD)/ecu.elf

# Testes unitários do controle em C (Unity, no host).
test-unit-c:
	cmake -S firmware/tests -B firmware/tests/build
	cmake --build firmware/tests/build -j
	ctest --test-dir firmware/tests/build --output-on-failure

# Testes unitários da planta, do codec CAN e do cliente do Renode (Go).
# Inclui o SIL (internal/sil), que compila firmware/control com o cgo.
test-unit-go:
	cd go && go vet ./... && go test -race ./...

test-unit: test-unit-c test-unit-go

# Malha fechada: controle C do firmware × planta Go, sem Renode (mostra as medições).
test-sil:
	cd go && go test -count=1 -v ./internal/sil

# Testes Robot do firmware rodando no Renode (usa o .elf já compilado).
test-renode:
	mkdir -p build/robot
	$(DOCKER_RUN) -w /work/build/robot $(RENODE_IMAGE) renode-test /work/renode/tests/firmware.robot

# Cliente Go contra o Renode real (monitor em localhost:1234).
test-renode-go:
	docker rm -f lambda-renode-test >/dev/null 2>&1 || true
	docker run -d --name lambda-renode-test -p 127.0.0.1:1234:1234 -v $(CURDIR):/work -w /work \
		$(RENODE_IMAGE) renode --disable-xwt --plain --port 1234 >/dev/null
	cd go && go test -tags renode -count=1 -v ./internal/renode/ -run Renode; \
		status=$$?; docker rm -f lambda-renode-test >/dev/null; exit $$status

test-integration: $(FW_TARGET)
	$(MAKE) test-sil
	$(MAKE) test-renode
	$(MAKE) test-renode-go

# Ponta a ponta no vcan0: ECU no Renode + planta + teste Go pelo CAN.
test-e2e: $(FW_TARGET)
	./scripts/e2e.sh

# Bancada para a feira (ECU + planta no vcan0). O painel roda à parte (make painel).
run: $(FW_TARGET)
	./scripts/bancada.sh -csv build/corrida.csv

# Painel (raylib). Com a bancada no ar (make run), em outro terminal:
painel:
	cd go && go run ./cmd/painel

# Painel sem Renode nem vcan0: o controle C roda dentro dele (SIL).
painel-demo:
	cd go && go run ./cmd/painel -demo

# Captura da tela do painel em modo demo (precisa de Xvfb): docs/painel.png
painel-captura:
	cd go && go build -o ../build/painel ./cmd/painel
	xvfb-run -a -s "-screen 0 1280x720x24" ./build/painel -demo -captura $(CURDIR)/docs/painel.png -apos 9

# Gráfico SVG de uma corrida: make grafico CSV=build/corrida.csv
CSV ?= build/corrida.csv
grafico:
	cd go && go run ./cmd/grafico -o $(abspath $(CSV:.csv=.svg)) $(abspath $(CSV))

# Renode interativo no terminal, com a ECU carregada (sem planta).
renode: $(FW_TARGET)
	docker run --rm -it -v $(CURDIR):/work -w /work $(RENODE_IMAGE) \
		renode --disable-xwt --console -e "include @renode/lambda.resc; start"

clean:
	rm -rf $(FW_BUILD) firmware/build-docker firmware/tests/build build
