package lock

import (
	"fmt"
	"regexp"
	"strconv"
)

// Socketpath between lock daemon and client
var Socketpath = "/var/run/bench.socket"

// CpufreqFlag ...
type CpufreqFlag struct {
	Percent int
}

func (f *CpufreqFlag) String() string {
	if f.Percent < 0 {
		return "none"
	}
	return fmt.Sprintf("%d", f.Percent)
}

// Set set the cpu frequency percentage
func (f *CpufreqFlag) Set(v string) error {
	if v == "none" {
		f.Percent = -1
	} else {
		m := regexp.MustCompile(`^([0-9]+)$`).FindStringSubmatch(v)
		if m == nil {
			return fmt.Errorf("cpufreq must be \"none\" or \"N\"")
		}
		f.Percent, _ = strconv.Atoi(m[1])
	}
	return nil
}
