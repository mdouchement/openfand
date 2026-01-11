package openfand

import (
	"sync"

	"github.com/mdouchement/logger"
	"github.com/mdouchement/openfand/openfan"
)

// A DummyOpenfanController should only be used for dev & tests.
type DummyOpenfanController struct {
	sync sync.Mutex
	pwms map[openfan.Fan]int
	log  logger.Logger
}

func NewDummyOpenfanController() *DummyOpenfanController {
	n := 10
	c := &DummyOpenfanController{
		pwms: make(map[openfan.Fan]int, n),
	}
	for i := range n {
		c.pwms[openfan.Fan(i)] = 0
	}

	return c
}

func (c *DummyOpenfanController) SetLogger(l logger.Logger) {
	c.log = l
}

func (c *DummyOpenfanController) Close() error {
	return nil
}

func (c *DummyOpenfanController) Port() string {
	return "x-testing"
}

func (c *DummyOpenfanController) HardwareInfo() (*openfan.HardwareInfo, error) {
	return &openfan.HardwareInfo{
		Revision:          "n/a",
		MCU:               "n/a",
		USB:               "n/a",
		FanChannelsTotal:  "n/a",
		FanChannelsArch:   "n/a",
		FanChannelsDriver: "n/a",
	}, nil
}

func (c *DummyOpenfanController) FirmwareInfo() (*openfan.FirmwareInfo, error) {
	return &openfan.FirmwareInfo{
		Revision:        "n/a",
		ProtocolVersion: "n/a",
	}, nil
}

func (c *DummyOpenfanController) RPMs() (map[openfan.Fan]uint16, error) {
	c.sync.Lock()
	defer c.sync.Unlock()

	rpms := make(map[openfan.Fan]uint16, len(c.pwms))
	for k, pwm := range c.pwms {
		rpms[k] = uint16(1500 * float32(pwm) / 100)
	}

	return rpms, nil
}

func (c *DummyOpenfanController) SetPWM(f openfan.Fan, pwm int) (int, error) {
	c.sync.Lock()
	defer c.sync.Unlock()

	c.pwms[f] = pwm
	return pwm, nil
}
