package helm

import (
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rancher/fleet/integrationtests/cli/testenv"
	"github.com/rancher/fleet/modules/cli/apply"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	env *testenv.Env
)

const (
	timeout = 5 * time.Second
	name    = "helm"
)

func TestFleet(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Fleet Suite")
}

var _ = BeforeSuite(func() {
	var err error
	SetDefaultEventuallyTimeout(timeout)
	env, err = testenv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(env.K8sClient.Create(env.Ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: testenv.Namespace,
		},
	})).NotTo(HaveOccurred())
})

var _ = AfterSuite(func() {
	env.Cancel()
	Expect(env.TestEnv.Stop()).ToNot(HaveOccurred())
})

// simulates fleet cli execution
func fleetApply(dirs []string, options *apply.Options) error {
	return apply.Apply(env.Ctx, &testenv.Getter{Env: env}, name, dirs, options)
}
