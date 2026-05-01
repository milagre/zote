// Package tokens is the shared Pulumi type-token prefix for zote components (URN stability).
package tokens

const Prefix = "zote"

func Token(module, name string) string {
	return Prefix + ":" + module + ":" + name
}

// Qualify builds "<namespace>-<name>" for distinct URNs when the same workload name appears in different namespaces.
func Qualify(namespace, name string) string {
	if namespace == "" {
		return name
	}

	return namespace + "-" + name
}
