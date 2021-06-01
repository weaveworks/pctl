package repo_test

import (
	"bytes"
	"errors"
	"io/ioutil"
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/weaveworks/pctl/pkg/repo"
	"github.com/weaveworks/pctl/pkg/repo/fakes"
	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"
)

var _ = Describe("Repo", func() {
	var (
		fakeHTTPClient  *fakes.FakeHTTPClient
		repoURL, branch string
	)

	BeforeEach(func() {
		fakeHTTPClient = new(fakes.FakeHTTPClient)
		repo.SetHTTPClient(fakeHTTPClient)

		repoURL = "github.com/foo/bar"
		branch = "main"
	})

	It("returns the profile definition", func() {
		httpBody := bytes.NewBufferString(`
apiVersion: profiles.fluxcd.io/v1alpha1
kind: Profile
metadata:
  name: nginx
spec:
  description: foo
  artifacts:
    - name: bar
      path: baz`)
		fakeHTTPClient.GetReturns(&http.Response{
			Body:       ioutil.NopCloser(httpBody),
			StatusCode: http.StatusOK,
		}, nil)

		definition, err := repo.GetProfileDefinition(repoURL, branch, "my-profile")
		Expect(err).NotTo(HaveOccurred())
		Expect(fakeHTTPClient.GetCallCount()).To(Equal(1))
		Expect(fakeHTTPClient.GetArgsForCall(0)).To(Equal("raw.githubusercontent.com/foo/bar/main/my-profile/profile.yaml"))
		Expect(definition).To(Equal(profilesv1.ProfileDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Name: "nginx",
			},
			TypeMeta: metav1.TypeMeta{
				Kind:       "Profile",
				APIVersion: "profiles.fluxcd.io/v1alpha1",
			},
			Spec: profilesv1.ProfileDefinitionSpec{
				Description: "foo",
				Artifacts: []profilesv1.Artifact{
					{
						Name: "bar",
						Path: "baz",
					},
				},
			},
		}))
	})

	When("the get request fails", func() {
		It("returns an error", func() {
			fakeHTTPClient.GetReturns(nil, errors.New("errord"))
			_, err := repo.GetProfileDefinition(repoURL, branch, "my-profile")
			Expect(err).To(MatchError("failed to fetch profile: errord"))
			Expect(fakeHTTPClient.GetCallCount()).To(Equal(1))
			Expect(fakeHTTPClient.GetArgsForCall(0)).To(Equal("raw.githubusercontent.com/foo/bar/main/my-profile/profile.yaml"))
		})
	})

	When("the return code is not 200", func() {
		It("returns an error", func() {
			fakeHTTPClient.GetReturns(&http.Response{StatusCode: 404, Body: ioutil.NopCloser(bytes.NewReader(nil))}, nil)
			_, err := repo.GetProfileDefinition(repoURL, branch, "my-profile")
			Expect(err).To(MatchError("failed to fetch profile: status code 404"))
			Expect(fakeHTTPClient.GetCallCount()).To(Equal(1))
			Expect(fakeHTTPClient.GetArgsForCall(0)).To(Equal("raw.githubusercontent.com/foo/bar/main/my-profile/profile.yaml"))

		})
	})

	When("the profile.yaml is not valid yaml", func() {
		It("returns an error", func() {
			httpBody := bytes.NewBufferString("{not valid yaml")
			fakeHTTPClient.GetReturns(&http.Response{
				Body:       ioutil.NopCloser(httpBody),
				StatusCode: http.StatusOK,
			}, nil)

			_, err := repo.GetProfileDefinition(repoURL, branch, "my-profile")
			Expect(err).To(MatchError(ContainSubstring("failed to parse profile")))
			Expect(fakeHTTPClient.GetCallCount()).To(Equal(1))
			Expect(fakeHTTPClient.GetArgsForCall(0)).To(Equal("raw.githubusercontent.com/foo/bar/main/my-profile/profile.yaml"))

		})
	})

	When("the profile URL is not a URL", func() {
		It("returns an error", func() {
			repoURL = "{}\"!@£!@$:!@£!@"
			_, err := repo.GetProfileDefinition(repoURL, branch, "my-profile")
			Expect(err).To(MatchError(ContainSubstring("failed to parse repo URL")))
			Expect(fakeHTTPClient.GetCallCount()).To(Equal(0))
		})
	})

	When("the profile URL is not github.com", func() {
		It("returns an error", func() {
			repoURL = "gitlab.com/foo/bar"
			_, err := repo.GetProfileDefinition(repoURL, branch, "my-profile")
			Expect(err).To(MatchError("unsupported git provider, only github.com is currently supported"))
			Expect(fakeHTTPClient.GetCallCount()).To(Equal(0))
		})
	})

	// TODO test for when the profile.yaml file is empty
})
