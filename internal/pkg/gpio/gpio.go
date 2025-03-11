package gpio

import (
	"log"
	"math/rand"

	"github.com/stianeikeland/go-rpio"
)

const (
	OPEN    = "OPEN"
	CLOSED  = "CLOSED"
	UNKNOWN = "UNKNOWN"
)

type PinClient struct {
	pin      rpio.Pin
	mockMode bool
}

func NewPinClient(pinNumber int, mockMode bool) PinClient {
	pin := rpio.Pin(pinNumber)
	err := rpio.Open()
	if err != nil {
		log.Printf("Unable to open gpio: %s, continuing but running in test mode.\n", err.Error())
	} else {
		pin.Input()
		pin.PullUp()
	}

	return PinClient{
		pin,
		mockMode,
	}
}

func (c *PinClient) CurrentStatus() string {
	var pinState int
	if c.mockMode {
		randStatus := []string{
			CLOSED,
			OPEN,
		}
		n := rand.Int() % len(randStatus)
		return randStatus[n]
	}
	pinState = int(c.pin.Read())

	if pinState == 0 {
		return CLOSED
	} else if pinState == 1 {
		return OPEN
	}
	return UNKNOWN
}

func (c *PinClient) Cleanup() {
	rpio.Close()
}
