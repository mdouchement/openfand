package openfand

import (
	"time"

	"github.com/mdouchement/openfand/hwmon/sensor"
	"github.com/mdouchement/openfand/openfan"
)

type OpenFan interface {
	RPMs() (map[openfan.Fan]uint16, error)
	SetPWM(f openfan.Fan, pwm int) (int, error)
}

type Sensor interface {
	Temperatures() ([]sensor.Temperature, error)
}

type Shaper interface {
	Eval(temps []sensor.Temperature) map[openfan.Fan]Evaluation
}

type Evaluation struct {
	ID              openfan.Fan          `json:"id"`
	EvaluedAt       time.Time            `json:"-"`
	Label           string               `json:"label"`
	PWM             int                  `json:"pwm"`
	RPM             uint16               `json:"rpm"`
	TemperatureID   sensor.TemperatureID `json:"-"`
	TemperatureName string               `json:"temperature_name"`
	Temperature     float64              `json:"temperature"`
}

func ToPtr[T any](v T) *T {
	return &v
}

type point struct {
	temperature float64
	pwm         int
}

type segment struct {
	temperature float64
	eval        func(float64) float64
}

const (
	eventUpdateEval      = "update-eval"
	eventUpdateRPMs      = "update-rpms"
	eventWatch           = "watch"
	eventRefreshWatchers = "refresh-watchers"
	eventUnwatch         = "unwatch"
)

type event struct {
	name      string
	eval      Evaluation
	rpms      map[openfan.Fan]uint16
	monitorID int64
	monitor   chan<- []byte
}

type refresh struct {
	current  uint8
	until    uint8
	interval time.Duration
}

func genID() int64 {
	time.Sleep(time.Nanosecond)
	return time.Now().UnixNano()
}
