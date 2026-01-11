package openfand

import (
	"errors"
	"fmt"
	"time"

	"github.com/mdouchement/openfand/hwmon/sensor"
	"github.com/mdouchement/openfand/openfan"
)

var (
	ErrNotFoundTemp = errors.New("temperature not found")
	ErrInvalidPWM   = errors.New("invalid PWM value")
)

type CurveShaper struct {
	labels map[openfan.Fan]string
	index  map[sensor.TemperatureID]map[openfan.Fan]func(t float64) int
}

func NewCurveShaper(cfg Config, temps []sensor.Temperature) (*CurveShaper, error) {
	s := &CurveShaper{
		labels: make(map[openfan.Fan]string),
		index:  make(map[sensor.TemperatureID]map[openfan.Fan]func(t float64) int),
	}

	findID := func(name string) (sensor.TemperatureID, error) {
		for _, t := range temps {
			if t.Name == name {
				return t.ID, nil
			}
		}

		return 0, ErrNotFoundTemp
	}

	//

	for _, fan := range cfg.FanSettings {
		s.labels[fan.ID] = fan.Label
		indexp := map[sensor.TemperatureID][]point{}

		for i, p := range fan.CurvePoints {
			for pwm, thresholds := range p {
				for tname, t := range thresholds {
					tid, err := findID(tname)
					if err != nil {
						return nil, fmt.Errorf("%s: %w", tname, err)
					}

					if i == 0 {
						// Setup the start of the curve with the first PWM defined in fan's the config.
						indexp[tid] = append(indexp[tid], point{temperature: 0, pwm: pwm})
					}

					indexp[tid] = append(indexp[tid], point{temperature: float64(t), pwm: pwm})
				}
			}
		}

		for tid, points := range indexp {
			if p := points[len(points)-1]; p.pwm < 100 {
				// Setup the end of the curve with the last PWM defined in fan's the config.
				indexp[tid] = append(indexp[tid], point{temperature: p.temperature, pwm: 100})
			}
		}

		indexs := map[sensor.TemperatureID][]segment{}
		for tid, points := range indexp {
			for i, p := range points[1:] { // i is previous index and p current point
				lowT, highT := points[i].temperature, p.temperature
				s := segment{
					temperature: lowT,
					eval:        PWMFromTempSegment(lowT, float64(points[i].pwm), highT, float64(p.pwm)),
				}

				indexs[tid] = append(indexs[tid], s)
			}
		}

		for tid, segments := range indexs {
			if s.index[tid] == nil {
				s.index[tid] = make(map[openfan.Fan]func(t float64) int)
			}

			s.index[tid][fan.ID] = func(t float64) int {
				for i := len(segments) - 1; i >= 0; i-- {
					s := segments[i]
					if t >= s.temperature {
						return int(s.eval(t))
					}
				}

				return 100 // In case of points[0] is not 0°C setting
			}
		}
	}

	return s, nil
}

func (s CurveShaper) Eval(temps []sensor.Temperature) map[openfan.Fan]Evaluation {
	pwms := map[openfan.Fan]Evaluation{}
	for _, t := range temps {
		for fid, eval := range s.index[t.ID] {
			// Find the maximum speed for the given fan that depends on several temperature sensors.
			pwms[fid] = maxPWM(pwms[fid], Evaluation{
				ID:              fid,
				EvaluedAt:       time.Now(),
				Label:           s.labels[fid],
				PWM:             eval(t.Temperature),
				TemperatureID:   t.ID,
				TemperatureName: t.Name,
				Temperature:     t.Temperature,
			})
		}
	}

	return pwms
}

func maxPWM(a, b Evaluation) Evaluation {
	if a.PWM > b.PWM {
		return a
	}
	return b
}

func PWMFromTempSegment(temp1, pwm1, temp2, pwm2 float64) func(temp float64) float64 {
	if temp1 == temp2 {
		// Simplify things in order to make clean a vertical slope
		temp2 = 2
		temp1 = 1
	}

	a := (pwm2 - pwm1) / (temp2 - temp1) // slope
	b := pwm1 - a*temp1                  // y-intercept

	return func(temp float64) float64 {
		return min(a*temp+b, 100)
	}
}
