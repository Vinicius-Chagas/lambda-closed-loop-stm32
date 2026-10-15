package canbus

import (
	"context"
	"net"

	"go.einride.tech/can"
	"go.einride.tech/can/pkg/socketcan"
)

// Bus é uma conexão SocketCAN (Linux), ex.: "vcan0".
type Bus struct {
	conn net.Conn
	rx   *socketcan.Receiver
	tx   *socketcan.Transmitter
}

// Open abre a interface CAN.
func Open(ctx context.Context, iface string) (*Bus, error) {
	conn, err := socketcan.DialContext(ctx, "can", iface)
	if err != nil {
		return nil, err
	}
	return &Bus{conn: conn, rx: socketcan.NewReceiver(conn), tx: socketcan.NewTransmitter(conn)}, nil
}

// Send transmite um frame.
func (b *Bus) Send(ctx context.Context, f can.Frame) error { return b.tx.TransmitFrame(ctx, f) }

// Receive bloqueia até o próximo frame (ok = false quando a conexão fecha).
func (b *Bus) Receive() (can.Frame, bool) {
	if !b.rx.Receive() {
		return can.Frame{}, false
	}
	return b.rx.Frame(), true
}

// Err é o erro que encerrou a recepção, se houver.
func (b *Bus) Err() error { return b.rx.Err() }

// Close fecha a conexão.
func (b *Bus) Close() error { return b.conn.Close() }
