package container

import (
	"encoding/json"
	"fmt"
	"strings"

	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi-random/sdk/v4/go/random"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/milagre/zote/pulumi/env"
	"github.com/milagre/zote/pulumi/stringdata"
)

const (
	adminUser = "admin"
	username  = "rabbitmq"

	hashingAlgorithm = "rabbit_password_hashing_sha512"
)

var enabledPlugins = []string{
	"rabbitmq_management",
	"rabbitmq_peer_discovery_k8s",
	"rabbitmq_prometheus",
	"rabbitmq_shovel_management",
	"rabbitmq_shovel",
}

var randomBytesIgnoredArgs = []string{"length"}

// userCreds pairs a per-user random salt with its random password so the
// password-hash computation can use both as a single Apply input.
type userCreds struct {
	salt     *random.RandomBytes
	password *random.RandomPassword
}

// registerCreds generates a RandomBytes salt and a RandomPassword for each
// user (including the synthetic admin). Returned map is keyed by user name.
func registerCreds(ctx *pulumi.Context, parentName string, comp pulumi.Resource, users []string, e env.Env) (map[string]userCreds, error) {
	creds := make(map[string]userCreds, len(users))
	for _, u := range users {
		salt, err := random.NewRandomBytes(ctx, fmt.Sprintf("%s-salt-%s", parentName, u), &random.RandomBytesArgs{
			Length: pulumi.Int(4),
		}, pulumi.Parent(comp), pulumi.IgnoreChanges(randomBytesIgnoredArgs))
		if err != nil {
			return nil, fmt.Errorf("salt for %q: %w", u, err)
		}

		password, err := random.NewRandomPassword(ctx, fmt.Sprintf("%s-password-%s", parentName, u), &random.RandomPasswordArgs{
			Length:   pulumi.Int(32),
			Numeric:  pulumi.Bool(true),
			Upper:    pulumi.Bool(true),
			Lower:    pulumi.Bool(true),
			Special:  pulumi.Bool(false),
			Keepers:  e.RandomKeepers(nil),
		},
			pulumi.Parent(comp),
			pulumi.IgnoreChanges(randomPasswordIgnoredArgs),
		)
		if err != nil {
			return nil, fmt.Errorf("password for %q: %w", u, err)
		}

		creds[u] = userCreds{salt: salt, password: password}
	}

	return creds, nil
}

// definitionsJSON returns a StringOutput that resolves to the JSON-encoded
// `definitions.json` body for importing permissions/users/vhosts on boot.
// It uses pulumi.All to wait on every user's salt+password output before
// computing each password hash in-process.
func definitionsJSON(users []User, vhosts []Vhost, creds map[string]userCreds) pulumi.StringOutput {
	inputs := make([]interface{}, 0, 2*len(users))
	order := make([]string, 0, len(users))
	for _, u := range users {
		c := creds[u.Name]
		order = append(order, u.Name)
		inputs = append(inputs, c.salt.Base64, c.password.Result)
	}

	return pulumi.All(inputs...).ApplyT(func(vs []interface{}) (string, error) {
		type userDef struct {
			HashingAlgorithm string   `json:"hashing_algorithm"`
			Name             string   `json:"name"`
			PasswordHash     string   `json:"password_hash"`
			Tags             []string `json:"tags"`
		}
		type permDef struct {
			User      string `json:"user"`
			Vhost     string `json:"vhost"`
			Configure string `json:"configure"`
			Read      string `json:"read"`
			Write     string `json:"write"`
		}
		type vhostDef struct {
			Name string `json:"name"`
		}

		byName := make(map[string]struct {
			salt string
			pass string
		}, len(order))
		for i, name := range order {
			byName[name] = struct {
				salt string
				pass string
			}{
				salt: vs[2*i].(string),
				pass: vs[2*i+1].(string),
			}
		}

		userDefs := make([]userDef, 0, len(users))
		for _, u := range users {
			c := byName[u.Name]
			hash, err := rabbitPasswordHashSHA512(c.salt, c.pass)
			if err != nil {
				return "", fmt.Errorf("hashing %q: %w", u.Name, err)
			}
			userDefs = append(userDefs, userDef{
				HashingAlgorithm: hashingAlgorithm,
				Name:             u.Name,
				PasswordHash:     hash,
				Tags:             u.Tags,
			})
		}

		permDefs := make([]permDef, 0)
		for _, v := range vhosts {
			for _, u := range v.Users {
				permDefs = append(permDefs, permDef{
					User: u, Vhost: v.Name,
					Configure: ".*", Read: ".*", Write: ".*",
				})
			}
			permDefs = append(permDefs, permDef{
				User: adminUser, Vhost: v.Name,
				Configure: ".*", Read: ".*", Write: ".*",
			})
		}

		vhostDefs := make([]vhostDef, 0, len(vhosts))
		for _, v := range vhosts {
			vhostDefs = append(vhostDefs, vhostDef{Name: v.Name})
		}

		out := struct {
			Permissions []permDef  `json:"permissions"`
			Users       []userDef  `json:"users"`
			Vhosts      []vhostDef `json:"vhosts"`
		}{permDefs, userDefs, vhostDefs}

		b, err := json.Marshal(out)
		if err != nil {
			return "", fmt.Errorf("marshaling definitions: %w", err)
		}

		return string(b), nil
	}).(pulumi.StringOutput)
}

