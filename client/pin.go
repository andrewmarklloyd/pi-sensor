package main

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/stianeikeland/go-rpio"
)

const (
	OPEN    = "OPEN"
	CLOSED  = "CLOSED"
	UNKNOWN = "UNKNOWN"
)

type pinClient struct {
	pin      rpio.Pin
	mockMode bool
}

func newPinClient(pinNumber int, mockMode bool) pinClient {
	pin := rpio.Pin(pinNumber)
	err := rpio.Open()
	if err != nil {
		log.Println(fmt.Sprintf("Unable to open gpio: %s, continuing but running in test mode.", err.Error()))
	}
	return pinClient{pin, mockMode}
}

func (c *pinClient) CurrentStatus() string {
	var pinState int
	if c.mockMode {
		rand.Seed(time.Now().Unix())
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

func (c *pinClient) Cleanup() {
	rpio.Close()
}
