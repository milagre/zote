package helm

import (
	"fmt"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// Values builds [pulumi.Map] from nested map/slice literals (leaves: string, int, int64, float64, bool, nil; else panic).
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
