package openfand

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/mdouchement/openfand/openfan"
	"go.yaml.in/yaml/v4"
)

type Config struct {
	Debug       bool            `yaml:"debug"`
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

		fan.ID = openfan.Fan(id - 1) // fan1 => 0, fan10 => 9

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
