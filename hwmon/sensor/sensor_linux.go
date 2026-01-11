package sensor

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mdouchement/openfand/hwmon/environment"
	//	"github.com/NVIDIA/go-nvml/pkg/nvml"
)

//
//
//
// Based from github.com/shirou/gopsutil/v4 and adapted
// gopsutil is distributed under BSD license.
//
//
//

// from utmp.h
const hostTemperatureScale = 1000

func builTemperature() (map[string]Temperature, func() error, error) {
	//
	// HWMON
	//

	files, err := getTemperatureFiles() // e.g. /sys/class/hwmon/hwmon0/temp1_input /sys/class/hwmon/hwmon0/temp2_input /sys/class/hwmon/hwmon1/temp1_input
	if err != nil {
		return nil, nil, fmt.Errorf("could not get temperature files: %w", err)
	}

	temperatures := make(map[string]Temperature, len(files))

	// Example of a directory that contains hardware monitoring files previously found:
	// device/           temp1_crit_alarm  temp2_crit_alarm  temp3_crit_alarm  temp4_crit_alarm  temp5_crit_alarm  temp6_crit_alarm  temp7_crit_alarm
	// name              temp1_input       temp2_input       temp3_input       temp4_input       temp5_input       temp6_input       temp7_input
	// power/            temp1_label       temp2_label       temp3_label       temp4_label       temp5_label       temp6_label       temp7_label
	// subsystem/        temp1_max         temp2_max         temp3_max         temp4_max         temp5_max         temp6_max         temp7_max
	// temp1_crit        temp2_crit        temp3_crit        temp4_crit        temp5_crit        temp6_crit        temp7_crit        uevent
	var errs []error
	for i, file := range files {
		var raw []byte
		var temperature float64

		// Get the base directory location
		directory := filepath.Dir(file)

		// Get the base filename prefix like temp1
		basename := strings.Split(filepath.Base(file), "_")[0]

		// Get the base path like <dir>/temp1
		basepath := filepath.Join(directory, basename)

		// Get the label of the temperature you are reading
		raw, _ = os.ReadFile(basepath + "_label") // label file is not exist when only one temp file in the directory
		label := strings.TrimSpace(string(raw))

		// Get the name of the temperature you are reading
		if raw, err = os.ReadFile(filepath.Join(directory, "name")); err != nil {
			errs = append(errs, err)
			continue
		}

		key := strings.TrimSpace(string(raw))
		if label != "" {
			// Format the label from "Core 0" to "core_0"
			key += "_" + strings.Join(strings.Split(strings.ToLower(label), " "), "_")
		}

		device := getDeviceName(filepath.Join(directory, "device"))
		if device == "" {
			device = strings.TrimSpace(string(raw)) // key
		}

		name := device
		if label != "" {
			name += ": " + label
		}

		// Get the temperature reading
		if raw, err = os.ReadFile(file); err != nil {
			errs = append(errs, err)
			continue
		}

		if temperature, err = strconv.ParseFloat(strings.TrimSpace(string(raw)), 64); err != nil {
			errs = append(errs, err)
			continue
		}

		temp := Temperature{
			ID:          TemperatureID(i),
			Key:         key,
			Name:        name,
			Device:      device,
			Temperature: temperature / hostTemperatureScale,
			High:        optionalValueReadFromFile(basepath+"_max") / hostTemperatureScale,
			Critical:    optionalValueReadFromFile(basepath+"_crit") / hostTemperatureScale,
		}
		temp.refresh = func() (Temperature, error) {
			if raw, err = os.ReadFile(file); err != nil {
				return Temperature{}, err
			}

			if temperature, err = strconv.ParseFloat(strings.TrimSpace(string(raw)), 64); err != nil {
				return Temperature{}, err
			}

			temp.Temperature = temperature / hostTemperatureScale
			return temp, nil
		}
		temperatures[file] = temp
	}

	//
	// NVIDIA propietary drivers
	// disabled because compilation issues on non NVIDIA systems.
	// Hope NVIDIA will provide HWMON.
	//

	// ret := nvml.Init()
	// if ret != nvml.SUCCESS {
	// 	return nil, nil, fmt.Errorf("nvidia: initialize NVML: %v", nvml.ErrorString(ret))
	// }

	// count, ret := nvml.DeviceGetCount()
	// if ret != nvml.SUCCESS {
	// 	return nil, nil, fmt.Errorf("nvidia: device count: %v", nvml.ErrorString(ret))
	// }

	// for i := range count {
	// 	device, ret := nvml.DeviceGetHandleByIndex(i)
	// 	if ret != nvml.SUCCESS {
	// 		return nil, nil, fmt.Errorf("nvidia: device index %d: %v", i, nvml.ErrorString(ret))
	// 	}

	// 	key, ret := device.GetUUID()
	// 	if ret != nvml.SUCCESS {
	// 		return nil, nil, fmt.Errorf("nvidia: device ID %d: %v", i, nvml.ErrorString(ret))
	// 	}

	// 	name, ret := device.GetName()
	// 	if ret != nvml.SUCCESS {
	// 		return nil, nil, fmt.Errorf("nvidia: device name %d: %v", i, nvml.ErrorString(ret))
	// 	}

	// 	temperature, ret := device.GetTemperature(nvml.TEMPERATURE_GPU)
	// 	if ret != nvml.SUCCESS {
	// 		return nil, nil, fmt.Errorf("nvidia: device temperature %d: %v", i, nvml.ErrorString(ret))
	// 	}

	// 	threshold, ret := device.GetTemperatureThreshold(nvml.TEMPERATURE_THRESHOLD_GPU_MAX)
	// 	if ret != nvml.SUCCESS {
	// 		return nil, nil, fmt.Errorf("nvidia: device threshold %d: %v", i, nvml.ErrorString(ret))
	// 	}

	// 	temp := Temperature{
	// 		Key:         "nvidia_" + key,
	// 		Name:        name,
	// 		Device:      name,
	// 		Temperature: float64(temperature),
	// 		High:        float64(threshold),
	// 		Critical:    float64(threshold),
	// 	}
	// 	temp.refresh = func() (Temperature, error) {
	// 		temperature, ret := device.GetTemperature(nvml.TEMPERATURE_GPU)
	// 		if ret != nvml.SUCCESS {
	// 			return Temperature{}, fmt.Errorf("nvidia: device temp %d: %v", i, nvml.ErrorString(ret))
	// 		}

	// 		temp.Temperature = float64(temperature)
	// 		return temp, nil
	// 	}
	// 	temperatures[key] = temp
	// }

	close := func() error {
		// ret := nvml.Shutdown()
		// if ret != nvml.SUCCESS {
		// 	return fmt.Errorf("nvidia: shutdown NVML: %v", nvml.ErrorString(ret))
		// }

		return nil
	}
	return temperatures, close, errors.Join(errs...)
}

