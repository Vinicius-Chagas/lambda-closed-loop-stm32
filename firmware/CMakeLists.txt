# Firmware da ECU (STM32F072). Compilar com:
#   cmake -B build -DCMAKE_TOOLCHAIN_FILE=cmake/arm-none-eabi.cmake && cmake --build build
cmake_minimum_required(VERSION 3.20)
project(ecu_lambda C ASM)

set(CMAKE_C_STANDARD 11)
set(CMAKE_C_STANDARD_REQUIRED ON)
if(NOT CMAKE_BUILD_TYPE)
    set(CMAKE_BUILD_TYPE RelWithDebInfo)
endif()
set(CMAKE_C_FLAGS_RELWITHDEBINFO "-O2 -g3")

set(TP ${CMAKE_CURRENT_SOURCE_DIR}/third_party)

file(GLOB HAL_SOURCES ${TP}/stm32f0xx_hal_driver/Src/*.c)
set(RTOS_SOURCES
    ${TP}/FreeRTOS-Kernel/tasks.c
    ${TP}/FreeRTOS-Kernel/queue.c
    ${TP}/FreeRTOS-Kernel/list.c
    ${TP}/FreeRTOS-Kernel/timers.c
    ${TP}/FreeRTOS-Kernel/portable/GCC/ARM_CM0/port.c
    ${TP}/FreeRTOS-Kernel/portable/GCC/ARM_CM0/portasm.c
    ${TP}/FreeRTOS-Kernel/portable/MemMang/heap_4.c
)
file(GLOB CONTROL_SOURCES ${CMAKE_CURRENT_SOURCE_DIR}/control/*.c)
file(GLOB APP_SOURCES ${CMAKE_CURRENT_SOURCE_DIR}/Core/Src/*.c)

add_library(vendor STATIC
    ${HAL_SOURCES}
    ${RTOS_SOURCES}
    ${TP}/cmsis_device_f0/Source/system_stm32f0xx.c
)
target_include_directories(vendor PUBLIC
    ${CMAKE_CURRENT_SOURCE_DIR}/Core/Inc
    ${TP}/cmsis_core/Include
    ${TP}/cmsis_device_f0/Include
    ${TP}/stm32f0xx_hal_driver/Inc
    ${TP}/FreeRTOS-Kernel/include
    ${TP}/FreeRTOS-Kernel/portable/GCC/ARM_CM0
)
target_compile_definitions(vendor PUBLIC STM32F072xB USE_HAL_DRIVER)

add_executable(ecu.elf
    ${APP_SOURCES}
    ${CONTROL_SOURCES}
    ${TP}/cmsis_device_f0/Source/startup_stm32f072xb.s
)
target_include_directories(ecu.elf PRIVATE ${CMAKE_CURRENT_SOURCE_DIR}/control)
target_compile_options(ecu.elf PRIVATE -Wall -Wextra -Werror)
target_link_libraries(ecu.elf PRIVATE vendor)
target_link_options(ecu.elf PRIVATE
    -T${CMAKE_CURRENT_SOURCE_DIR}/STM32F072RBTx_FLASH.ld
    -Wl,-Map=${CMAKE_CURRENT_BINARY_DIR}/ecu.map
    -Wl,--print-memory-usage
)

add_custom_command(TARGET ecu.elf POST_BUILD
    COMMAND ${CMAKE_SIZE} ecu.elf
    COMMAND ${CMAKE_OBJCOPY} -O binary ecu.elf ecu.bin
)
