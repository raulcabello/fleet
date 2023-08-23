package apply

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"strings"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"
	httpgit "github.com/go-git/go-git/v5/plumbing/transport/http"
	gossh "github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/gogits/go-gogs-client"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	cp "github.com/otiai10/copy"
	"github.com/rancher/fleet/integrationtests/cli"
	"github.com/rancher/fleet/internal/cmd/cli/apply"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"golang.org/x/crypto/ssh"
)

const (
	gogsUser = "test"
	gogsPass = "pass"
)

var (
	gogsClient *gogs.Client
)

var _ = Describe("Fleet apply gets content from git repo", Ordered, func() {

	var (
		name    string
		options apply.Options
	)

	When("Public repo that contains simple resources", func() {
		BeforeEach(func() {
			name = "simple"
			_, url, err := createGogsContainer()
			Expect(err).NotTo(HaveOccurred())
			err = createRepo(url, cli.AssetsPath+name, false)
			Expect(err).NotTo(HaveOccurred())
			options = apply.Options{
				Output:  gbytes.NewBuffer(),
				GitRepo: url + "/test/test-repo",
			}
			err = fleetApply(name, []string{}, options)
			Expect(err).NotTo(HaveOccurred())
		})

		It("Bundle is created with all the resources", func() {
			Eventually(func() bool {
				bundle, err := cli.GetBundleFromOutput(options.Output)
				Expect(err).NotTo(HaveOccurred())
				Expect(len(bundle.Spec.Resources)).To(Equal(2))
				isSvcPresent, err := cli.IsResourcePresentInBundle(cli.AssetsPath+"simple/svc.yaml", bundle.Spec.Resources)
				Expect(err).NotTo(HaveOccurred())
				isDeploymentPresent, err := cli.IsResourcePresentInBundle(cli.AssetsPath+"simple/deployment.yaml", bundle.Spec.Resources)
				Expect(err).NotTo(HaveOccurred())

				return isSvcPresent && isDeploymentPresent
			}).Should(BeTrue())
		})
	})

	When("Private repo that contains simple resources in a nested folder", func() {
		When("Basic authentication is provided", func() {
			BeforeEach(func() {
				name = "nested_simple"
				_, url, err := createGogsContainer()
				Expect(err).NotTo(HaveOccurred())
				err = createRepo(url, cli.AssetsPath+name, true)
				Expect(err).NotTo(HaveOccurred())
				options = apply.Options{
					Output:  gbytes.NewBuffer(),
					GitRepo: url + "/test/test-repo",
					GitAuth: &httpgit.BasicAuth{
						Username: gogsUser,
						Password: gogsPass,
					},
				}
				err = fleetApply(name, []string{}, options)
				Expect(err).NotTo(HaveOccurred())
			})

			It("Bundle is created with all the resources", func() {
				Eventually(func() bool {
					bundle, err := cli.GetBundleFromOutput(options.Output)
					Expect(err).NotTo(HaveOccurred())
					Expect(len(bundle.Spec.Resources)).To(Equal(3))
					isSvcPresent, err := cli.IsResourcePresentInBundle(cli.AssetsPath+"simple/svc.yaml", bundle.Spec.Resources)
					Expect(err).NotTo(HaveOccurred())
					isDeploymentPresent, err := cli.IsResourcePresentInBundle(cli.AssetsPath+"simple/deployment.yaml", bundle.Spec.Resources)
					Expect(err).NotTo(HaveOccurred())

					return isSvcPresent && isDeploymentPresent
				}).Should(BeTrue())
			})
		})

		FWhen("ssh authentication is provided", func() {
			BeforeEach(func() {
				name = "nested_simple"
				container, url, err := createGogsContainer()
				Expect(err).NotTo(HaveOccurred())
				privateKey, err := createAndAddKeys()
				Expect(err).NotTo(HaveOccurred())
				err = createRepo(url, cli.AssetsPath+name, true)
				Expect(err).NotTo(HaveOccurred())
				auth, err := gossh.NewPublicKeys("git", []byte(privateKey), "")
				Expect(err).NotTo(HaveOccurred())
				auth.HostKeyCallback = ssh.InsecureIgnoreHostKey()
				sshPort, err := container.MappedPort(context.Background(), "22")
				Expect(err).NotTo(HaveOccurred())

				options = apply.Options{
					Output:  gbytes.NewBuffer(),
					GitRepo: "ssh://git@localhost:" + sshPort.Port() + "/test/test-repo",
					GitAuth: auth,
				}
				Eventually(func() error {
					return fleetApply(name, []string{}, options)
				}).Should(Not(HaveOccurred()))
			})

			It("Bundle is created with all the resources", func() {
				Eventually(func() bool {
					bundle, err := cli.GetBundleFromOutput(options.Output)
					Expect(err).NotTo(HaveOccurred())
					Expect(len(bundle.Spec.Resources)).To(Equal(3))
					isSvcPresent, err := cli.IsResourcePresentInBundle(cli.AssetsPath+"simple/svc.yaml", bundle.Spec.Resources)
					Expect(err).NotTo(HaveOccurred())
					isDeploymentPresent, err := cli.IsResourcePresentInBundle(cli.AssetsPath+"simple/deployment.yaml", bundle.Spec.Resources)
					Expect(err).NotTo(HaveOccurred())

					return isSvcPresent && isDeploymentPresent
				}).Should(BeTrue())
			})
		})

		When("Authentication is not provided", func() {
			It("Error is returned", func() {
				name = "nested_simple"
				_, url, err := createGogsContainer()
				Expect(err).NotTo(HaveOccurred())
				err = createRepo(url, cli.AssetsPath+name, true)
				Expect(err).NotTo(HaveOccurred())
				options = apply.Options{
					Output:  gbytes.NewBuffer(),
					GitRepo: url + "/test/test-repo",
				}
				err = fleetApply(name, []string{}, options)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("error cloning repo: authentication required"))
			})
		})
	})
})

