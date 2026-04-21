package container

import (
	"fmt"

	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	rbacv1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/rbac/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// registerRBAC creates the ServiceAccount, Role, and RoleBinding required
// by rabbitmq_peer_discovery_k8s (`get` on endpoints, `create` on events).
func registerRBAC(
	ctx *pulumi.Context,
	parentName string,
	comp pulumi.Resource,
	namespace string,
	releaseName string,
) (*corev1.ServiceAccount, error) {
	ns := pulumi.String(namespace)
	sa, err := corev1.NewServiceAccount(ctx, parentName+"-sa", &corev1.ServiceAccountArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String(releaseName),
			Namespace: ns,
		},
	}, pulumi.Parent(comp))
	if err != nil {
		return nil, fmt.Errorf("service account: %w", err)
	}

	role, err := rbacv1.NewRole(ctx, parentName+"-role", &rbacv1.RoleArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String(releaseName),
			Namespace: ns,
		},
		Rules: rbacv1.PolicyRuleArray{
			&rbacv1.PolicyRuleArgs{
				Verbs:     pulumi.StringArray{pulumi.String("get")},
				ApiGroups: pulumi.StringArray{pulumi.String("")},
				Resources: pulumi.StringArray{pulumi.String("endpoints")},
			},
			&rbacv1.PolicyRuleArgs{
				Verbs:     pulumi.StringArray{pulumi.String("create")},
				ApiGroups: pulumi.StringArray{pulumi.String("")},
				Resources: pulumi.StringArray{pulumi.String("events")},
			},
		},
	}, pulumi.Parent(comp))
	if err != nil {
		return nil, fmt.Errorf("role: %w", err)
	}

	if _, err := rbacv1.NewRoleBinding(ctx, parentName+"-rolebinding", &rbacv1.RoleBindingArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String(releaseName),
			Namespace: ns,
		},
		Subjects: rbacv1.SubjectArray{
			&rbacv1.SubjectArgs{
				Kind:      pulumi.String("ServiceAccount"),
				Name:      sa.Metadata.Name().Elem(),
				Namespace: ns,
			},
		},
		RoleRef: &rbacv1.RoleRefArgs{
			ApiGroup: pulumi.String("rbac.authorization.k8s.io"),
			Kind:     pulumi.String("Role"),
			Name:     role.Metadata.Name().Elem(),
		},
	}, pulumi.Parent(comp)); err != nil {
		return nil, fmt.Errorf("rolebinding: %w", err)
	}

	return sa, nil
}
