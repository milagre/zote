// Package stringdata builds Secret .data (base64) from [pulumi.StringOutput] values.
// Prefer Data over stringData: reads round-trip as data only, so stringData tends to churn in preview.
package stringdata

import (
	"encoding/base64"
	"sort"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func SecretData(pairs map[string]pulumi.StringOutput) pulumi.StringMapOutput {
	keys := make([]string, 0, len(pairs))
	for k := range pairs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	in := make([]any, len(keys))
	for i, k := range keys {
		in[i] = pairs[k]
	}

	return pulumi.All(in...).ApplyT(func(args []any) map[string]string {
		out := make(map[string]string, len(keys))
		for i, k := range keys {
			plain := args[i].(string)
			out[k] = base64.StdEncoding.EncodeToString([]byte(plain))
		}
		return out
	}).(pulumi.StringMapOutput)
}
