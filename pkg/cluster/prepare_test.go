package cluster_test

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/weaveworks/pctl/pkg/cluster"
	"github.com/weaveworks/pctl/pkg/runner/fakes"
)

type mockTransport struct {
	res *http.Response
}

func (t *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return t.res, nil
}

var _ = Describe("prepare", func() {
	When("dry run is set", func() {
		It("can prepare the environment with everything that profiles needs without actually modifying the cluster", func() {
			runner := &fakes.FakeRunner{}
			content, err := ioutil.ReadFile(filepath.Join("testdata", "prepare.yaml"))
			Expect(err).NotTo(HaveOccurred())
			client := &http.Client{
				Transport: &mockTransport{
					res: &http.Response{
						StatusCode: http.StatusOK,
						Body:       ioutil.NopCloser(bytes.NewReader(content)),
					},
				},
			}
			tmp, err := ioutil.TempDir("", "prepare_01")
			Expect(err).NotTo(HaveOccurred())
			p := &cluster.Preparer{
				PrepConfig: cluster.PrepConfig{
					Location: tmp,
					DryRun:   true,
				},
				Fetcher: &cluster.Fetcher{
					Client: client,
				},
				Applier: &cluster.Applier{
					Runner: runner,
				},
			}
			err = p.Prepare()
			Expect(err).NotTo(HaveOccurred())
			Expect(runner.RunCallCount()).To(Equal(1))
			arg, args := runner.RunArgsForCall(0)
			Expect(arg).To(Equal("kubectl"))
			Expect(args).To(Equal([]string{"apply", "-f", filepath.Join(tmp, "prepare.yaml"), "--dry-run=client", "--output=yaml"}))
		})
	})
	When("dry-run is not set", func() {
		It("sets up the environment with everything that profiles needs", func() {
			runner := &fakes.FakeRunner{}
			content, err := ioutil.ReadFile(filepath.Join("testdata", "prepare.yaml"))
			Expect(err).NotTo(HaveOccurred())
			client := &http.Client{
				Transport: &mockTransport{
					res: &http.Response{
						StatusCode: http.StatusOK,
						Body:       ioutil.NopCloser(bytes.NewReader(content)),
					},
				},
			}
			tmp, err := ioutil.TempDir("", "prepare_02")
			Expect(err).NotTo(HaveOccurred())
			p := &cluster.Preparer{
				PrepConfig: cluster.PrepConfig{
					BaseURL:  "https://github.com/weaveworks/profiles/releases",
					Location: tmp,
				},
				Fetcher: &cluster.Fetcher{
					Client: client,
				},
				Applier: &cluster.Applier{
					Runner: runner,
				},
			}
			err = p.Prepare()
			Expect(err).NotTo(HaveOccurred())
			Expect(runner.RunCallCount()).To(Equal(1))
			arg, args := runner.RunArgsForCall(0)
			Expect(arg).To(Equal("kubectl"))
			Expect(args).To(Equal([]string{"apply", "-f", filepath.Join(tmp, "prepare.yaml")}))
		})
	})
	When("context and config is provided", func() {
		It("passes that over to kubernetes", func() {
			runner := &fakes.FakeRunner{}
			content, err := ioutil.ReadFile(filepath.Join("testdata", "prepare.yaml"))
			Expect(err).NotTo(HaveOccurred())
			client := &http.Client{
				Transport: &mockTransport{
					res: &http.Response{
						StatusCode: http.StatusOK,
						Body:       ioutil.NopCloser(bytes.NewReader(content)),
					},
				},
			}
			tmp, err := ioutil.TempDir("", "prepare_02")
			Expect(err).NotTo(HaveOccurred())
			p := &cluster.Preparer{
				PrepConfig: cluster.PrepConfig{
					BaseURL:     "https://github.com/weaveworks/profiles/releases",
					Location:    tmp,
					KubeContext: "context",
					KubeConfig:  "kubeconfig",
				},
				Fetcher: &cluster.Fetcher{
					Client: client,
				},
				Applier: &cluster.Applier{
					Runner: runner,
				},
			}
			err = p.Prepare()
			Expect(err).NotTo(HaveOccurred())
			Expect(runner.RunCallCount()).To(Equal(1))
			arg, args := runner.RunArgsForCall(0)
			Expect(arg).To(Equal("kubectl"))
			Expect(args).To(Equal([]string{"apply", "-f", filepath.Join(tmp, "prepare.yaml"), "--context=context", "--kubeconfig=kubeconfig"}))
		})
	})
	When("there is an error running kubectl apply", func() {
		It("will fail and show a proper error to the user", func() {
			runner := &fakes.FakeRunner{}
			runner.RunReturns([]byte("nope"), errors.New("nope"))
			content, err := ioutil.ReadFile(filepath.Join("testdata", "prepare.yaml"))
			Expect(err).NotTo(HaveOccurred())
			client := &http.Client{
				Transport: &mockTransport{
					res: &http.Response{
						StatusCode: http.StatusOK,
						Body:       ioutil.NopCloser(bytes.NewReader(content)),
					},
				},
			}
			p, err := cluster.NewPreparer(cluster.PrepConfig{})
			Expect(err).NotTo(HaveOccurred())
			p.Applier = &cluster.Applier{
				Runner: runner,
			}
			p.Fetcher = &cluster.Fetcher{
				Client: client,
			}
			err = p.Prepare()
			Expect(err).To(MatchError("install failed: nope"))
		})
	})
	When("a specific version is defined", func() {
		It("will respect the version and try to download that", func() {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				Expect(r.URL.String()).To(Equal("/download/v0.0.1/prepare.yaml"))
			}))
			p, err := cluster.NewPreparer(cluster.PrepConfig{
				BaseURL: server.URL,
				Version: "v0.0.1",
			})
			Expect(err).NotTo(HaveOccurred())
			p.Fetcher = &cluster.Fetcher{
				Client: server.Client(),
			}
			// we deliberately ignore the error here. the important part is the called url.
			_ = p.Prepare()
		})
		It("will update the controller's image version so the right version is downloaded", func() {
			runner := &fakes.FakeRunner{}
			content, err := ioutil.ReadFile(filepath.Join("testdata", "prepare.yaml"))
			Expect(err).NotTo(HaveOccurred())
			client := &http.Client{
				Transport: &mockTransport{
					res: &http.Response{
						StatusCode: http.StatusOK,
						Body:       ioutil.NopCloser(bytes.NewReader(content)),
					},
				},
			}
			tmp, err := ioutil.TempDir("", "prepare_change_version_01")
			Expect(err).NotTo(HaveOccurred())
			p := &cluster.Preparer{
				PrepConfig: cluster.PrepConfig{
					Location: tmp,
					DryRun:   true,
					Keep:     true,
					Version:  "v0.0.1",
				},
				Fetcher: &cluster.Fetcher{
					Client: client,
				},
				Applier: &cluster.Applier{
					Runner: runner,
				},
			}
			err = p.Prepare()
			Expect(err).NotTo(HaveOccurred())
			content, err = ioutil.ReadFile(filepath.Join(tmp, "prepare.yaml"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(ContainSubstring("weaveworks/profiles-controller:v0.0.1"))

		})
		It("will only try and fetch versions starting with (v)", func() {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				Expect(r.URL.String()).To(Equal("/latest/download/prepare.yaml"))
			}))
			p, err := cluster.NewPreparer(cluster.PrepConfig{
				BaseURL: server.URL,
				Version: "0.0.1",
			})
			Expect(err).NotTo(HaveOccurred())
			p.Fetcher = &cluster.Fetcher{
				Client: server.Client(),
			}
			// we deliberately ignore the error here. the important part is the called url.
			_ = p.Prepare()
		})
	})
	When("there is an error accessing the given URL", func() {
		It("will provide a sensible failure", func() {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadGateway)
			}))
			p, err := cluster.NewPreparer(cluster.PrepConfig{
				BaseURL: server.URL,
			})
			Expect(err).NotTo(HaveOccurred())
			p.Fetcher = &cluster.Fetcher{
				Client: server.Client(),
			}
			err = p.Prepare()
			msg := fmt.Sprintf("failed to download prepare.yaml from %s/latest/download/prepare.yaml, status: 502 Bad Gateway", server.URL)
			Expect(err).To(MatchError(msg))
		})
	})
	When("the base url is invalid", func() {
		It("will provide a sensible failure", func() {
			p, err := cluster.NewPreparer(cluster.PrepConfig{
				BaseURL: "invalid",
			})
			Expect(err).NotTo(HaveOccurred())
			err = p.Prepare()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unsupported protocol scheme"))
		})
	})
	When("the user decided to keep the downloaded file(s)", func() {
		It("will not delete the downloaded file(s)", func() {
			runner := &fakes.FakeRunner{}
			content, err := ioutil.ReadFile(filepath.Join("testdata", "prepare.yaml"))
			Expect(err).NotTo(HaveOccurred())
			client := &http.Client{
				Transport: &mockTransport{
					res: &http.Response{
						StatusCode: http.StatusOK,
						Body:       ioutil.NopCloser(bytes.NewReader(content)),
					},
				},
			}
			tmp, err := ioutil.TempDir("", "prepare_should_keep_things")
			Expect(err).NotTo(HaveOccurred())
			p := &cluster.Preparer{
				PrepConfig: cluster.PrepConfig{
					BaseURL:  "https://github.com/weaveworks/profiles/releases",
					Location: tmp,
					Keep:     true,
				},
				Fetcher: &cluster.Fetcher{
					Client: client,
				},
				Applier: &cluster.Applier{
					Runner: runner,
				},
			}
			err = p.Prepare()
			Expect(err).NotTo(HaveOccurred())
			downloadedContent, err := ioutil.ReadFile(filepath.Join(tmp, "prepare.yaml"))
			Expect(err).NotTo(HaveOccurred())
			Expect(downloadedContent).To(Equal(content))
		})
	})
	When("when all is done", func() {
		It("should remove any temporary folders", func() {
			runner := &fakes.FakeRunner{}
			content, err := ioutil.ReadFile(filepath.Join("testdata", "prepare.yaml"))
			Expect(err).NotTo(HaveOccurred())
			client := &http.Client{
				Transport: &mockTransport{
					res: &http.Response{
						StatusCode: http.StatusOK,
						Body:       ioutil.NopCloser(bytes.NewReader(content)),
					},
				},
			}
			tmp, err := ioutil.TempDir("", "prepare_should_be_deleted_01")
			Expect(err).NotTo(HaveOccurred())
			p := &cluster.Preparer{
				PrepConfig: cluster.PrepConfig{
					BaseURL:  "https://github.com/weaveworks/profiles/releases",
					Location: tmp,
				},
				Fetcher: &cluster.Fetcher{
					Client: client,
				},
				Applier: &cluster.Applier{
					Runner: runner,
				},
			}
			err = p.Prepare()
			Expect(err).NotTo(HaveOccurred())
			_, err = os.Stat(tmp)
			Expect(os.IsNotExist(err)).To(BeTrue())
		})
	})
})
