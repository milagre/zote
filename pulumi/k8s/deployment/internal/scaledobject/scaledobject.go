// Package scaledobject emits the KEDA ScaledObject (and, when a RabbitMQ
// trigger is present, its TriggerAuthentication) that autoscales a single
// Deployment. It is generic to zote's own usage but opinionated within the
// confines of what zote deploys: it knows the RabbitMQ management-API trigger
// shape and the zamqp consumer-utilization metric naming, so callers supply
// only intent (a queue name, a target utilization) rather than wiring.
//
// Two independent, opt-in triggers are supported:
//
//   - Queue: RabbitMQ queue depth via the management API. It is
//     pod-independent, so it is the only trigger that can reactivate a
//     workload that has scaled to zero.
//   - Utilization: the zamqp consumer utilization gauge read from the stats
//     stack (Mimir). It cannot lift a workload off zero (no pods means no
//     metric), so a zero floor requires a Queue trigger as well.
package scaledobject

import (
	"fmt"
	"strconv"

	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	apiextensions "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/apiextensions"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	apiVersion = "keda.sh/v1alpha1"

	defaultVhost = "/"

	// activationOnlyQueueValue makes the RabbitMQ trigger behave as a pure
	// 0<->1 gate: KEDA's AverageValue target computes ceil(queueLength/value),
	// so a target this large yields exactly one replica for any realistic
	// backlog while still activating the workload the moment a message appears.
	// Sizing the running pool beyond one is left to the utilization trigger.
	// A caller that wants proportional queue scaling sets MessagesPerReplica.
	activationOnlyQueueValue = 100_000_000

	// defaultInitialCooldownSeconds is the grace period a scale-to-zero
	// workload gets before KEDA may take it to zero for the first time. The
	// deployment layer seeds such a workload with one replica precisely so that
	// pod can declare the queue this ScaledObject reads; without a floor on
	// time-to-first-scaledown that pod could be reaped before it finishes
	// booting, leaving the trigger pointed at a queue that does not exist.
	// Sized to cover an image pull and startup, and only ever paid once.
	defaultInitialCooldownSeconds = 300
)

// Spec turns on KEDA autoscaling for one workload. At least one trigger must
// be set; both are optional and independent.
type Spec struct {
	Queue       *QueueTrigger
	Utilization *UtilizationTrigger

	// CooldownSeconds is KEDA's cooldownPeriod: how long a workload must stay
	// idle before scaling back to the floor. Zero uses KEDA's default (300s).
	CooldownSeconds int

	// InitialCooldownSeconds is KEDA's initialCooldownPeriod: how long after
	// this ScaledObject is created KEDA must wait before the first scale to
	// zero. It only applies to a zero floor, where zero takes the bootstrap
	// default (see defaultInitialCooldownSeconds); a nonzero floor ignores it.
	InitialCooldownSeconds int
}

// QueueTrigger scales on RabbitMQ queue depth over the management API.
type QueueTrigger struct {
	// Queue is the queue whose depth drives scaling. Required; never defaulted
	// (queue names live in application topology, not in the deployer).
	Queue string

	// Vhost defaults to "/".
	Vhost string

	// MessagesPerReplica, when > 0, makes the queue trigger scale replicas
	// proportionally to backlog (target messages per replica). When 0 (the
	// default) the trigger is activation-only: it lifts the workload from zero
	// to one on any message and leaves further scaling to other triggers.
	MessagesPerReplica int

	// HostSecretName names a Secret in the workload namespace whose "host" key
	// holds the full RabbitMQ management URI including credentials.
	HostSecretName pulumi.StringInput
}

// UtilizationTrigger scales on the zamqp consumer utilization gauge.
type UtilizationTrigger struct {
	// TargetPercent is the per-replica utilization target (1..100).
	TargetPercent int

	// ServerAddress is the Prometheus-compatible query endpoint (Mimir). The
	// deployment layer fills it from the cluster when left empty.
	ServerAddress string

	// Query is the PromQL expression whose scalar result drives scaling.
	// Required: the workload owns which signal it scales on (see
	// deployment.ZAMQPUtilizationStat for the zamqp-consumer convenience).
	Query string
}

// Args is the input to Register.
type Args struct {
	Namespace string

	// TargetName is the Deployment this ScaledObject scales; it is also the
	// ScaledObject's own name.
	TargetName string

	// Min and Max are the replica bounds (from the workload's profile). Min may
	// be 0 only when a Queue trigger is present.
	Min int
	Max int

	Spec Spec
}

// Register creates the ScaledObject and any supporting TriggerAuthentication
// as children of parent.
func Register(
	ctx *pulumi.Context,
	name string,
	args Args,
	parent pulumi.Resource,
	opts ...pulumi.ResourceOption,
) error {
	if err := args.validate(); err != nil {
		return fmt.Errorf("scaledobject: %w", err)
	}

	childOpts := append([]pulumi.ResourceOption{pulumi.Parent(parent)}, opts...)

	triggers := pulumi.Array{}

	if q := args.Spec.Queue; q != nil {
		auth, err := registerTriggerAuth(ctx, name, args, q, childOpts...)
		if err != nil {
			return err
		}

		triggers = append(triggers, queueTrigger(q, auth))
	}

	if u := args.Spec.Utilization; u != nil {
		triggers = append(triggers, utilizationTrigger(u))
	}

	spec := pulumi.Map{
		"scaleTargetRef": pulumi.Map{
			"name": pulumi.String(args.TargetName),
		},
		"minReplicaCount": pulumi.Int(args.Min),
		"maxReplicaCount": pulumi.Int(args.Max),
		"triggers":        triggers,
	}
	if args.Spec.CooldownSeconds > 0 {
		spec["cooldownPeriod"] = pulumi.Int(args.Spec.CooldownSeconds)
	}
	if initial := args.initialCooldownSeconds(); initial > 0 {
		spec["initialCooldownPeriod"] = pulumi.Int(initial)
	}

	_, err := apiextensions.NewCustomResource(ctx, name, &apiextensions.CustomResourceArgs{
		ApiVersion: pulumi.String(apiVersion),
		Kind:       pulumi.String("ScaledObject"),
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String(args.TargetName),
			Namespace: pulumi.String(args.Namespace),
		},
		OtherFields: kubernetes.UntypedArgs{"spec": spec},
	}, childOpts...)
	if err != nil {
		return fmt.Errorf("scaledobject: %w", err)
	}

	return nil
}

