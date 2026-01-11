package main

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"os"
	"os/signal"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"time"

	"github.com/mdouchement/logger"
	"github.com/mdouchement/openfand"
	showcurves "github.com/mdouchement/openfand/cmd/openfand/show_curves"
	showsensors "github.com/mdouchement/openfand/cmd/openfand/show_sensors"
	"github.com/mdouchement/openfand/hwmon/sensor"
	"github.com/mdouchement/openfand/openfan"
	"github.com/spf13/cobra"
)

var (
	version  = "dev"
	revision = "none"
	date     = "unknown"

	cpath string
	dummy bool
)

func main() {
	cmd := &cobra.Command{
		Use:     "openfand",
		Short:   "A controller for OpenFanController hardware",
		Version: fmt.Sprintf("%s - build %.7s @ %s - %s", version, revision, date, runtime.Version()),
		Args:    cobra.NoArgs,
		RunE:    daemon,
	}
	cmd.Flags().StringVarP(&cpath, "config", "c", "/etc/openfand/openfand.yml", "Configfile path")
	cmd.Flags().BoolVarP(&dummy, "dummy", "", false, "Start openfand with a dummy openfan controller")
	cmd.AddCommand(showcurves.Command())
	cmd.AddCommand(showsensors.Command())
	cmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Version for openfand",
		Args:  cobra.NoArgs,
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Println(cmd.Version)
		},
	})

	if err := cmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func daemon(_ *cobra.Command, args []string) error {
	cfg, err := openfand.Load(cpath)
	if err != nil {
		return err
	}

	level := slog.LevelInfo
	if cfg.Debug {
		level = slog.LevelDebug
	}

	h := logger.NewSlogTextHandler(os.Stdout, &logger.SlogTextOption{
		Level:            level,
		ForceColors:      true,
		ForceFormatting:  true,
		PrefixRE:         regexp.MustCompile(`^(\[.*?\])\s`),
		DisableTimestamp: true, // Provided by journalctl
		// FullTimestamp:    true,
		// TimestampFormat:  "2006-01-02 15:04:05",
	})
	log := logger.WrapSlogHandler(h)
	ctx := logger.WithLogger(context.Background(), log)

	log.Infof("openfand version %s", version)

	var fan openfand.OpenFan = openfand.NewDummyOpenfanController()
	if !dummy {
		ctrl, err := openfan.OpenAuto()
		if err != nil {
			return fmt.Errorf("openfan: %w", err)
		}
		if cfg.Debug {
			ctrl.SetLogger(log)
		}

		{
			log.Infof("Fan Controller port `%s`", ctrl.Port())

			hw, err := ctrl.HardwareInfo()
			if err != nil {
				panic(err)
			}
			log.Infof("Hardware - REV: %s - MCU: %s - USB: %s - FAN_CHANNELS_TOTAL: %s - FAN_CHANNELS_ARCH: %s - FAN_CHANNELS_DRIVER: %s",
				hw.Revision, hw.MCU, hw.USB, hw.FanChannelsTotal, hw.FanChannelsArch, hw.FanChannelsDriver)

			fw, err := ctrl.FirmwareInfo()
			if err != nil {
				panic(err)
			}
			log.Infof("Firmware - REV: %s - PROTOCOL_VERSION: %s", fw.Revision, fw.ProtocolVersion)
		}

		defer ctrl.Close()
		fan = ctrl
	}

	collector, err := sensor.New()
	if err != nil {
		return err
	}
	defer collector.Close()

	temps, err := trimCollector(cfg, collector)
	if err != nil {
		return err
	}

	shaper, err := openfand.NewCurveShaper(cfg, temps)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(ctx)

	controler, err := openfand.New(cfg, fan, collector, shaper, 500*time.Millisecond)
	if err != nil {
		cancel()
		return err
	}
	controler.Launch(ctx)

	ctx, stop := signal.NotifyContext(ctx, os.Interrupt)
	defer stop()

	<-ctx.Done()
	cancel()

	log.Info("Gracefully shutdown")
	return nil
}

func trimCollector(cfg openfand.Config, collector *sensor.Collector) ([]sensor.Temperature, error) {
	temps, err := collector.Temperatures()
	if err != nil {
		return nil, fmt.Errorf("collect temperatures: %w", err)
	}
	exists := map[string]bool{}
	unwanted := map[string]bool{}
	for _, temp := range temps {
		exists[temp.Name] = true
		unwanted[temp.Name] = true
	}

	for _, fan := range cfg.FanSettings {
		for _, point := range fan.CurvePoints {
			for _, thresholds := range point {
				for name := range thresholds {
					if !exists[name] {
						return nil, fmt.Errorf("not found: %s", strconv.Quote(name))
					}

					delete(unwanted, name)
				}
			}
		}
	}

	collector.Drop(slices.Collect(maps.Keys(unwanted))...)
	return temps, nil
}
