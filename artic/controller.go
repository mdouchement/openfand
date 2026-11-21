package artic

import (
	"errors"
	"fmt"

	"github.com/mdouchement/logger"
	"github.com/mdouchement/openfand/openfan"
)

var (
	ErrNotFound   = errors.New("device not found/plugged")
	ErrInvalidPWM = errors.New("invalid PWM value")
)

type Controller struct {
	log  logger.Logger
	fans map[openfan.Fan]*HWMonFan
}

func Open() (*Controller, error) {
	fans, err := builFans()
	if err != nil {
		return nil, err
	}

	c := &Controller{
		fans: fans,
	}

	return c, nil
}

func (c *Controller) SetLogger(l logger.Logger) {
	c.log = l
}

func (c *Controller) RPMs() (map[openfan.Fan]uint16, error) {
	var err error
	m := make(map[openfan.Fan]uint16, len(c.fans))
	for id, fan := range c.fans {
		err = fan.Refresh() // TODO: concurrent refresh?
		if err != nil {
			return nil, fmt.Errorf("fan_all_get_rpm: %d: v: %w", id, err)
		}

		m[id] = fan.RPM
	}

	return m, nil
}

func (c *Controller) SetPWM(f openfan.Fan, pwm int) (int, error) {
	if pwm < 0 || pwm > 100 {
		return 0, ErrInvalidPWM
	}

	n := pwm*255/100 + 1
	n = min(n, 255)

	fan, ok := c.fans[f]
	if !ok {
		return 0, errors.New("fan_set_pwm: fan not found")
	}

	err := fan.SetPWM(uint8(n))
	if err != nil {
		return 0, fmt.Errorf("fan_set_pwm: %w", err)
	}

	return n, nil
}
