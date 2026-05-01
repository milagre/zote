// Package cloud is the small shared contract for cloud-specific Service annotations.
package cloud

type Cloud interface {
	PublicLoadBalancerAnnotations() map[string]string
	PrivateLoadBalancerAnnotations() map[string]string
}