func registerTriggerAuth(
	ctx *pulumi.Context,
	name string,
	args Args,
	q *QueueTrigger,
	opts ...pulumi.ResourceOption,
) (*apiextensions.CustomResource, error) {
	auth, err := apiextensions.NewCustomResource(ctx, name+"-auth", &apiextensions.CustomResourceArgs{
		ApiVersion: pulumi.String(apiVersion),
		Kind:       pulumi.String("TriggerAuthentication"),
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String(args.TargetName + "-rabbitmq"),
			Namespace: pulumi.String(args.Namespace),
		},
		OtherFields: kubernetes.UntypedArgs{
			"spec": pulumi.Map{
				"secretTargetRef": pulumi.Array{
					pulumi.Map{
						"parameter": pulumi.String("host"),
						"name":      q.HostSecretName,
						"key":       pulumi.String("host"),
					},
				},
			},
		},
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("scaledobject: trigger auth: %w", err)
	}

	return auth, nil
}

func queueTrigger(q *QueueTrigger, auth *apiextensions.CustomResource) pulumi.Map {
	vhost := q.Vhost
	if vhost == "" {
		vhost = defaultVhost
	}

	value := q.MessagesPerReplica
	if value <= 0 {
		value = activationOnlyQueueValue
	}

	return pulumi.Map{
		"type": pulumi.String("rabbitmq"),
		"metadata": pulumi.Map{
			"protocol":  pulumi.String("http"),
			"queueName": pulumi.String(q.Queue),
			"vhostName": pulumi.String(vhost),
			"mode":      pulumi.String("QueueLength"),
			"value":     pulumi.String(strconv.Itoa(value)),
		},
		"authenticationRef": pulumi.Map{
			"name": auth.Metadata.Name().Elem(),
		},
	}
}

func utilizationTrigger(u *UtilizationTrigger) pulumi.Map {
	return pulumi.Map{
		"type": pulumi.String("prometheus"),
		// Value (not the KEDA default AverageValue) makes the HPA treat the
		// avg-utilization query like a CPU-style utilization target:
		// desired = ceil(replicas * currentAvg / target), which scales the pool
		// up as long as the average stays above target. AverageValue would
		// instead cap scaling at ceil(currentAvg / target).
		"metricType": pulumi.String("Value"),
		"metadata": pulumi.Map{
			"serverAddress": pulumi.String(u.ServerAddress),
			"query":         pulumi.String(u.Query),
			"threshold":     pulumi.String(strconv.Itoa(u.TargetPercent)),
		},
	}
}

// initialCooldownSeconds is the grace period before the first scale to zero.
// It is meaningful only for a zero floor, which is also the only case the
// deployment layer bootstraps with a seeded replica; a workload that never
// reaches zero has nothing to protect. Zero on a zero floor takes the default
// rather than KEDA's (which is no grace period at all).
func (a Args) initialCooldownSeconds() int {
	if a.Min > 0 {
		return 0
	}
	if a.Spec.InitialCooldownSeconds > 0 {
		return a.Spec.InitialCooldownSeconds
	}

	return defaultInitialCooldownSeconds
}

func (a Args) validate() error {
	if a.Namespace == "" {
		return fmt.Errorf("Namespace is required")
	}
	if a.TargetName == "" {
		return fmt.Errorf("TargetName is required")
	}
	if a.Spec.Queue == nil && a.Spec.Utilization == nil {
		return fmt.Errorf("at least one trigger (Queue or Utilization) is required")
	}
	if a.Min < 0 {
		return fmt.Errorf("Min must be >= 0")
	}
	if a.Max < 1 {
		return fmt.Errorf("Max must be >= 1")
	}
	if a.Max < a.Min {
		return fmt.Errorf("Max must be >= Min")
	}

	// A zero floor can only be reactivated by a pod-independent trigger; the
	// utilization metric vanishes with the last pod.
	if a.Min == 0 && a.Spec.Queue == nil {
		return fmt.Errorf("Min=0 requires a Queue trigger to reactivate from zero")
	}

	if q := a.Spec.Queue; q != nil {
		if q.Queue == "" {
			return fmt.Errorf("Queue.Queue is required")
		}
		if q.HostSecretName == nil {
			return fmt.Errorf("Queue.HostSecretName is required")
		}
	}
	if u := a.Spec.Utilization; u != nil {
		if u.TargetPercent < 1 || u.TargetPercent > 100 {
			return fmt.Errorf("Utilization.TargetPercent must be within 1..100")
		}
		if u.ServerAddress == "" {
			return fmt.Errorf("Utilization.ServerAddress is required")
		}
		if u.Query == "" {
			return fmt.Errorf("Utilization.Query is required")
		}
	}

	return nil
}
