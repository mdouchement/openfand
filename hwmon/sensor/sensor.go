package sensor

import (
	"errors"
	"slices"
)

type Collector struct {
	temps map[string]Temperature
	close func() error
}

type TemperatureID uint16 // 65535 possible sensors should be plenty

type Temperature struct {
	ID          TemperatureID `json:"-" cbor:"-"`
	Key         string        `json:"key" cbor:"1,keyasint,omitempty,omitzero"`
	Name        string        `json:"name" cbor:"2,keyasint,omitempty,omitzero"`
	Device      string        `json:"device" cbor:"3,keyasint,omitempty,omitzero"`
	Temperature float64       `json:"temperature" cbor:"4,keyasint,omitempty,omitzero"`
	High        float64       `json:"high" cbor:"5,keyasint,omitempty,omitzero"`
	Critical    float64       `json:"critical" cbor:"6,keyasint,omitempty,omitzero"`
	refresh     func() (Temperature, error)
}

func New() (*Collector, error) {
	temps, close, err := builTemperature()
	return &Collector{
		temps: temps,
		close: close,
	}, err
}

func (c *Collector) Drop(names ...string) {
	var keys []string
	for k, v := range c.temps {
		if slices.Contains(names, v.Name) {
			keys = append(keys, k)
		}
	}

	for _, k := range keys {
		delete(c.temps, k)
	}
}

func (c *Collector) Temperatures() ([]Temperature, error) {
	temps := make([]Temperature, 0, len(c.temps))
	errs := make([]error, 0)
	for _, t := range c.temps {
		t, err := t.refresh()
		if err != nil {
			errs = append(errs, err)
			continue
		}

		temps = append(temps, t)
	}

	return temps, errors.Join(errs...)
}

func (c *Collector) Close() error {
	return c.close()
}
