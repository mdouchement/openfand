package showsensors

import (
	"fmt"
	"slices"
	"strings"

	"github.com/mdouchement/openfand/hwmon/sensor"
	"github.com/spf13/cobra"
)

func Command() *cobra.Command {
	return &cobra.Command{
		Use:   "show-sensors",
		Short: "Show the name of available sensors",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, args []string) error {
			collector, err := sensor.New()
			if err != nil {
				return err
			}
			defer collector.Close()

			temps, err := collector.Temperatures()
			if err != nil {
				return err
			}

			slices.SortStableFunc(temps, func(a, b sensor.Temperature) int {
				aname, bname := strings.ToLower(a.Name), strings.ToLower(b.Name)

				if aname < bname {
					return -1
				}

				if aname == bname {
					return 0
				}

				return 1
			})

			for _, t := range temps {
				fmt.Printf("%3.0f°C   \"%s\"\n", t.Temperature, t.Name)
			}

			return nil
		},
	}
}
