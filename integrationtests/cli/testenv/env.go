package testenv

import (
	"context"
	"path/filepath"

	fleetclient "github.com/rancher/fleet/modules/cli/pkg/client"
	"github.com/rancher/fleet/pkg/apis/fleet.cattle.io/v1alpha1"
	"github.com/rancher/fleet/pkg/generated/controllers/fleet.cattle.io"
	wranglerapply "github.com/rancher/wrangler/pkg/apply"
	"github.com/rancher/wrangler/pkg/generated/controllers/core"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

const (
	Namespace  = "fleet-cli-integration-tests"
	AssetsPath = "../assets/"
)

type Env struct {
	TestEnv   *envtest.Environment
	Ctx       context.Context
	Cancel    context.CancelFunc
	K8sClient client.Client
	Fclient   *fleetclient.Client
}

func Start() (*Env, error) {
	ctx, cancel := context.WithCancel(context.TODO())

	testEnv := &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "..", "charts", "fleet-crd", "templates", "crds.yaml")},
		ErrorIfCRDPathMissing: true,
	}

	var err error
	cfg, err := testEnv.Start()
	if err != nil {
		cancel()
		return nil, err
	}

	customScheme := scheme.Scheme
	customScheme.AddKnownTypes(schema.GroupVersion{Group: "fleet.cattle.io", Version: "v1alpha1"}, &v1alpha1.Bundle{}, &v1alpha1.BundleList{})
	customScheme.AddKnownTypes(schema.GroupVersion{Group: "fleet.cattle.io", Version: "v1alpha1"}, &v1alpha1.BundleDeployment{}, &v1alpha1.BundleDeploymentList{})

	k8sClient, err := client.New(cfg, client.Options{Scheme: customScheme})
	if err != nil {
		cancel()
		return nil, err
	}

	fclient := &fleetclient.Client{
		Namespace: Namespace,
	}
	fleet, err := fleet.NewFactoryFromConfig(cfg)
	if err != nil {
		cancel()
		return nil, err
	}
	fclient.Fleet = fleet.Fleet().V1alpha1()
	core, err := core.NewFactoryFromConfig(cfg)
	if err != nil {
		cancel()
		return nil, err
	}
	fclient.Core = core.Core().V1()
	fclient.Apply, err = wranglerapply.NewForConfig(cfg)
	if err != nil {
		cancel()
		return nil, err
	}
	fclient.Apply = fclient.Apply.
		WithDynamicLookup().
		WithDefaultNamespace(fclient.Namespace).
		WithListerNamespace(fclient.Namespace).
		WithRestrictClusterScoped()

	return &Env{
		Ctx:       ctx,
		Cancel:    cancel,
		K8sClient: k8sClient,
		Fclient:   fclient,
		TestEnv:   testEnv,
	}, nil

}

type Getter struct {
	Env *Env
}

func (g *Getter) Get() (*fleetclient.Client, error) {
	return g.Env.Fclient, nil
}

func (g *Getter) GetNamespace() string {
	return Namespace
}
