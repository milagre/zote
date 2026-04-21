// Package profile describes a workload's resource profile (CPU, memory,
// replica count). Input is the same string shape used by Kubernetes manifests
// ("100m" / "512M"); validated numeric values are exposed for consumers.
package profile

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

var (
	cpuPattern = regexp.MustCompile(`^[0-9]+m$`)
	memPattern = regexp.MustCompile(`^[0-9]+M$`)
)

// Raw is the YAML-shaped input form of a profile.
type Raw struct {
	CPU RawRange  `yaml:"cpu"`
	Mem RawRange  `yaml:"mem"`
	Num *IntRange `yaml:"num,omitempty"`
}

type RawRange struct {
	Min string `yaml:"min"`
	Max string `yaml:"max"`
}

type IntRange struct {
	Min int `yaml:"min"`
	Max int `yaml:"max"`
}

type FloatRange struct {
	Min float64
	Max float64
}

// Profile is a validated, numeric resource profile.
type Profile struct {
	CPUCores FloatRange
	MemMB    IntRange
	Num      *IntRange
}

// New validates raw input and returns a numeric profile. CPU must be "Nm",
// memory must be "NM", and max must be >= min for every range.
func New(raw Raw) (Profile, error) {
	cpuMin, err := parseMillis(raw.CPU.Min)
	if err != nil {
		return Profile{}, fmt.Errorf("cpu min: %w", err)
	}
	cpuMax, err := parseMillis(raw.CPU.Max)
	if err != nil {
		return Profile{}, fmt.Errorf("cpu max: %w", err)
	}
	if cpuMax < cpuMin {
		return Profile{}, fmt.Errorf("cpu maximum must be greater or equal to cpu minimum")
	}

	memMin, err := parseMegabytes(raw.Mem.Min)
	if err != nil {
		return Profile{}, fmt.Errorf("mem min: %w", err)
	}
	memMax, err := parseMegabytes(raw.Mem.Max)
	if err != nil {
		return Profile{}, fmt.Errorf("mem max: %w", err)
	}
	if memMax < memMin {
		return Profile{}, fmt.Errorf("mem maximum must be greater or equal to mem minimum")
	}

	p := Profile{
		CPUCores: FloatRange{Min: float64(cpuMin) / 1000.0, Max: float64(cpuMax) / 1000.0},
		MemMB:    IntRange{Min: memMin, Max: memMax},
	}

	if raw.Num != nil {
		if raw.Num.Max < raw.Num.Min {
			return Profile{}, fmt.Errorf("num maximum must be greater or equal to num minimum")
		}
		n := *raw.Num
		p.Num = &n
	}

	return p, nil
}

// MinCoresMilli returns the minimum CPU request as a Kubernetes
// millicore string ("Nm"). CPU values round-trip cleanly from their
// "Nm" input form; the rounding guards against float representation
// drift only.
func (p Profile) MinCoresMilli() string {
	return fmt.Sprintf("%dm", cpuToMilli(p.CPUCores.Min))
}

// MaxCoresMilli returns the maximum CPU limit as a Kubernetes
// millicore string ("Nm").
func (p Profile) MaxCoresMilli() string {
	return fmt.Sprintf("%dm", cpuToMilli(p.CPUCores.Max))
}

// MinMemMiB returns the minimum memory request as a Kubernetes
// mebibyte string ("NMi"). The profile's memory value is carried as
// MemMB but is rendered in mebibyte units to match the convention
// every zote chart wrapper and Kubernetes manifest in the library
// uses.
func (p Profile) MinMemMiB() string {
	return fmt.Sprintf("%dMi", p.MemMB.Min)
}

// MaxMemMiB returns the maximum memory limit as a Kubernetes
// mebibyte string ("NMi").
func (p Profile) MaxMemMiB() string {
	return fmt.Sprintf("%dMi", p.MemMB.Max)
}

func cpuToMilli(cores float64) int {
	return int(math.Round(cores * 1000))
}

func parseMillis(s string) (int, error) {
	if !cpuPattern.MatchString(s) {
		return 0, fmt.Errorf("must be a number ending in 'm', got %q", s)
	}

	return strconv.Atoi(strings.TrimSuffix(s, "m"))
}

func parseMegabytes(s string) (int, error) {
	if !memPattern.MatchString(s) {
		return 0, fmt.Errorf("must be a number ending in 'M', got %q", s)
	}

	return strconv.Atoi(strings.TrimSuffix(s, "M"))
}
