package helm

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/rancher/fleet/integrationtests/cli/testenv"
	"github.com/rancher/fleet/modules/cli/apply"
	"github.com/rancher/fleet/pkg/apis/fleet.cattle.io/v1alpha1"
	"github.com/rancher/fleet/pkg/bundlereader"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	numberOfFilesInHelmTestRepo = 9
	bundleName                  = "assets-helm"
)

var repo = repository{}

var _ = Describe("Fleet apply helm release", Ordered, func() {
	When("apply a folder with fleet.yaml that contains a helm release", func() {
		When("no auth is required", func() {
			BeforeAll(func() {
				repo.startRepository(false)
			})
			AfterAll(func() {
				err := repo.stopRepository()
				Expect(err).NotTo(HaveOccurred())
				err = env.Fclient.Fleet.Bundle().Delete(testenv.Namespace, bundleName, &metav1.DeleteOptions{})
				Expect(err).NotTo(HaveOccurred())
			})
			It("fleet apply success", func() {
				err := fleetApply([]string{testenv.AssetsPath + "helm"}, &apply.Options{})
				Expect(err).NotTo(HaveOccurred())
			})

			It("then Bundle is created with all the resources inside of the helm release", func() {
				Eventually(verifyResourcesArePresent).Should(BeTrue())
			})
		})
		When("auth is required, and it is not provided", func() {
			BeforeAll(func() {
				repo.startRepository(true)
			})
			AfterAll(func() {
				err := repo.stopRepository()
				Expect(err).NotTo(HaveOccurred())
			})
			It("fleet apply fails when no auth provided", func() {
				err := fleetApply([]string{testenv.AssetsPath + "helm"}, &apply.Options{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).Should(Equal("failed to read helm repo from http://localhost:3000/index.yaml, error code: 401, response body: Unauthorised."))
			})
		})
		When("auth is required, and it is provided without --helm-repo-url-regex", func() {
			BeforeAll(func() {
				repo.startRepository(true)
			})
			AfterAll(func() {
				err := repo.stopRepository()
				Expect(err).NotTo(HaveOccurred())
				err = env.Fclient.Fleet.Bundle().Delete(testenv.Namespace, bundleName, &metav1.DeleteOptions{})
				Expect(err).NotTo(HaveOccurred())
			})
			It("fleet apply success", func() {
				err := fleetApply([]string{testenv.AssetsPath + "helm"}, &apply.Options{Auth: bundlereader.Auth{Username: username, Password: password}})
				Expect(err).NotTo(HaveOccurred())
			})

			It("then Bundle is created with all the resources inside of the helm release", func() {
				Eventually(verifyResourcesArePresent).Should(BeTrue())
			})
		})
		When("auth is required, it is provided and --helm-repo-url-regex matches the repo url", func() {
			BeforeAll(func() {
				repo.startRepository(true)
			})
			AfterAll(func() {
				err := repo.stopRepository()
				Expect(err).NotTo(HaveOccurred())
				err = env.Fclient.Fleet.Bundle().Delete(testenv.Namespace, bundleName, &metav1.DeleteOptions{})
				Expect(err).NotTo(HaveOccurred())
			})
			It("fleet apply success", func() {
				err := fleetApply([]string{testenv.AssetsPath + "helm"}, &apply.Options{
					Auth:             bundlereader.Auth{Username: username, Password: password},
					HelmRepoUrlRegex: "http://localhost/*",
				})
				Expect(err).NotTo(HaveOccurred())
			})

			It("then Bundle is created with all the resources inside of the helm release", func() {
				Eventually(verifyResourcesArePresent).Should(BeTrue())
			})
		})
		When("auth is required, and it is provided but --helm-repo-url-regex doesn't match", func() {
			BeforeAll(func() {
				repo.startRepository(true)
			})
			AfterAll(func() {
				err := repo.stopRepository()
				Expect(err).NotTo(HaveOccurred())
			})
			It("fleet apply fails when --helm-repo-url-regex doesn't match the helm repo url", func() {
				err := fleetApply([]string{testenv.AssetsPath + "helm"}, &apply.Options{
					Auth:             bundlereader.Auth{Username: username, Password: password},
					HelmRepoUrlRegex: "nomatch",
				})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).Should(Equal("failed to read helm repo from http://localhost:3000/index.yaml, error code: 401, response body: Unauthorised."))
			})
		})
		When("auth is required, and it is provided but --helm-repo-url-regex is not valid", func() {
			BeforeAll(func() {
				repo.startRepository(true)
			})
			AfterAll(func() {
				err := repo.stopRepository()
				Expect(err).NotTo(HaveOccurred())
			})
			It("fleet apply fails when --helm-repo-url-regex is not valid", func() {
				err := fleetApply([]string{testenv.AssetsPath + "helm"}, &apply.Options{
					Auth:             bundlereader.Auth{Username: username, Password: password},
					HelmRepoUrlRegex: "a(b",
				})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).Should(Equal("error parsing regexp: missing closing ): `a(b`"))
			})
		})
	})
})

func verifyResourcesArePresent() bool {
	bundle, err := env.Fclient.Fleet.Bundle().Get(testenv.Namespace, bundleName, metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred())
	paths, err := getAllResourcesPathFromTheHelmRelease()
	Expect(err).NotTo(HaveOccurred())
	Expect(len(paths)).Should(Equal(numberOfFilesInHelmTestRepo))
	// should contain all resources plus the fleet.yaml
	Expect(len(bundle.Spec.Resources)).Should(Equal(numberOfFilesInHelmTestRepo + 1))
	for _, path := range paths {
		Expect(isResourcePresent(path, bundle.Spec.Resources)).Should(BeTrue(), "validate resource "+path)
	}
	return true
}

// returns path for all resources in the assets/helmrepository/testrepo folder
func getAllResourcesPathFromTheHelmRelease() ([]string, error) {
	paths := []string{}
	err := filepath.Walk(testenv.AssetsPath+"helmrepository/testrepo", func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return paths, nil
}

func isResourcePresent(resourcePath string, resources []v1alpha1.BundleResource) bool {
	resourceFile, err := os.ReadFile(resourcePath)
	Expect(err).NotTo(HaveOccurred())

	for _, resource := range resources {
		if strings.ReplaceAll(resource.Content, "\n", "") == strings.ReplaceAll(string(resourceFile), "\n", "") {
			return true
		}
	}

	return false
}
