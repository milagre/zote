// Package tokens centralizes the Pulumi type-token namespace used by the
// Zote library. A single package means renames (which break URN stability)
// are impossible to do accidentally in a component package.
package tokens

// Prefix is the global namespace prefix for every Zote component.
const Prefix = "zote"

// Token constructs a full type token from a module path (e.g. "infra") and
// a resource name (e.g. "Grafana"). The shape is `zote:<module>:<Name>`.
func Token(module, name string) string {
	return Prefix + ":" + module + ":" + name
}

// Qualify returns a Kubernetes-namespace-qualified Pulumi logical name
// (e.g. Qualify("backend", "api") == "backend-api"). Every Zote
// component uses this form for its URN so two namespaces that share
// a workload name (backend/api and finance/api) stay distinct in the
// Pulumi resource graph. When namespace is empty the bare name is
// returned so non-namespaced callers keep working.
func Qualify(namespace, name string) string {
	if namespace == "" {
		return name
	}

	return namespace + "-" + name
}
