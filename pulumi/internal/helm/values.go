package helm

import (
	"fmt"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// Values converts a plain Go map literal into the pulumi.Map shape
// Helm's Values input expects (accepted by both helm/v3 Release and
// helm/v4 Chart). Wrappers in this library build their opinionated
// default values as static Go literals; Values walks that literal once
// to produce the pulumi input.
//
// Supported leaf types: string, int, int64, float64, bool, nil. Any
// other leaf panics — the helper is only fed literals built inside
// this library, so a panic signals a compile-time bug rather than
// untrusted input drift.
func Values(m map[string]any) pulumi.Map {
	out := pulumi.Map{}
	for k, v := range m {
		out[k] = valueInput(v)
	}

	return out
}

func valueInput(v any) pulumi.Input {
	switch x := v.(type) {
	case map[string]any:
		return Values(x)

	case []any:
		arr := make(pulumi.Array, 0, len(x))
		for _, el := range x {
			arr = append(arr, valueInput(el))
		}

		return arr

	case string:
		return pulumi.String(x)
	case int:
		return pulumi.Int(x)
	case int64:
		return pulumi.Int(int(x))
	case float64:
		return pulumi.Float64(x)
	case bool:
		return pulumi.Bool(x)
	case nil:
		return nil

	default:
		panic(fmt.Sprintf("helm.Values: unsupported type %T for value %v", v, v))
	}
}
