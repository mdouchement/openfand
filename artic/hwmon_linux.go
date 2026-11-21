package artic

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/mdouchement/openfand/hwmon/environment"
	"github.com/mdouchement/openfand/openfan"
)

type HWMonFan struct {
	sync sync.RWMutex
	ID   openfan.Fan
	PWM  uint8
	RPM  uint16
	pwm  string
	rpm  string
}

func (f *HWMonFan) Refresh() error {
	f.sync.RLock()
	defer f.sync.RUnlock()

	pwm, err := os.ReadFile(f.pwm)
	if err != nil {
		return err
	}
	pwm = bytes.TrimSpace(pwm)

	n, err := strconv.ParseUint(string(pwm), 10, 8)
	if err != nil {
		return err
	}
	f.PWM = uint8(n)

	//

	rpm, err := os.ReadFile(f.rpm)
	if err != nil {
		return err
	}
	rpm = bytes.TrimSpace(rpm)

	n, err = strconv.ParseUint(string(rpm), 10, 16)
	if err != nil {
		return err
	}
	f.RPM = uint16(n)

	return nil
}

func (f *HWMonFan) SetPWM(pwm uint8) error {
	f.sync.Lock()
	defer f.sync.Unlock()

	file, err := os.OpenFile(f.pwm, os.O_WRONLY, 0) // Since we are on readonly, the perm does not affect the file
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = fmt.Fprintf(file, "%d\n", pwm)
	if err != nil {
		return err
	}

	return file.Sync()
}

//

func builFans() (map[openfan.Fan]*HWMonFan, error) {
	rpms, pwms, err := getArticFanControllerFiles()
	if err != nil {
		return nil, err
	}

	if len(rpms) != len(pwms) {
		return nil, fmt.Errorf("inconcistent number of Artic interfaces")
	}

	fans := make(map[openfan.Fan]*HWMonFan, len(pwms))

	for _, pwm := range pwms {
		rpm, id, ok := find(pwm, rpms)
		if !ok {
			return nil, fmt.Errorf("no fanX_input match found for %s", pwm)
		}

		fans[id] = &HWMonFan{
			ID:  id,
			pwm: pwm,
			rpm: rpm,
		}
	}

	return fans, nil
}

func getArticFanControllerFiles() ([]string, []string, error) {
	var rpms, pwms []string
	var err error

	// Find the (first) artic_fan
	// https://www.kernel.org/doc/Documentation/hwmon/sysfs-interface
	if rpms, err = filepath.Glob(environment.GetEnvPath(environment.KeyHostSys, "/sys", "/class/hwmon/hwmon*/name")); err != nil {
		return nil, nil, err
	}

	if len(rpms) == 0 {
		// CentOS has an intermediate /device directory:
		// https://github.com/giampaolo/psutil/issues/971
		if rpms, err = filepath.Glob(environment.GetEnvPath(environment.KeyHostSys, "/sys", "/class/hwmon/hwmon*/device/name")); err != nil {
			return nil, nil, err
		}
	}

	var directory string
	for _, f := range rpms {
		raw, err := os.ReadFile(f)
		if err != nil {
			return nil, nil, err
		}

		if strings.TrimSpace(string(raw)) == devicename {
			directory = filepath.Dir(f)
			break
		}
	}

	if directory == "" {
		return nil, nil, ErrNotFound
	}

	//

	// RPM files
	if rpms, err = filepath.Glob(filepath.Join(directory, "fan*_input")); err != nil {
		return nil, nil, err
	}

	// Read-write PWM files
	pwms, err = filepath.Glob(filepath.Join(directory, "pwm*"))
	if err != nil {
		return nil, nil, err
	}

	return rpms, pwms, nil
}

func find(pwm string, rpms []string) (string, openfan.Fan, bool) {
	n := strings.TrimPrefix(filepath.Base(pwm), "pwm") // Extracts 10 from /sys/class/hwmon/hwmon11/pwm10
	match := fmt.Sprintf("fan%s_input", n)

	id, err := strconv.ParseUint(n, 10, 16)
	if err != nil {
		return "", 0, false
	}

	for _, rpm := range rpms {
		if strings.HasSuffix(rpm, match) {
			return rpm, openfan.Fan(id), true
		}
	}

	return "", openfan.Fan(id), false
}
