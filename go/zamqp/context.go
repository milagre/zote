package zamqp

import (
	"context"
)

// Logical context keys for interoperability: zamqp_job_id, zamqp_message_id.

type contextKeyType string

var contextKey contextKeyType = "zamqp_ids"

type contextValues struct {
	jobID     string
	messageID string
}

// Context returns a derived context that carries jobID and messageID for IDs.
func Context(ctx context.Context, jobID, messageID string) context.Context {
	return context.WithValue(ctx, contextKey, contextValues{jobID: jobID, messageID: messageID})
}

// IDs returns the job and message IDs from ctx when set together via Context.
func IDs(ctx context.Context) (jobID, messageID string, ok bool) {
	v, ok := ctx.Value(contextKey).(contextValues)
	if !ok {
		return "", "", false
	}
	if v.jobID == "" || v.messageID == "" {
		return "", "", false
	}
	return v.jobID, v.messageID, true
}
