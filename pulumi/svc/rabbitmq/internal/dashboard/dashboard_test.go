package dashboard

import (
	"strings"
	"testing"
)

func TestRender(t *testing.T) {
	got, err := render()
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	if strings.Contains(got, "__DATASOURCE__") {
		t.Fatal("dashboard still contains datasource placeholder")
	}

	if !strings.Contains(got, `"uid": "mimir"`) {
		t.Fatal("dashboard datasource not bound to mimir uid")
	}

	if !strings.Contains(got, `"uid": "rabbitmq"`) {
		t.Fatal("dashboard uid not present")
	}

	if !strings.Contains(got, `"title": "RabbitMQ"`) {
		t.Fatal("dashboard title not present")
	}

	if !strings.Contains(got, `rabbitmq_queue_messages{namespace=\"$namespace\", name=\"$cluster\"`) {
		t.Fatal("dashboard queue query not present")
	}
}
