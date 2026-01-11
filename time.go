package openfand

import (
	"encoding/json"
	"time"

	"go.yaml.in/yaml/v4"
)

type Duration struct {
	time.Duration
}

func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

func (d *Duration) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return nil
	}

	var str string
	err := json.Unmarshal(data, &str)
	if err != nil {
		return err
	}

	d.Duration, err = time.ParseDuration(str)
	return err
}

func (d Duration) MarshalYAML() (any, error) {
	return d.String(), nil
}

func (d *Duration) UnmarshalYAML(value *yaml.Node) error {
	var str string
	err := value.Decode(&str)
	if err != nil {
		return err
	}

	if str == "" {
		return nil
	}

	d.Duration, err = time.ParseDuration(str)
	return err
}
