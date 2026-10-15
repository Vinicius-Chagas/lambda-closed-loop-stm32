#include "stm32f0xx_hal.h"
#include "ecu_hw.h"
#include "app.h"

int main(void)
{
    HAL_Init();          /* inicia o tick do HAL no TIM14 */
    clock_init();
    HAL_InitTick(TICK_INT_PRIORITY);  /* refaz o tick com o clock final */

    log_init();
    log_printf("\r\nECU lambda - boot\r\n");

    timing_init();
    app_start();         /* cria as tarefas e inicia o escalonador */

    for(;;) {}
}

void HardFault_Handler(void)
{
    ecu_fatal("HardFault");
}

void vApplicationStackOverflowHook(TaskHandle_t task, char *name)
{
    (void)task;
    log_printf("stack overflow: %s\r\n", name);
    ecu_fatal("stack");
}

void vApplicationMallocFailedHook(void)
{
    ecu_fatal("malloc");
}

/* Sem tarefa pronta: dorme até a próxima interrupção. Economiza energia na
 * placa e, no Renode, deixa o tempo ocioso passar sem emular instruções. */
void vApplicationIdleHook(void)
{
    __WFI();
}
