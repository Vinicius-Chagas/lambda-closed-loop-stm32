# Testes unitários do código de controle (C puro), compilados no host:
#   cmake -S firmware/tests -B firmware/tests/build && cmake --build firmware/tests/build
#   ctest --test-dir firmware/tests/build --output-on-failure
cmake_minimum_required(VERSION 3.20)
project(ecu_control_tests C)

set(CMAKE_C_STANDARD 11)
set(CMAKE_C_STANDARD_REQUIRED ON)
enable_testing()

set(CONTROL ${CMAKE_CURRENT_SOURCE_DIR}/../control)
set(UNITY ${CMAKE_CURRENT_SOURCE_DIR}/../third_party/Unity)

add_library(unity STATIC ${UNITY}/unity.c)
target_include_directories(unity PUBLIC ${UNITY})
target_compile_definitions(unity PUBLIC UNITY_INCLUDE_DOUBLE)

file(GLOB CONTROL_SOURCES ${CONTROL}/*.c)
add_library(control STATIC ${CONTROL_SOURCES})
target_include_directories(control PUBLIC ${CONTROL})
target_compile_options(control PRIVATE -Wall -Wextra -Werror -Wdouble-promotion)
target_link_libraries(control PUBLIC m)

file(GLOB TEST_SOURCES ${CMAKE_CURRENT_SOURCE_DIR}/test_*.c)
foreach(src ${TEST_SOURCES})
    get_filename_component(name ${src} NAME_WE)
    add_executable(${name} ${src})
    target_compile_options(${name} PRIVATE -Wall -Wextra -Werror)
    target_link_libraries(${name} PRIVATE control unity)
    add_test(NAME ${name} COMMAND ${name})
endforeach()