// rabbitmqConf renders the static/dynamic rabbitmq.conf body. The admin
// password flows through because the stanza "default_pass = ..." needs it
// cleartext at config-load time.
func rabbitmqConf(releaseName string, adminPassword pulumi.StringOutput) pulumi.StringOutput {
	return adminPassword.ApplyT(func(p string) string {
		return strings.Join([]string{
			"cluster_formation.peer_discovery_backend = k8s",
			"cluster_formation.k8s.host = kubernetes.default.svc.cluster.local",
			"cluster_formation.k8s.address_type = hostname",
			"cluster_formation.k8s.service_name = " + releaseName + "-headless",
			"",
			"definitions.import_backend = local_filesystem",
			"definitions.local.path = /etc/rabbitmq/definitions.json",
			"",
			"queue_master_locator=min-masters",
			"",
			"loopback_users.admin = true",
			"default_vhost = /",
			"default_user = admin",
			"default_pass = " + p,
			"default_permissions.configure = .*",
			"default_permissions.read = .*",
			"default_permissions.write = .*",
			"default_user_tags.administrator = true",
			"default_user_tags.management = true",
			"",
		}, "\n")
	}).(pulumi.StringOutput)
}

// enabledPluginsContents returns the contents of the enabled_plugins file
// (an Erlang-style list literal).
func enabledPluginsContents() string {
	return "[" + strings.Join(enabledPlugins, ",") + "].\n"
}

// configResources creates the cluster-wide ConfigMap (enabled_plugins,
// rabbitmq.conf, definitions.json, username) and Secret (admin password,
// erlang cookie).
func configResources(
	ctx *pulumi.Context,
	parentName string,
	comp pulumi.Resource,
	namespace string,
	releaseName string,
	creds map[string]userCreds,
	users []User,
	vhosts []Vhost,
	erlangCookie *random.RandomPassword,
	password *random.RandomPassword,
) (*corev1.ConfigMap, *corev1.Secret, error) {
	ns := pulumi.String(namespace)
	patchForce := pulumi.StringMap{"pulumi.com/patchForce": pulumi.String("true")}
	cm, err := corev1.NewConfigMap(ctx, parentName+"-cfg", &corev1.ConfigMapArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:        pulumi.String("cfg-" + releaseName),
			Namespace:   ns,
			Annotations: patchForce,
		},
		Data: pulumi.StringMap{
			"enabled_plugins":  pulumi.String(enabledPluginsContents()),
			"definitions.json": definitionsJSON(users, vhosts, creds),
			"rabbitmq.conf":    rabbitmqConf(releaseName, creds[adminUser].password.Result),
			"username":         pulumi.String(username),
		},
	}, pulumi.Parent(comp))
	if err != nil {
		return nil, nil, fmt.Errorf("config configmap: %w", err)
	}

	secretData := stringdata.SecretData(map[string]pulumi.StringOutput{
		"password":      password.Result,
		"erlang_cookie": erlangCookie.Result,
	})

	sec, err := corev1.NewSecret(ctx, parentName+"-cfg", &corev1.SecretArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:        pulumi.String("cfg-" + releaseName),
			Namespace:   ns,
			Annotations: patchForce,
		},
		Type: pulumi.String("Opaque"),
		Data: secretData,
	}, pulumi.Parent(comp))
	if err != nil {
		return nil, nil, fmt.Errorf("config secret: %w", err)
	}

	return cm, sec, nil
}
