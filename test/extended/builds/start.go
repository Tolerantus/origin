package builds

import (
	"fmt"
	"sync"
	"time"

	"k8s.io/kubernetes/pkg/util/wait"

	g "github.com/onsi/ginkgo"
	o "github.com/onsi/gomega"

	exutil "github.com/openshift/origin/test/extended/util"
)

var _ = g.Describe("[builds][Slow] starting a build using CLI", func() {
	defer g.GinkgoRecover()
	var (
		buildFixture   = exutil.FixturePath("testdata", "test-build.json")
		exampleGemfile = exutil.FixturePath("testdata", "test-build-app", "Gemfile")
		exampleBuild   = exutil.FixturePath("testdata", "test-build-app")
		oc             = exutil.NewCLI("cli-start-build", exutil.KubeConfigPath())
	)

	g.JustBeforeEach(func() {
		g.By("waiting for builder service account")
		err := exutil.WaitForBuilderAccount(oc.KubeREST().ServiceAccounts(oc.Namespace()))
		o.Expect(err).NotTo(o.HaveOccurred())
		oc.Run("create").Args("-f", buildFixture).Execute()
	})

	g.Describe("oc start-build --wait", func() {
		g.It("should start a build and wait for the build to complete", func() {
			g.By("starting the build with --wait flag")
			out, err := oc.Run("start-build").Args("sample-build", "--wait").Output()
			fmt.Fprintf(g.GinkgoWriter, "\nstart-build output:\n%s\n", out)
			o.Expect(err).NotTo(o.HaveOccurred())

			g.By(fmt.Sprintf("verifying the build %q status", out))
			err = exutil.WaitForABuild(oc.REST().Builds(oc.Namespace()), out, exutil.CheckBuildSuccessFn, exutil.CheckBuildFailedFn)
			if err != nil {
				exutil.DumpBuildLogs("sample-build", oc)
			}
			o.Expect(err).NotTo(o.HaveOccurred())
		})

		g.It("should start a build and wait for the build to fail", func() {
			g.By("starting the build with --wait flag but wrong --commit")
			out, err := oc.Run("start-build").
				Args("sample-build", "--wait", "--commit", "fffffff").
				Output()
			fmt.Fprintf(g.GinkgoWriter, "\nstart-build output:\n%s\n", out)
			o.Expect(err).To(o.HaveOccurred())
			o.Expect(out).Should(o.ContainSubstring(`status is "Failed"`))
		})
	})

	g.Describe("override environment", func() {
		g.It("should accept environment variables", func() {
			g.By("starting the build with -e FOO=bar,VAR=test")
			out, err := oc.Run("start-build").Args("sample-build", "-e", "FOO=bar,VAR=test").Output()
			fmt.Fprintf(g.GinkgoWriter, "\nstart-build output:\n%s\n", out)
			o.Expect(err).NotTo(o.HaveOccurred())

			g.By("waiting for the build to complete")
			err = exutil.WaitForABuild(oc.REST().Builds(oc.Namespace()), "sample-build-1", exutil.CheckBuildSuccessFn, exutil.CheckBuildFailedFn)
			if err != nil {
				exutil.DumpBuildLogs("sample-build", oc)
			}
			o.Expect(err).NotTo(o.HaveOccurred())

			g.By("verifying the build output")
			out, err = oc.Run("logs").Args("-f", "bc/sample-build").Output()
			fmt.Fprintf(g.GinkgoWriter, "\n: build logs: %s\n", out)
			o.Expect(err).NotTo(o.HaveOccurred())

			g.By(fmt.Sprintf("verifying the build output contains the env vars"))
			o.Expect(out).To(o.ContainSubstring("FOO=bar"))
			o.Expect(out).To(o.ContainSubstring("VAR=test"))

			g.By(fmt.Sprintf("verifying the build output contains inherited env vars"))
			// This variable is not set and thus inherited from the original build config
			o.Expect(out).To(o.ContainSubstring("BAR=test"))

			g.By(fmt.Sprintf("verifying the build %q status", out))
			err = exutil.WaitForABuild(oc.REST().Builds(oc.Namespace()), "sample-build-1", exutil.CheckBuildSuccessFn, exutil.CheckBuildFailedFn)
			if err != nil {
				exutil.DumpBuildLogs("sample-build", oc)
			}
			o.Expect(err).NotTo(o.HaveOccurred())
		})

		g.It("should allow to change build log level", func() {
			g.By("starting the build with --build-loglevel=1")
			out, err := oc.Run("start-build").Args("sample-build", "--build-loglevel=1").Output()
			fmt.Fprintf(g.GinkgoWriter, "\nstart-build output:\n%s\n", out)
			o.Expect(err).NotTo(o.HaveOccurred())

			g.By("waiting for the build to complete")
			err = exutil.WaitForABuild(oc.REST().Builds(oc.Namespace()), "sample-build-1", exutil.CheckBuildSuccessFn, exutil.CheckBuildFailedFn)
			if err != nil {
				exutil.DumpBuildLogs("sample-build", oc)
			}

			g.By("verifying the build output")
			buildLog, err := oc.Run("logs").Args("-f", "bc/sample-build").Output()
			fmt.Fprintf(g.GinkgoWriter, "\nbuild log:\n%s\n", buildLog)
			o.Expect(err).NotTo(o.HaveOccurred())

			g.By(fmt.Sprintf("verifying the build output is not verbose"))
			o.Expect(buildLog).NotTo(o.ContainSubstring("Creating a new S2I builder"))
		})
	})

	g.Describe("binary builds", func() {
		g.It("should accept --from-file as input", func() {
			g.By("starting the build with a Dockerfile")
			out, err := oc.Run("start-build").Args("sample-build", fmt.Sprintf("--from-file=%s", exampleGemfile)).Output()
			fmt.Fprintf(g.GinkgoWriter, "\nstart-build output:\n%s\n", out)
			o.Expect(err).NotTo(o.HaveOccurred())

			err = exutil.WaitForABuild(oc.REST().Builds(oc.Namespace()), "sample-build-1", exutil.CheckBuildSuccessFn, exutil.CheckBuildFailedFn)
			if err != nil {
				exutil.DumpBuildLogs("sample-build", oc)
			}
			o.Expect(err).NotTo(o.HaveOccurred())

			g.By(fmt.Sprintf("verifying the build %q status", out))
			buildLog, err := oc.Run("logs").Args("-f", "bc/sample-build").Output()
			fmt.Fprintf(g.GinkgoWriter, "\nbuild log:\n%s\n", buildLog)
			o.Expect(err).NotTo(o.HaveOccurred())

			o.Expect(out).To(o.ContainSubstring("Uploading file"))
			o.Expect(out).To(o.ContainSubstring("as binary input for the build ..."))
			o.Expect(buildLog).To(o.ContainSubstring("Your bundle is complete"))

		})

		g.It("should accept --from-dir as input", func() {
			g.By("starting the build with a directory")
			out, err := oc.Run("start-build").Args("sample-build", fmt.Sprintf("--from-dir=%s", exampleBuild)).Output()
			fmt.Fprintf(g.GinkgoWriter, "\nstart-build output:\n%s\n", out)
			o.Expect(err).NotTo(o.HaveOccurred())

			g.By("waiting for the build to complete")
			err = exutil.WaitForABuild(oc.REST().Builds(oc.Namespace()), "sample-build-1", exutil.CheckBuildSuccessFn, exutil.CheckBuildFailedFn)
			if err != nil {
				exutil.DumpBuildLogs("sample-build", oc)
			}
			o.Expect(err).NotTo(o.HaveOccurred())

			g.By(fmt.Sprintf("verifying the build %q output", out))
			buildLog, err := oc.Run("logs").Args("-f", "bc/sample-build").Output()
			fmt.Fprintf(g.GinkgoWriter, "\nbuild logs:\n%s\n", buildLog)
			o.Expect(err).NotTo(o.HaveOccurred())

			o.Expect(out).To(o.ContainSubstring("Uploading directory"))
			o.Expect(out).To(o.ContainSubstring("as binary input for the build ..."))
			o.Expect(buildLog).To(o.ContainSubstring("Your bundle is complete"))
		})

		g.It("should accept --from-repo as input", func() {
			g.By("starting the build with a Git repository")
			out, err := oc.Run("start-build").Args("sample-build", fmt.Sprintf("--from-repo=%s", exampleBuild)).Output()
			fmt.Fprintf(g.GinkgoWriter, "\nstart-build output:\n%s\n", out)
			o.Expect(err).NotTo(o.HaveOccurred())

			g.By("waiting for the build to complete")
			err = exutil.WaitForABuild(oc.REST().Builds(oc.Namespace()), "sample-build-1", exutil.CheckBuildSuccessFn, exutil.CheckBuildFailedFn)
			if err != nil {
				exutil.DumpBuildLogs("sample-build", oc)
			}
			o.Expect(err).NotTo(o.HaveOccurred())

			g.By(fmt.Sprintf("verifying the build %q output", out))
			buildLog, err := oc.Run("logs").Args("-f", "bc/sample-build").Output()
			fmt.Fprintf(g.GinkgoWriter, "\nbuild log:\n%s\n", buildLog)
			o.Expect(err).NotTo(o.HaveOccurred())

			o.Expect(out).To(o.ContainSubstring("Uploading"))
			o.Expect(out).To(o.ContainSubstring(`at commit "HEAD"`))
			o.Expect(out).To(o.ContainSubstring("as binary input for the build ..."))
			o.Expect(buildLog).To(o.ContainSubstring("Your bundle is complete"))
		})

		g.It("should accept --from-repo with --commit as input", func() {
			g.By("starting the build with a Git repository")
			// NOTE: This actually takes the commit from the origin repository. If the
			// test-build-app changes, this commit has to be bumped.
			out, err := oc.Run("start-build").Args("sample-build", "--commit=f0f3834", fmt.Sprintf("--from-repo=%s", exampleBuild)).Output()
			fmt.Fprintf(g.GinkgoWriter, "\nstart-build output:\n%s\n", out)
			o.Expect(err).NotTo(o.HaveOccurred())

			g.By("waiting for the build to complete")
			err = exutil.WaitForABuild(oc.REST().Builds(oc.Namespace()), "sample-build-1", exutil.CheckBuildSuccessFn, exutil.CheckBuildFailedFn)
			if err != nil {
				exutil.DumpBuildLogs("sample-build", oc)
			}
			o.Expect(err).NotTo(o.HaveOccurred())

			g.By(fmt.Sprintf("verifying the build %q output", out))
			buildLog, err := oc.Run("logs").Args("-f", "bc/sample-build").Output()
			fmt.Fprintf(g.GinkgoWriter, "\nbuild log:\n%s\n", buildLog)
			o.Expect(err).NotTo(o.HaveOccurred())

			o.Expect(out).To(o.ContainSubstring("Uploading"))
			o.Expect(out).To(o.ContainSubstring(`at commit "f0f3834"`))
			o.Expect(buildLog).To(o.ContainSubstring(`"commit":"f0f38342e53eac2a6995acca81d06bd9dd6d4964"`))
			o.Expect(out).To(o.ContainSubstring("as binary input for the build ..."))
			o.Expect(buildLog).To(o.ContainSubstring("Your bundle is complete"))
		})
	})

	g.Describe("cancelling build started by oc start-build --wait", func() {
		g.It("should start a build and wait for the build to cancel", func() {
			g.By("starting the build with --wait flag")
			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				defer g.GinkgoRecover()
				out, err := oc.Run("start-build").Args("sample-build", "--wait").Output()
				fmt.Fprintf(g.GinkgoWriter, "\ngo routine start-build output:\n%s\n", out)
				defer wg.Done()
				o.Expect(err).To(o.HaveOccurred())
				o.Expect(out).Should(o.ContainSubstring(`status is "Cancelled"`))
			}()

			g.By("getting the build name")
			var buildName string
			wait.Poll(time.Duration(100*time.Millisecond), 1*time.Minute, func() (bool, error) {
				out, err := oc.Run("get").
					Args("build", "--template", "{{ (index .items 0).metadata.name }}").Output()
				// Give it second chance in case the build resource was not created yet
				if err != nil || len(out) == 0 {
					return false, nil
				}
				buildName = out
				return true, nil
			})

			o.Expect(buildName).ToNot(o.BeEmpty())

			g.By(fmt.Sprintf("cancelling the build %q", buildName))
			err := oc.Run("cancel-build").Args(buildName).Execute()
			o.Expect(err).ToNot(o.HaveOccurred())
			wg.Wait()
		})

	})

})
