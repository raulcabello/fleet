package simple

import (
	"github.com/rancher/fleet/integrationtests/cli/testenv"
	"os"

	"github.com/rancher/fleet/modules/cli/apply"
	"github.com/rancher/fleet/pkg/apis/fleet.cattle.io/v1alpha1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Fleet apply with yaml resources", Ordered, func() {
	When("apply a folder with yaml resources", func() {
		It("fleet apply is called", func() {
			err := fleetApply("simple", []string{testenv.AssetsPath + "simple"}, &apply.Options{})
			Expect(err).NotTo(HaveOccurred())
		})

		It("then Bundle is created with all the resources", func() {
			Eventually(isBundlePresentAndHasResources).Should(BeTrue())
		})
	})
})

func isBundlePresentAndHasResources() bool {
	bundle, err := env.Fclient.Fleet.Bundle().Get(testenv.Namespace, "assets-simple", metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred())
	isSvcPresent := isResourcePresent(testenv.AssetsPath+"simple/svc.yaml", bundle.Spec.Resources)
	isDeploymentPresent := isResourcePresent(testenv.AssetsPath+"simple/deployment.yaml", bundle.Spec.Resources)

	return isSvcPresent && isDeploymentPresent
}

func isResourcePresent(resourcePath string, resources []v1alpha1.BundleResource) bool {
	resourceFile, err := os.ReadFile(resourcePath)
	Expect(err).NotTo(HaveOccurred())

	for _, resource := range resources {
		if resource.Content == string(resourceFile) {
			return true
		}
	}

	return false
}