func createGogsContainer() (testcontainers.Container, string, error) {
	tmpDir := GinkgoT().TempDir()
	err := cp.Copy("../assets/gitserver", tmpDir)
	if err != nil {
		return nil, "", err
	}
	req := testcontainers.ContainerRequest{
		Image:        "gogs/gogs:0.13",
		ExposedPorts: []string{"3000/tcp", "22/tcp"},
		WaitingFor:   wait.ForHTTP("/").WithPort("3000/tcp"),
		Mounts: testcontainers.ContainerMounts{
			{
				Source: testcontainers.GenericBindMountSource{HostPath: tmpDir},
				Target: "/data",
			},
		},
	}
	container, err := testcontainers.GenericContainer(context.Background(), testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})

	if err != nil {
		return nil, "", err
	}

	url, err := getURL(context.Background(), container)
	if err != nil {
		return nil, "", err
	}

	c := gogs.NewClient(url, "")
	token, err := c.CreateAccessToken(gogsUser, gogsPass, gogs.CreateAccessTokenOption{
		Name: "test",
	})
	if err != nil {
		return nil, "", err
	}

	gogsClient = gogs.NewClient(url, token.Sha1)

	return container, url, nil
}

// createRepo creates a git repo for testing
func createRepo(url string, path string, private bool) error {
	name := "test-repo"
	_, err := gogsClient.CreateRepo(gogs.CreateRepoOption{
		Name:    name,
		Private: private,
	})
	if err != nil {
		return err
	}
	repoURL := url + "/" + gogsUser + "/" + name

	// add initial commit
	tmp, err := os.MkdirTemp("", name)
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	_, err = gogit.PlainInit(tmp, false)
	if err != nil {
		return err
	}
	r, err := gogit.PlainOpen(tmp)
	if err != nil {
		return err
	}
	w, err := r.Worktree()
	if err != nil {
		return err
	}
	err = cp.Copy(path, tmp)
	if err != nil {
		return err
	}
	_, err = w.Add(".")
	if err != nil {
		return err
	}
	_, err = w.Commit("test commit", &gogit.CommitOptions{
		Author: &object.Signature{
			Name:  "Test user",
			Email: "test@test.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		return err
	}
	cfg, err := r.Config()
	if err != nil {
		return err
	}
	cfg.Remotes["upstream"] = &config.RemoteConfig{
		Name: "upstream",
		URLs: []string{repoURL},
	}
	err = r.SetConfig(cfg)
	if err != nil {
		return err
	}
	err = r.Push(&gogit.PushOptions{
		RemoteName: "upstream",
		RemoteURL:  repoURL,
		Auth: &httpgit.BasicAuth{
			Username: gogsUser,
			Password: gogsPass,
		},
	})
	if err != nil {
		return err
	}

	return nil
}

func getURL(ctx context.Context, container testcontainers.Container) (string, error) {
	mappedPort, err := container.MappedPort(ctx, "3000")
	if err != nil {
		return "", err
	}
	host, err := container.Host(ctx)
	if err != nil {
		return "", err
	}
	url := "http://" + host + ":" + mappedPort.Port()

	return url, nil
}

// createAndAddKeys creates a public private key pair. It adds the public key to gogs, and returns the private key.
func createAndAddKeys() (string, error) {
	publicKey, privateKey, err := makeSSHKeyPair()
	if err != nil {
		return "", err
	}

	_, err = gogsClient.CreatePublicKey(gogs.CreateKeyOption{
		Title: "test",
		Key:   publicKey,
	})
	if err != nil {
		return "", err
	}

	return privateKey, nil
}

func makeSSHKeyPair() (string, string, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return "", "", err
	}

	var privKeyBuf strings.Builder
	privateKeyPEM := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)}
	if err := pem.Encode(&privKeyBuf, privateKeyPEM); err != nil {
		return "", "", err
	}
	pub, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return "", "", err
	}
	var pubKeyBuf strings.Builder
	pubKeyBuf.Write(ssh.MarshalAuthorizedKey(pub))

	return pubKeyBuf.String(), privKeyBuf.String(), nil
}
