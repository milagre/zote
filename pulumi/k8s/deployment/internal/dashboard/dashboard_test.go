package dashboard

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/milagre/zote/pulumi/env"
)

func TestRenderZAMQPConsumerDashboard(t *testing.T) {
	e, err := env.New("zote", "local", "dev", "local", "/root", "APP")
	if err != nil {
		t.Fatalf("env.New: %v", err)
	}

	got, err := render(Spec{
		Env:       e,
		Namespace: "apps",
		Name:      "my-worker",
		Process:   "zamqp-consumer",
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	wantMetric := "app_apps_my_worker_zamqp_consumer_utilization"
	if !strings.Contains(got, wantMetric) {
		t.Fatalf("dashboard missing utilization metric %q", wantMetric)
	}

	wantReceived := "app_apps_my_worker_zamqp_consumer_received"
	if !strings.Contains(got, wantReceived) {
		t.Fatalf("dashboard missing received metric %q", wantReceived)
	}

	if !strings.Contains(got, `"title": "Apps: My Worker"`) {
		t.Fatalf("dashboard title not rendered")
	}

	if !strings.Contains(got, `"uid": "apps-my-worker"`) {
		t.Fatalf("dashboard uid not rendered")
	}

	if !strings.Contains(got, `"uid": "mimir"`) {
		t.Fatalf("dashboard datasource not bound to mimir uid")
	}

	if strings.Contains(got, "__DATASOURCE__") {
		t.Fatalf("dashboard still contains datasource placeholder")
	}
}

func TestRenderZAPIDashboard(t *testing.T) {
	e, err := env.New("zote", "local", "dev", "local", "/root", "APP")
	if err != nil {
		t.Fatalf("env.New: %v", err)
	}

	got, err := render(Spec{
		Env:       e,
		Namespace: "apps",
		Name:      "my-api",
		Process:   "zapi",
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	wantRequests := "app_apps_my_api_zapi_requests"
	if !strings.Contains(got, wantRequests) {
		t.Fatalf("dashboard missing requests metric %q", wantRequests)
	}

	wantResponses := "app_apps_my_api_zapi_responses"
	if !strings.Contains(got, wantResponses) {
		t.Fatalf("dashboard missing responses metric %q", wantResponses)
	}

	wantBusy := "app_apps_my_api_zapi_busy_seconds"
	if !strings.Contains(got, wantBusy) {
		t.Fatalf("dashboard missing busy seconds metric %q", wantBusy)
	}

	if !strings.Contains(got, `"title": "Apps: My Api"`) {
		t.Fatalf("dashboard title not rendered")
	}
}

// Autoscale bounds are drawn on the scaling signal's own panel, and the axis
// reaches capacity even while the signal sits far below it.
func TestRenderZAPIDashboardScaleBounds(t *testing.T) {
	e, err := env.New("zote", "local", "dev", "local", "/root", "APP")
	if err != nil {
		t.Fatalf("env.New: %v", err)
	}

	spec := Spec{Env: e, Namespace: "apps", Name: "my-api", Process: "zapi"}

	assertScaleBounds(t, spec, ZAPIBusySecondsMetric(e, "apps", "my-api"), 8, 10)
}

func TestRenderZAMQPConsumerDashboardScaleBounds(t *testing.T) {
	e, err := env.New("zote", "local", "dev", "local", "/root", "APP")
	if err != nil {
		t.Fatalf("env.New: %v", err)
	}

	spec := Spec{Env: e, Namespace: "apps", Name: "my-worker", Process: "zamqp-consumer"}

	assertScaleBounds(t, spec, ZAMQPConsumerUtilizationMetric(e, "apps", "my-worker"), 70, 100)
}

// Each process dashboard charts the workload's own Deployment replicas, matched
// on the namespace and name the Deployment is created with.
func TestRenderDashboardPods(t *testing.T) {
	e, err := env.New("zote", "local", "dev", "local", "/root", "APP")
	if err != nil {
		t.Fatalf("env.New: %v", err)
	}

	for _, process := range []string{"zapi", "zamqp-consumer"} {
		got, err := render(Spec{Env: e, Namespace: "apps", Name: "my-worker", Process: process})
		if err != nil {
			t.Fatalf("%s: render: %v", process, err)
		}

		want := `kube_deployment_status_replicas_available{namespace=\"apps\", deployment=\"my-worker\"}`
		if !strings.Contains(got, want) {
			t.Errorf("%s: dashboard missing %s", process, want)
		}
	}
}

func TestDashboardTitle(t *testing.T) {
	got := dashboardTitle("apps", "my-worker")
	want := "Apps: My Worker"
	if got != want {
		t.Fatalf("dashboardTitle = %q, want %q", got, want)
	}
}

func TestRenderUnsupportedProcessType(t *testing.T) {
	e, err := env.New("zote", "local", "dev", "local", "/root", "APP")
	if err != nil {
		t.Fatalf("env.New: %v", err)
	}

	_, err = render(Spec{
		Env:       e,
		Namespace: "apps",
		Name:      "my-api",
		Process:   "cron",
	})
	if err == nil {
		t.Fatal("render = nil, want error")
	}
}

type panelDefaults struct {
	Custom struct {
		AxisSoftMax     *float64 `json:"axisSoftMax"`
		ThresholdsStyle struct {
			Mode string `json:"mode"`
		} `json:"thresholdsStyle"`
	} `json:"custom"`
	Thresholds struct {
		Steps []struct {
			Value *float64 `json:"value"`
		} `json:"steps"`
	} `json:"thresholds"`
}

// assertScaleBounds checks that the panel charting metric draws no bounds for
// spec as given, and draws target and capacity once spec carries them.
func assertScaleBounds(t *testing.T, spec Spec, metric string, target, capacity float64) {
	t.Helper()

	defaults := panelDefaultsCharting(t, spec, metric)
	if got := defaults.Custom.ThresholdsStyle.Mode; got != "off" {
		t.Fatalf("without bounds: thresholds style = %q, want off", got)
	}

	spec.Capacity = capacity
	spec.Target = target

	defaults = panelDefaultsCharting(t, spec, metric)
	if got := defaults.Custom.ThresholdsStyle.Mode; got == "off" {
		t.Fatalf("with bounds: thresholds not drawn")
	}
	if defaults.Custom.AxisSoftMax == nil || *defaults.Custom.AxisSoftMax != capacity {
		t.Fatalf("with bounds: axisSoftMax = %v, want %v", defaults.Custom.AxisSoftMax, capacity)
	}

	var values []float64
	for _, step := range defaults.Thresholds.Steps {
		if step.Value != nil {
			values = append(values, *step.Value)
		}
	}
	if len(values) != 2 || values[0] != target || values[1] != capacity {
		t.Fatalf("with bounds: threshold values = %v, want [%v %v]", values, target, capacity)
	}
}

// panelDefaultsCharting renders spec and returns the field defaults of the
// panel charting metric.
func panelDefaultsCharting(t *testing.T, spec Spec, metric string) panelDefaults {
	t.Helper()

	got, err := render(spec)
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	var dashboard struct {
		Panels []struct {
			FieldConfig struct {
				Defaults panelDefaults `json:"defaults"`
			} `json:"fieldConfig"`
			Targets []struct {
				Expr string `json:"expr"`
			} `json:"targets"`
		} `json:"panels"`
	}
	if err := json.Unmarshal([]byte(got), &dashboard); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	for _, panel := range dashboard.Panels {
		for _, target := range panel.Targets {
			if strings.Contains(target.Expr, metric) {
				return panel.FieldConfig.Defaults
			}
		}
	}

	t.Fatalf("no panel charts %q", metric)
	return panelDefaults{}
}
