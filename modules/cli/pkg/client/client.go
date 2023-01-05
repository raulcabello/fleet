package client

import (
	"fmt"

	"github.com/rancher/fleet/pkg/generated/controllers/fleet.cattle.io"
	fleetcontrollers "github.com/rancher/fleet/pkg/generated/controllers/fleet.cattle.io/v1alpha1"
	"github.com/rancher/wrangler/pkg/apply"
	"github.com/rancher/wrangler/pkg/generated/controllers/core"
	corev1 "github.com/rancher/wrangler/pkg/generated/controllers/core/v1"
	"github.com/rancher/wrangler/pkg/kubeconfig"
)

type Getter interface {
	Get() (*Client, error)
	GetNamespace() string
}

type getter struct {
	Kubeconfig string
	Context    string
	Namespace  string
}

func (g *getter) Get() (*Client, error) {
	if g == nil {
		return nil, fmt.Errorf("client is not configured, please set client getter")
	}
	return NewClient(g.Kubeconfig, g.Context, g.Namespace)
}

func (g *getter) GetNamespace() string {
	return g.Namespace
}

type Client struct {
	Fleet     fleetcontrollers.Interface
	Core      corev1.Interface
	Apply     apply.Apply
	Namespace string
}

func NewGetterWithNamespace(namespace string) Getter {
	return &getter{Namespace: namespace}
}

func NewGetter(kubeconfig, context, namespace string) Getter {
	return &getter{
		Kubeconfig: kubeconfig,
		Context:    context,
		Namespace:  namespace,
	}
}

func NewClient(kubeConfig, context, namespace string) (*Client, error) {
	cc := kubeconfig.GetNonInteractiveClientConfigWithContext(kubeConfig, context)
	ns, _, err := cc.Namespace()
	if err != nil {
		return nil, err
	}

	if namespace != "" {
		ns = namespace
	}

	restConfig, err := cc.ClientConfig()
	if err != nil {
		return nil, err
	}

	c := &Client{
		Namespace: ns,
	}

	fleet, err := fleet.NewFactoryFromConfig(restConfig)
	if err != nil {
		return nil, err
	}
	c.Fleet = fleet.Fleet().V1alpha1()

	core, err := core.NewFactoryFromConfig(restConfig)
	if err != nil {
		return nil, err
	}
	c.Core = core.Core().V1()

	c.Apply, err = apply.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	if c.Namespace == "" {
		c.Namespace = "default"
	}

	c.Apply = c.Apply.
		WithDynamicLookup().
		WithDefaultNamespace(c.Namespace).
		WithListerNamespace(c.Namespace).
		WithRestrictClusterScoped()

	return c, nil
}
