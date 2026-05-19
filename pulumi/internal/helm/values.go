package helm

import (
	"fmt"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// Values builds [pulumi.Map] from nested map/slice literals. Leaves may be plain Go
// scalars (string, int, int64, float64, bool, nil) or any value satisfying pulumi.Input
// (for example StringOutput or BoolOutput from other resources).
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

	case map[string]string:
		out := pulumi.StringMap{}
		for k, v := range x {
			out[k] = pulumi.String(v)
		}

		return out

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

	case pulumi.Input:
		return x

	default:
		panic(fmt.Sprintf("helm.Values: unsupported type %T for value %v", v, v))
	}
}
