# Atalhos do projeto. Requisitos: arm-none-eabi-gcc, cmake, Docker (Renode), Go.
RENODE_IMAGE ?= antmicro/renode@sha256:b30df3a8ec4eb6c793cca6ed62733c3619363de52ffd9f52de98a11f3500471b
DOCKER_RUN   = docker run --rm -v $(CURDIR):/work -w /work --user $(shell id -u):$(shell id -g) -e HOME=/tmp
FW_BUILD     = firmware/build

.PHONY: all firmware test-unit test-unit-c test-integration test-renode renode clean

all: firmware

firmware:
	cmake -S firmware -B $(FW_BUILD) -DCMAKE_TOOLCHAIN_FILE=cmake/arm-none-eabi.cmake
	cmake --build $(FW_BUILD) -j

# Testes unitários do controle em C (Unity, no host).
test-unit-c:
	cmake -S firmware/tests -B firmware/tests/build
	cmake --build firmware/tests/build -j
	ctest --test-dir firmware/tests/build --output-on-failure

test-unit: test-unit-c

# Testes Robot do firmware rodando no Renode (usa o .elf já compilado).
test-renode:
	mkdir -p build/robot
	$(DOCKER_RUN) -w /work/build/robot $(RENODE_IMAGE) renode-test /work/renode/tests/firmware.robot

test-integration: firmware
	$(MAKE) test-renode

# Renode interativo no terminal, com a ECU carregada (sem planta).
renode: firmware
	docker run --rm -it -v $(CURDIR):/work -w /work $(RENODE_IMAGE) \
		renode --disable-xwt --console -e "include @renode/lambda.resc; start"

clean:
	rm -rf $(FW_BUILD) firmware/tests/build build
