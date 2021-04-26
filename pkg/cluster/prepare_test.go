package cluster_test

import (
	"bytes"
	"errors"
	"io/ioutil"
	"net/http"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/weaveworks/pctl/pkg/cluster"

	gitfakes "github.com/weaveworks/pctl/pkg/git/fakes"
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
			runner := &gitfakes.FakeRunner{}
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
			files, err := ioutil.ReadDir(tmp)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(files) > 0).To(BeTrue())
			Expect(runner.RunCallCount()).To(Equal(1)) // there should be no call because of dryrun.
			arg, args := runner.RunArgsForCall(0)
			Expect(arg).To(Equal("kubectl"))
			Expect(args).To(Equal([]string{"apply", "-f", tmp, "--dry-run=client"}))
		})
	})
	When("dry-run is not set", func() {
		It("sets up the environment with everything that profiles needs", func() {
			runner := &gitfakes.FakeRunner{}
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
			files, err := ioutil.ReadDir(tmp)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(files) > 0).To(BeTrue())
			Expect(runner.RunCallCount()).To(Equal(1))
			arg, args := runner.RunArgsForCall(0)
			Expect(arg).To(Equal("kubectl"))
			Expect(args).To(Equal([]string{"apply", "-f", tmp}))
		})
	})
	When("there is an error downloading the manifests", func() {
		It("will fail and show a proper error to the user", func() {
			runner := &gitfakes.FakeRunner{}
			client := &http.Client{
				Transport: &mockTransport{
					res: &http.Response{
						StatusCode: http.StatusNotFound,
						Status:     http.StatusText(http.StatusNotFound),
					},
				},
			}
			tmp, err := ioutil.TempDir("", "prepare_03")
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
			Expect(err).To(MatchError("failed to download manifests.tar.gz from https://github.com/weaveworks/profiles/releases/latest/download/manifests.tar.gz, status: Not Found"))
		})
	})
	When("there is an error accessing the output folder", func() {
		It("will fail and show a proper error to the user", func() {
			runner := &gitfakes.FakeRunner{}
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
			p := &cluster.Preparer{
				PrepConfig: cluster.PrepConfig{
					BaseURL:  "https://github.com/weaveworks/profiles/releases",
					Location: "/inaccessible",
				},
				Fetcher: &cluster.Fetcher{
					Client: client,
				},
				Applier: &cluster.Applier{
					Runner: runner,
				},
			}
			err = p.Prepare()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to untar manifests.tar.gz from https://github.com/weaveworks/profiles/releases/latest/download/manifests.tar.gz, error: mkdir /inaccessible:"))
		})
	})
	When("there is an error running kubectl apply", func() {
		It("will fail and show a proper error to the user", func() {
			runner := &gitfakes.FakeRunner{}
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
			tmp, err := ioutil.TempDir("", "prepare_04")
			Expect(err).NotTo(HaveOccurred())
			p := &cluster.Preparer{
				PrepConfig: cluster.PrepConfig{
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
			Expect(err).To(MatchError("install failed: nope"))
		})
	})
	When("there is an invalid tar file being downloaded", func() {
		It("fails to untar it", func() {
			runner := &gitfakes.FakeRunner{}
			client := &http.Client{
				Transport: &mockTransport{
					res: &http.Response{
						StatusCode: http.StatusOK,
						Body:       ioutil.NopCloser(bytes.NewReader([]byte("not-tar"))),
					},
				},
			}
			p := &cluster.Preparer{
				PrepConfig: cluster.PrepConfig{
					BaseURL:  "https://github.com/weaveworks/profiles/releases",
					Location: "/inaccessible",
				},
				Fetcher: &cluster.Fetcher{
					Client: client,
				},
				Applier: &cluster.Applier{
					Runner: runner,
				},
			}
			err := p.Prepare()
			Expect(err).To(MatchError("failed to untar manifests.tar.gz from https://github.com/weaveworks/profiles/releases/latest/download/manifests.tar.gz, error: requires gzip-compressed body: unexpected EOF"))
		})
	})
})
