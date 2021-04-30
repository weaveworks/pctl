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
			content, err := ioutil.ReadFile(filepath.Join("testdata", "manifests.tar.gz"))
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
			Expect(args).To(Equal([]string{"apply", "-f", tmp, "--dry-run=client", "--output=yaml"}))
		})
	})
	When("dry-run is not set", func() {
		It("sets up the environment with everything that profiles needs", func() {
			runner := &fakes.FakeRunner{}
			content, err := ioutil.ReadFile(filepath.Join("testdata", "manifests.tar.gz"))
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
			Expect(args).To(Equal([]string{"apply", "-f", tmp}))
		})
	})
	When("context and config is provided", func() {
		It("passes that over to kubernetes", func() {
			runner := &fakes.FakeRunner{}
			content, err := ioutil.ReadFile(filepath.Join("testdata", "manifests.tar.gz"))
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
			Expect(args).To(Equal([]string{"apply", "-f", tmp, "--context=context", "--kubeconfig=kubeconfig"}))
		})
	})
	When("there is an error running kubectl apply", func() {
		It("will fail and show a proper error to the user", func() {
			runner := &fakes.FakeRunner{}
			runner.RunReturns([]byte("nope"), errors.New("nope"))
			content, err := ioutil.ReadFile(filepath.Join("testdata", "manifests.tar.gz"))
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
	When("there is an invalid tar file being downloaded", func() {
		It("fails to untar it", func() {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-type", "application/gzip")
				_, _ = w.Write([]byte("invalid-tar"))
			}))
			p, err := cluster.NewPreparer(cluster.PrepConfig{
				BaseURL: server.URL,
			})
			Expect(err).NotTo(HaveOccurred())
			p.Fetcher = &cluster.Fetcher{
				Client: server.Client(),
			}
			err = p.Prepare()
			Expect(err).To(HaveOccurred())
			fmt.Println("error: ", err)
			msg := fmt.Sprintf("failed to untar manifests.tar.gz from %s/latest/download/manifests.tar.gz, error: requires gzip-compressed body: gzip: invalid header", server.URL)
			Expect(err.Error()).To(Equal(msg))
		})
	})
	When("a specific version is defined", func() {
		It("the fetcher will respect the version and try to download that", func() {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				Expect(r.URL.String()).To(Equal("/download/v0.0.1/manifests.tar.gz"))
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
		It("will only try and fetch versions starting with (v)", func() {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				Expect(r.URL.String()).To(Equal("/latest/download/manifests.tar.gz"))
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
			msg := fmt.Sprintf("failed to download manifests.tar.gz from %s/latest/download/manifests.tar.gz, status: 502 Bad Gateway", server.URL)
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
	When("when all is done", func() {
		It("should remove any temporary folders", func() {
			runner := &fakes.FakeRunner{}
			content, err := ioutil.ReadFile(filepath.Join("testdata", "manifests.tar.gz"))
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
