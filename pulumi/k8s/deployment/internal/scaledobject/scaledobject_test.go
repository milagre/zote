package scaledobject

import (
	"strings"
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func validQueueTrigger() *QueueTrigger {
	return &QueueTrigger{
		Queue:          "jobs",
		HostSecretName: pulumi.String("amqp-apps-monitor"),
	}
}

func validUtilizationTrigger() *UtilizationTrigger {
	return &UtilizationTrigger{
		TargetPercent: 80,
		ServerAddress: "http://mimir.infra.svc:80/prometheus",
		Query:         `avg({__name__="x"})`,
	}
}

// TestArgsValidate isolates each rule so the returned message is trustworthy at
// the call site. The safety-critical rule is that a zero floor demands a
// pod-independent (queue) trigger.
func TestArgsValidate(t *testing.T) {
	base := func() Args {
		return Args{
			Namespace:  "apps",
			TargetName: "worker",
			Min:        0,
			Max:        5,
			Spec:       Spec{Queue: validQueueTrigger()},
		}
	}

	tests := []struct {
		name      string
		mutate    func(*Args)
		wantError string
	}{
		{"queue-only zero floor", func(*Args) {}, ""},
		{"utilization added", func(a *Args) { a.Spec.Utilization = validUtilizationTrigger() }, ""},
		{
			"utilization-only with nonzero floor",
			func(a *Args) {
				a.Min = 1
				a.Spec = Spec{Utilization: validUtilizationTrigger()}
			},
			"",
		},
		{"missing namespace", func(a *Args) { a.Namespace = "" }, "Namespace is required"},
		{"missing target", func(a *Args) { a.TargetName = "" }, "TargetName is required"},
		{
			"no triggers",
			func(a *Args) { a.Spec = Spec{} },
			"at least one trigger",
		},
		{
			"zero floor without queue trigger",
			func(a *Args) {
				a.Min = 0
				a.Spec = Spec{Utilization: validUtilizationTrigger()}
			},
			"Min=0 requires a Queue trigger",
		},
		{"max below min", func(a *Args) { a.Min = 3; a.Max = 2 }, "Max must be >= Min"},
		{"max below one", func(a *Args) { a.Min = 0; a.Max = 0 }, "Max must be >= 1"},
		{
			"queue trigger without queue name",
			func(a *Args) { a.Spec.Queue.Queue = "" },
			"Queue.Queue is required",
		},
		{
			"queue trigger without host secret",
			func(a *Args) { a.Spec.Queue.HostSecretName = nil },
			"Queue.HostSecretName is required",
		},
		{
			"utilization out of range",
			func(a *Args) {
				a.Min = 1
				a.Spec = Spec{Utilization: validUtilizationTrigger()}
				a.Spec.Utilization.TargetPercent = 0
			},
			"TargetPercent must be within 1..100",
		},
		{
			"utilization without server address",
			func(a *Args) {
				a.Min = 1
				a.Spec = Spec{Utilization: validUtilizationTrigger()}
				a.Spec.Utilization.ServerAddress = ""
			},
			"Utilization.ServerAddress is required",
		},
		{
			"utilization without query",
			func(a *Args) {
				a.Min = 1
				a.Spec = Spec{Utilization: validUtilizationTrigger()}
				a.Spec.Utilization.Query = ""
			},
			"Utilization.Query is required",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			a := base()
			tc.mutate(&a)
			err := a.validate()
			switch {
			case tc.wantError == "" && err != nil:
				t.Fatalf("validate() = %v, want nil", err)
			case tc.wantError != "" && err == nil:
				t.Fatalf("validate() = nil, want error %q", tc.wantError)
			case tc.wantError != "" && !strings.Contains(err.Error(), tc.wantError):
				t.Fatalf("validate() = %v, want error containing %q", err, tc.wantError)
			}
		})
	}
}
