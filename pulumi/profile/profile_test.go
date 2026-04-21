package profile_test

import (
	"testing"

	"github.com/milagre/zote/pulumi/profile"
)

func TestNew_ok(t *testing.T) {
	t.Parallel()

	p, err := profile.New(profile.Raw{
		CPU: profile.RawRange{Min: "100m", Max: "500m"},
		Mem: profile.RawRange{Min: "128M", Max: "512M"},
		Num: &profile.IntRange{Min: 1, Max: 3},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got, want := p.CPUCores.Min, 0.1; got != want {
		t.Errorf("CPUCores.Min: got %v, want %v", got, want)
	}
	if got, want := p.CPUCores.Max, 0.5; got != want {
		t.Errorf("CPUCores.Max: got %v, want %v", got, want)
	}
	if got, want := p.MemMB.Min, 128; got != want {
		t.Errorf("MemMB.Min: got %d, want %d", got, want)
	}
	if got, want := p.MemMB.Max, 512; got != want {
		t.Errorf("MemMB.Max: got %d, want %d", got, want)
	}
	if p.Num == nil || p.Num.Min != 1 || p.Num.Max != 3 {
		t.Errorf("Num: got %+v, want {1 3}", p.Num)
	}
}

func TestNew_optionalNum(t *testing.T) {
	t.Parallel()

	p, err := profile.New(profile.Raw{
		CPU: profile.RawRange{Min: "100m", Max: "100m"},
		Mem: profile.RawRange{Min: "64M", Max: "64M"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Num != nil {
		t.Errorf("Num: got %+v, want nil", p.Num)
	}
}

func TestProfile_renderStrings(t *testing.T) {
	t.Parallel()

	p, err := profile.New(profile.Raw{
		CPU: profile.RawRange{Min: "200m", Max: "1500m"},
		Mem: profile.RawRange{Min: "196M", Max: "256M"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cases := map[string]struct {
		got, want string
	}{
		"MinCoresMilli": {p.MinCoresMilli(), "200m"},
		"MaxCoresMilli": {p.MaxCoresMilli(), "1500m"},
		"MinMemMiB":     {p.MinMemMiB(), "196Mi"},
		"MaxMemMiB":     {p.MaxMemMiB(), "256Mi"},
	}

	for name, tc := range cases {
		if tc.got != tc.want {
			t.Errorf("%s: got %q, want %q", name, tc.got, tc.want)
		}
	}
}

func TestNew_errors(t *testing.T) {
	t.Parallel()

	cases := map[string]profile.Raw{
		"cpu min missing suffix": {
			CPU: profile.RawRange{Min: "100", Max: "500m"},
			Mem: profile.RawRange{Min: "128M", Max: "512M"},
		},
		"cpu max missing suffix": {
			CPU: profile.RawRange{Min: "100m", Max: "500"},
			Mem: profile.RawRange{Min: "128M", Max: "512M"},
		},
		"cpu max less than min": {
			CPU: profile.RawRange{Min: "500m", Max: "100m"},
			Mem: profile.RawRange{Min: "128M", Max: "512M"},
		},
		"mem wrong suffix": {
			CPU: profile.RawRange{Min: "100m", Max: "500m"},
			Mem: profile.RawRange{Min: "128Mi", Max: "512M"},
		},
		"mem max less than min": {
			CPU: profile.RawRange{Min: "100m", Max: "500m"},
			Mem: profile.RawRange{Min: "512M", Max: "128M"},
		},
		"num max less than min": {
			CPU: profile.RawRange{Min: "100m", Max: "100m"},
			Mem: profile.RawRange{Min: "64M", Max: "64M"},
			Num: &profile.IntRange{Min: 5, Max: 2},
		},
	}

	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, err := profile.New(raw); err == nil {
				t.Fatalf("expected error for %q", name)
			}
		})
	}
}
