// Package stringdata builds kubernetes Secret Data maps from sensitive
// outputs without nesting bare outputs inside pulumi.StringMap (which can
// reach the API as empty values).
//
// Use SecretData with corev1.SecretArgs.Data, not StringData: the API
// never returns stringData on read (only base64 data), so using StringData
// causes perpetual preview diffs against state while applies are often
// no-ops when the cluster already holds the same bytes.
package stringdata

import (
	"encoding/base64"
	"sort"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// SecretData merges keyed pulumi.StringOutput values into one
// StringMapOutput suitable for corev1.SecretArgs.Data. Each value is
// RFC4648 base64-encoded as required by the Kubernetes API.
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
