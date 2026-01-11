package showcurves

import (
	"bytes"
	"fmt"
	"image"
	_ "image/png"
	"maps"
	"os"
	"slices"
	"strconv"

	"github.com/go-analyze/charts"
	"github.com/mattn/go-sixel"
	"github.com/mdouchement/openfand"
	"github.com/mdouchement/openfand/hwmon/sensor"
	"github.com/mdouchement/openfand/openfan"
	"github.com/spf13/cobra"
)

func Command() *cobra.Command {
	var cpath string
	var resolution int

	cmd := &cobra.Command{
		Use:   "show-curves",
		Short: "Show the curves for each fan",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, args []string) error {
			cfg, err := openfand.Load(cpath)
			if err != nil {
				return err
			}

			collector, err := sensor.New()
			if err != nil {
				return err
			}
			defer collector.Close()

			temps, err := collector.Temperatures()
			if err != nil {
				return err
			}

			shaper, err := openfand.NewCurveShaper(cfg, temps)
			if err != nil {
				return err
			}

			var maxT int
			labels := map[openfan.Fan]string{}
			probes := map[string]sensor.TemperatureID{}
			for _, fan := range cfg.FanSettings {
				labels[fan.ID] = fan.Label

				for _, p := range fan.CurvePoints {
					for _, thresholds := range p {
						for tname, v := range thresholds {
							maxT = max(maxT, v)
							if _, ok := probes[tname]; !ok {
								for _, t := range temps {
									if t.Name == tname {
										probes[tname] = t.ID
										break
									}
								}
							}
						}
					}
				}
			}

			maxT = max(maxT, 100) // Set defaults to 100°C which leads to better x-axis values.

			//
			// Compute points
			//

			m := make(map[openfan.Fan]map[sensor.TemperatureID]charts.LineSeries)
			decimals := 100 // HWMON can return 42.321°C

			for probe, tid := range probes {
				for t := range maxT + 1 {
					for decimal := range decimals {
						temps = []sensor.Temperature{
							{
								ID:          tid,
								Name:        probe,
								Temperature: float64(t) + float64(decimal)/float64(decimals),
							},
						}

						for _, eval := range shaper.Eval(temps) {
							if _, ok := m[eval.ID]; !ok {
								m[eval.ID] = make(map[sensor.TemperatureID]charts.LineSeries)
							}
							if _, ok := m[eval.ID][eval.TemperatureID]; !ok {
								m[eval.ID][eval.TemperatureID] = charts.LineSeries{
									Name: eval.TemperatureName,
								}
							}

							ls := m[eval.ID][eval.TemperatureID]
							ls.Values = append(ls.Values, float64(eval.PWM))
							m[eval.ID][eval.TemperatureID] = ls
						}
					}
				}
			}

			//
			// Render charts
			//

			for _, fid := range slices.Sorted(maps.Keys(m)) {
				fm := m[fid]

				var set charts.LineSeriesList
				for _, ls := range fm {
					set = append(set, ls)
				}

				opt := charts.NewLineChartOptionWithSeries(set)
				opt.Theme = charts.GetTheme(charts.ThemeVividDark)
				opt.Padding = charts.NewBox(20, 20, 20, 20)
				opt.Title.Text = fmt.Sprintf("fan%d: %s", fid+1, labels[fid])
				opt.Title.FontStyle.FontSize = 16
				opt.Title.Offset = charts.OffsetLeft
				opt.Legend = charts.LegendOption{
					Show:     openfand.ToPtr(true),
					Offset:   charts.OffsetCenter,
					Vertical: openfand.ToPtr(true),
					Padding:  charts.NewBox(0, 0, 0, 20),
				}
				opt.Symbol = charts.SymbolNone
				opt.LineStrokeWidth = 2
				opt.StrokeSmoothingTension = 1
				opt.XAxis.Show = openfand.ToPtr(true)
				opt.XAxis.Title = "°C"
				opt.XAxis.Labels = []string{} // Reset
				for t := range maxT + 1 {
					for range decimals {
						// Generate same integer for all decimals points of that integer.
						// It offers a better `opt.XAxis.LabelCount = maxT / 10' display.
						opt.XAxis.Labels = append(opt.XAxis.Labels, strconv.Itoa(t))
					}
				}
				opt.XAxis.LabelCount = maxT / 10
				opt.YAxis = []charts.YAxisOption{
					{
						Show:                   openfand.ToPtr(true),
						Title:                  "%",
						Min:                    openfand.ToPtr(float64(0)),
						Max:                    openfand.ToPtr(float64(100)),
						RangeValuePaddingScale: openfand.ToPtr(float64(0)),
						Unit:                   10,
					},
				}
				p := charts.NewPainter(charts.PainterOptions{
					OutputFormat: charts.ChartOutputPNG,
					Width:        resolution,
					Height:       int(float64(resolution) / (16.0 / 9.0)),
				})

				err := p.LineChart(opt)
				if err != nil {
					return fmt.Errorf("fan%d: %w", fid+1, err)
				}

				mPNG, err := p.Bytes()
				if err != nil {
					return fmt.Errorf("fan%d: %w", fid+1, err)
				}

				m, _, err := image.Decode(bytes.NewReader(mPNG))
				if err != nil {
					return fmt.Errorf("fan%d: %w", fid+1, err)
				}

				codec := sixel.NewEncoder(os.Stdout)
				err = codec.Encode(m)
				if err != nil {
					return fmt.Errorf("fan%d: %w", fid+1, err)
				}
			}

			return nil
		},
	}
	cmd.Flags().StringVarP(&cpath, "config", "c", "/etc/openfand/openfand.yml", "Configfile path")
	cmd.Flags().IntVarP(&resolution, "resolution", "r", 1000, "The width size in pixel of each graph")

	return cmd
}

func genTemps(probs map[string]sensor.TemperatureID, t float64) []sensor.Temperature {
	var temps []sensor.Temperature
	for tname, id := range probs {
		temps = append(temps, sensor.Temperature{
			ID:          id,
			Name:        tname,
			Temperature: t,
		})
	}

	return temps
}
