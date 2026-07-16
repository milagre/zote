package dashboard

import (
	"strings"
	"testing"

	"github.com/milagre/zote/pulumi/env"
)

func TestRenderZAMQPConsumerDashboard(t *testing.T) {
	e, err := env.New("wm", "local", "dev", "local", "/root", "WM")
	if err != nil {
		t.Fatalf("env.New: %v", err)
	}

	got, err := render(Spec{
		Env:       e,
		Namespace: "finance",
		Name:      "account-analyzer",
		Process:   "zamqp-consumer",
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	wantMetric := "wm_finance_account_analyzer_zamqp_consumer_utilization"
	if !strings.Contains(got, wantMetric) {
		t.Fatalf("dashboard missing utilization metric %q", wantMetric)
	}

	wantReceived := "wm_finance_account_analyzer_zamqp_consumer_received"
	if !strings.Contains(got, wantReceived) {
		t.Fatalf("dashboard missing received metric %q", wantReceived)
	}

	if !strings.Contains(got, `"title": "Finance: Account Analyzer"`) {
		t.Fatalf("dashboard title not rendered")
	}

	if !strings.Contains(got, `"uid": "finance-account-analyzer"`) {
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
	e, err := env.New("wm", "local", "dev", "local", "/root", "WM")
	if err != nil {
		t.Fatalf("env.New: %v", err)
	}

	got, err := render(Spec{
		Env:       e,
		Namespace: "finance",
		Name:      "api",
		Process:   "zapi",
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	wantRequests := "wm_finance_api_zapi_requests"
	if !strings.Contains(got, wantRequests) {
		t.Fatalf("dashboard missing requests metric %q", wantRequests)
	}

	wantResponses := "wm_finance_api_zapi_responses"
	if !strings.Contains(got, wantResponses) {
		t.Fatalf("dashboard missing responses metric %q", wantResponses)
	}

	if !strings.Contains(got, `"title": "Finance: Api"`) {
		t.Fatalf("dashboard title not rendered")
	}
}

func TestDashboardTitle(t *testing.T) {
	got := dashboardTitle("backend", "client-events-processor")
	want := "Backend: Client Events Processor"
	if got != want {
		t.Fatalf("dashboardTitle = %q, want %q", got, want)
	}
}

func TestRenderUnsupportedProcessType(t *testing.T) {
	e, err := env.New("wm", "local", "dev", "local", "/root", "WM")
	if err != nil {
		t.Fatalf("env.New: %v", err)
	}

	_, err = render(Spec{
		Env:       e,
		Namespace: "finance",
		Name:      "api",
		Process:   "cron",
	})
	if err == nil {
		t.Fatal("render = nil, want error")
	}
}
