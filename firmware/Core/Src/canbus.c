/*
 * bxCAN a 500 kbit/s (PB8 = RX, PB9 = TX). Frames recebidos vão para uma
 * fila do FreeRTOS pela interrupção da FIFO0.
 */
#include "stm32f0xx_hal.h"
#include "ecu_hw.h"
#include "queue.h"

#define RX_QUEUE_LEN 8u

static CAN_HandleTypeDef hcan;
static QueueHandle_t rx_queue;

void HAL_CAN_MspInit(CAN_HandleTypeDef *h)
{
    (void)h;
    __HAL_RCC_GPIOB_CLK_ENABLE();
    GPIO_InitTypeDef gpio = {
        .Pin = GPIO_PIN_8 | GPIO_PIN_9,
        .Mode = GPIO_MODE_AF_PP,
        .Pull = GPIO_NOPULL,
        .Speed = GPIO_SPEED_FREQ_HIGH,
        .Alternate = GPIO_AF4_CAN,
    };
    HAL_GPIO_Init(GPIOB, &gpio);
    __HAL_RCC_CAN1_CLK_ENABLE();

    /* Sai do modo sleep antes do HAL pedir o modo de inicialização.
     * O modelo bxCAN do Renode não confirma INRQ com SLEEP ainda ativo
     * (renode/renode#998); no hardware real isto é inofensivo. */
    CAN->MCR &= ~CAN_MCR_SLEEP;

    HAL_NVIC_SetPriority(CEC_CAN_IRQn, 2, 0);
    HAL_NVIC_EnableIRQ(CEC_CAN_IRQn);
}

bool canbus_init(void)
{
    rx_queue = xQueueCreate(RX_QUEUE_LEN, sizeof(can_frame_t));
    if(rx_queue == NULL) {
        return false;
    }

    /* 48 MHz / 6 = 8 MHz; 1 + 11 + 4 = 16 quanta -> 500 kbit/s, amostra em 75 % */
    hcan.Instance = CAN;
    hcan.Init.Prescaler = SYSCLK_HZ / (CAN_BITRATE * 16u);
    hcan.Init.Mode = CAN_MODE_NORMAL;
    hcan.Init.SyncJumpWidth = CAN_SJW_1TQ;
    hcan.Init.TimeSeg1 = CAN_BS1_11TQ;
    hcan.Init.TimeSeg2 = CAN_BS2_4TQ;
    hcan.Init.TimeTriggeredMode = DISABLE;
    hcan.Init.AutoBusOff = ENABLE;
    hcan.Init.AutoWakeUp = DISABLE;
    hcan.Init.AutoRetransmission = ENABLE;
    hcan.Init.ReceiveFifoLocked = DISABLE;
    hcan.Init.TransmitFifoPriority = ENABLE;
    if(HAL_CAN_Init(&hcan) != HAL_OK) {
        return false;
    }

    /* Aceita todos os IDs padrão na FIFO0. */
    CAN_FilterTypeDef filter = {
        .FilterBank = 0,
        .FilterMode = CAN_FILTERMODE_IDMASK,
        .FilterScale = CAN_FILTERSCALE_32BIT,
        .FilterIdHigh = 0,
        .FilterIdLow = 0,
        .FilterMaskIdHigh = 0,
        .FilterMaskIdLow = 0,
        .FilterFIFOAssignment = CAN_RX_FIFO0,
        .FilterActivation = ENABLE,
    };
    if(HAL_CAN_ConfigFilter(&hcan, &filter) != HAL_OK) {
        return false;
    }
    if(HAL_CAN_Start(&hcan) != HAL_OK) {
        return false;
    }
    return HAL_CAN_ActivateNotification(&hcan, CAN_IT_RX_FIFO0_MSG_PENDING) == HAL_OK;
}

bool canbus_send(const can_frame_t *frame)
{
    if(HAL_CAN_GetTxMailboxesFreeLevel(&hcan) == 0u) {
        return false;
    }
    CAN_TxHeaderTypeDef header = {
        .StdId = frame->id,
        .IDE = CAN_ID_STD,
        .RTR = CAN_RTR_DATA,
        .DLC = frame->len,
        .TransmitGlobalTime = DISABLE,
    };
    uint32_t mailbox;
    return HAL_CAN_AddTxMessage(&hcan, &header, (uint8_t *)frame->data, &mailbox) == HAL_OK;
}

bool canbus_receive(can_frame_t *frame, TickType_t timeout)
{
    return xQueueReceive(rx_queue, frame, timeout) == pdTRUE;
}

void HAL_CAN_RxFifo0MsgPendingCallback(CAN_HandleTypeDef *h)
{
    CAN_RxHeaderTypeDef header;
    can_frame_t frame;
    BaseType_t woken = pdFALSE;

    while(HAL_CAN_GetRxFifoFillLevel(h, CAN_RX_FIFO0) > 0u) {
        if(HAL_CAN_GetRxMessage(h, CAN_RX_FIFO0, &header, frame.data) != HAL_OK) {
            break;
        }
        if(header.IDE != CAN_ID_STD || header.RTR != CAN_RTR_DATA) {
            continue;
        }
        frame.id = header.StdId;
        frame.len = (uint8_t)header.DLC;
        xQueueSendFromISR(rx_queue, &frame, &woken);
    }
    portYIELD_FROM_ISR(woken);
}

void CEC_CAN_IRQHandler(void)
{
    HAL_CAN_IRQHandler(&hcan);
}
