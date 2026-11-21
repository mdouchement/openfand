package openfand

import (
	"fmt"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/mdouchement/openfand/openfan"
	"go.yaml.in/yaml/v4"
)

const (
	DriverArtic   = "artic"
	DriverOpenFan = "openfan"
)

type Config struct {
	Debug       bool            `yaml:"debug"`
	Driver      string          `yaml:"driver"`
	Socket      string          `yaml:"socket"`
	FanSettings map[string]*Fan `yaml:"fan_settings"`
}

type Fan struct {
	ID              openfan.Fan                 `yaml:"-"`
	Label           string                      `yaml:"label"`
	FanSetUp        Duration                    `yaml:"fan_step_up"`
	FanSetDown      Duration                    `yaml:"fan_step_down"`
	CurvePointsYAML []map[string]map[string]int `yaml:"curve_points"`
	CurvePoints     []map[int]map[string]int    `yaml:"-"`
}

func Load(path string) (Config, error) {
	var c Config

	f, err := os.Open(path)
	if err != nil {
		return c, err
	}
	defer f.Close()

	codec := yaml.NewDecoder(f)
	err = codec.Decode(&c)
	if err != nil {
		return c, err
	}

	if c.Driver == "" {
		c.Driver = DriverOpenFan
	}
	if !slices.Contains([]string{DriverArtic, DriverOpenFan}, c.Driver) {
		return c, fmt.Errorf("unsupported driver: %s", c.Driver)
	}

	//

	reName := regexp.MustCompile(`^fan(\d+)$`)
	rePWM := regexp.MustCompile(`\d+%`)
	for fname, fan := range c.FanSettings {
		match := reName.FindStringSubmatch(fname)
		if len(match) != 2 {
			return c, fmt.Errorf("%s: invalid name", fname)
		}
		id, err := strconv.ParseUint(match[1], 10, 8)
		if err != nil {
			return c, fmt.Errorf("%s: invalid number", fname) // Should not happen because of the regex check
		}
		if id < 1 || id > 10 {
			return c, fmt.Errorf("%s: invalid number range", fname)
		}

		fan.ID = c.toRealID(id)

		if len(fan.CurvePointsYAML) == 0 {
			return c, fmt.Errorf("%s: no curve_points provided", fname)
		}

		fan.CurvePoints = make([]map[int]map[string]int, len(fan.CurvePointsYAML))

		var prevPWM int
		for i, point := range fan.CurvePointsYAML {
			fan.CurvePoints[i] = make(map[int]map[string]int)

			for pwm, thresholds := range point {
				if !rePWM.MatchString(pwm) {
					return c, fmt.Errorf("%s: invalid pwm format %s", fname, pwm)
				}

				PWM, err := strconv.Atoi(strings.TrimRight(pwm, "%"))
				if err != nil {
					return c, fmt.Errorf("%s: %s: %w", fname, pwm, err)
				}
				if PWM < 0 || PWM > 100 {
					return c, fmt.Errorf("%s: %s: pwm must in range [0,100]", fname, pwm)
				}
				if PWM < prevPWM {
					return c, fmt.Errorf("%s: %s: pwm greater than previous one", fname, pwm)
				}
				prevPWM = PWM

				if len(thresholds) == 0 {
					return c, fmt.Errorf("%s: %s: no temperature thresholds specified", fname, pwm)
				}

				fan.CurvePoints[i][PWM] = thresholds
			}
		}
	}

	return c, nil
}

func (c Config) toRealID(id uint64) openfan.Fan {
	switch c.Driver {
	case DriverArtic:
	case DriverOpenFan:
		id-- // fan1 => 0, fan10 => 9
	}

	return openfan.Fan(id)
}

func (c Config) ToConfigID(id openfan.Fan) openfan.Fan {
	switch c.Driver {
	case DriverArtic:
	case DriverOpenFan:
		id++ // 0 => fan1, 9 => fan10
	}

	return openfan.Fan(id)
}
