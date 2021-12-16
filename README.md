# Archived. Profiles work has moved to https://github.com/weaveworks/weave-gitops

# pctl
pctl is a cli tool for interacting with [Profiles](https://github.com/weaveworks/profiles)

<!--
To update the TOC, install https://github.com/kubernetes-sigs/mdtoc
and run: mdtoc -inplace README.md
-->

 <!-- toc -->
- [Usage](#usage)
- [Contributing](#contributing)
  - [Working with profiles](#working-with-profiles)
    - [Using local pin](#using-local-pin)
- [Release process](#release-process)
  - [Tests](#tests)
    - [Configuring Integration Tests](#configuring-integration-tests)
<!-- /toc -->

## Usage

For operational documentation please visit the [Profiles Documentation](https://profiles.dev/docs).

## Contributing

In order to run CLI commands you need a profiles catalog controller up and running along with its API in a cluster.
To get a local setup clone the [Profiles repo](https://github.com/weaveworks/profiles) and run `make local-env`.
This will deploy a local kind cluster with the catalog controller and API running. Once the environment is setup
run the following to use pctl against it:

1. Create your catalog, for example there is a `examples/profile-catalog-source.yaml` file in the profiles repo
`kubectl apply -f profiles/examples/profile-catalog-source.yaml`
1. Ensure the current context in kubeconfig is set to the `profiles` cluster (`kubectl config current-context` should return `kind-profiles`)
1. Create a `pctl` binary with `make build`.

### Working with profiles

In order to keep versioning parity and drift to a minimum with [Profiles](https://github.com/weaveworks/profiles) the following development
process must be in place:

#### Using local pin

- Open a new branch in profiles and create new code
- In `pctl` do a `replace` to a local location like this: `go mod edit -replace github.com/weaveworks/profiles=<profiles location>`
- Work on the changes and once ready, open a PR with this local mod in place

This has the benefit of being really simple, but the counter is that the PR checks will fail because this can't build on CI.
The better way is to use a dev tag pin if you want CI to be happy as well for most in-place checks and verifications.

## Release process
There are some manual steps right now, should be streamlined soon.

Steps:

1. Create a new release notes file:
   ```sh
   touch docs/release_notes/<version>.md
   ```

1. Copy-and paste the release notes from the draft on the releases page into this file.
   _Note: sometimes the release drafter is a bit of a pain, verify that the notes are
   correct by doing something like: `git log --first-parent tag1..tag2`._

1. Update the `var Version` in `pkg/version/release.go` file to be the desired version.

1. PR the release notes and version bump into main.

1. Navigate to the `Actions` tab and manually trigger the `Release` job. When the job finishes verify that:
  1. The release has been created in Github
    1. With the correct assets
    1. With the correct release notes
  1. The image has been pushed to docker
  1. The image can be pulled and used in a deployment

_Note_ that `<version>` must be in the following format: `v0.0.1`.

### Tests

1. Run `make integration` for integration tests _(This will set up the required env, no need to do anything beforehand.
   Note: if you have a `local-env` running and have created profile catalog sources in it, this will influence your tests.)_
1. Run `make unit` for unit tests
1. Run `make test` to run all tests

#### Configuring Integration Tests

There are two configurable values in the integration tests as the time of this writing.

1. `PCTL_TEST_REPOSITORY_URL` -- configures the remote test repository for the `create-pr` test. This needs to be a
repository the user has push access to and access to create a pull request in GitHub.
1. `GIT_TOKEN` -- it is used by `create-pr` test to creating a pull request on GitHub. Without this token the test
doesn't run.

See `make help` for all development commands.