func getTemperatureFiles() ([]string, error) {
	var files []string
	var err error

	// Only the temp*_input file provides current temperature
	// value in millidegree Celsius as reported by the temperature to the device:
	// https://www.kernel.org/doc/Documentation/hwmon/sysfs-interface
	if files, err = filepath.Glob(environment.GetEnvPath(environment.KeyHostSys, "/sys", "/class/hwmon/hwmon*/temp*_input")); err != nil {
		return nil, err
	}

	if len(files) == 0 {
		// CentOS has an intermediate /device directory:
		// https://github.com/giampaolo/psutil/issues/971
		if files, err = filepath.Glob(environment.GetEnvPath(environment.KeyHostSys, "/sys", "/class/hwmon/hwmon*/device/temp*_input")); err != nil {
			return nil, err
		}
	}

	return files, nil
}

func optionalValueReadFromFile(filename string) float64 {
	var raw []byte
	var err error
	var value float64

	// Check if file exists
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return 0
	}

	if raw, err = os.ReadFile(filename); err != nil {
		return 0
	}

	if value, err = strconv.ParseFloat(strings.TrimSpace(string(raw)), 64); err != nil {
		return 0
	}

	return float64(value)
}

func getDeviceName(directory string) string {
	for _, file := range []string{"name", "model"} {
		if raw, err := os.ReadFile(filepath.Join(directory, file)); err == nil {
			return strings.TrimSpace(string(raw))
		}
	}

	return ""
}
