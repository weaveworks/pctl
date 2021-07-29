# pctl
pctl is a cli tool for interacting with [Profiles](https://github.com/weaveworks/profiles)

<!--
To update the TOC, install https://github.com/kubernetes-sigs/mdtoc
and run: mdtoc -inplace README.md
-->

 <!-- toc -->
- [Usage](#usage)
  - [Get](#get)
  - [Add](#add)
    - [Add via Catalog](#add-via-catalog)
    - [Add via URL](#add-via-url)
    - [Configuring a profile](#configuring-a-profile)
    - [Cloning repository resources](#cloning-repository-resources)
    - [Architecture](#architecture)
  - [Prepare](#prepare)
    - [Pre-Flight check](#pre-flight-check)
  - [Catalog service options](#catalog-service-options)
- [Development](#development)
  - [Working with profiles](#working-with-profiles)
    - [Using local pin](#using-local-pin)
    - [Using dev tag pin](#using-dev-tag-pin)
    - [Using doki and the Makefile targets](#using-doki-and-the-makefile-targets)
- [Release process](#release-process)
  - [Tests](#tests)
    - [Configuring Integration Tests](#configuring-integration-tests)
<!-- /toc -->

## Usage

For more information on all commands, run `pctl --help` or `pctl <subcommand> --help`.

### Get
pctl can be used to get installed and catalog profiles, example:
```sh
$ pctl get nginx
INSTALLED PACKAGES
NAMESPACE       NAME            SOURCE                                  AVAILABLE UPDATES
default         nginx-profile    nginx-catalog/weaveworks-nginx/v0.1.0   v0.1.1
CATALOG/PROFILE                                   VERSION DESCRIPTION
nginx-catalog-1/weaveworks-nginx                  0.0.1   This installs nginx.
nginx-catalog-1/bitnami-nginx                     1.0.1   This installs bitnami nginx.
nginx-catalog-1/some-other-nginx                  1.0.1   This installs some other nginx.
```

pctl can be used to get a catalog for profiles, example:
```sh
$ pctl get nginx --catalog
PACKAGE CATALOG  
CATALOG/PROFILE                         VERSION DESCRIPTION
nginx-catalog-1/weaveworks-nginx        0.0.1   This installs nginx.
nginx-catalog-1/some-other-nginx        1.0.1   This installs some other nginx.
```

pctl get also lists all available profiles and installed profiles, example:
```sh
$ pctl get 
INSTALLED PACKAGES
NAMESPACE       NAME            SOURCE                                  AVAILABLE UPDATES
default         pctl-profile    nginx-catalog/weaveworks-nginx/v0.1.0   v0.1.1
CATALOG/PROFILE                                   VERSION DESCRIPTION
nginx-catalog-1/weaveworks-nginx                  0.0.1   This installs nginx.
nginx-catalog-1/bitnami-nginx                     1.0.1   This installs bitnami nginx.
nginx-catalog-1/some-other-nginx                  1.0.1   This installs some other nginx.
prometheus-catalog-1/weaveworks-prometheus        0.0.4   This installs prometheus.
```

pctl can be used to get more information about a specific profile, example:

```
$ pctl get nginx-catalog-1/weaveworks-nginx --profile-version v0.0.1
Catalog         nginx-catalog-1
Name            weaveworks-nginx
Version         0.0.1
Description     This installs nginx.
URL             https://github.com/weaveworks/nginx-profile
Maintainer      weaveworks (https://github.com/weaveworks/profiles)
Prerequisites   Kubernetes 1.18+
```

pctl can be used to get the profile installed in a cluster, example:
```
pctl get --installed
INSTALLED PACKAGES
NAMESPACE       NAME            SOURCE                                  AVAILABLE UPDATES
default         pctl-profile    nginx-catalog/weaveworks-nginx/v0.1.0   v0.1.1
```

It also includes profiles installed through a direct link and a branch:

```
pctl get --installed
NAMESPACE       NAME            SOURCE                                                                          AVAILABLE UPDATES
default         pctl-profile    nginx-catalog/weaveworks-nginx/v0.1.0                                           -
default         update-profile  https://github.com/weaveworks/profiles-examples:branch-and-url:bitnami-nginx    -
```

The source, in case of a branch install, is put together as follows: `url:branch:profile-name`.

Available updates can be viewed in case of profiles which have been installed through a catalog.
If that catalog contains an earlier version, `AVAILABLE UPDATES` section will list them.


### Add

#### Add via Catalog

pctl can be used to install a profile, example:

```
pctl add nginx-catalog/weaveworks-nginx/v0.1.0
```

you can omit the version and pctl will add the latest by default, example:
```
pctl add nginx-catalog/weaveworks-nginx
```


This results in a profile installation folder being created (defaults to the name of the profile). Example:

```
$ pctl add --git-repository flux-system/flux-system nginx-catalog/weaveworks-nginx/v0.1.0
generating profile installation from source: catalog entry nginx-catalog/weaveworks-nginx/v0.1.0
installation completed successfully

$ tree weaveworks-nginx
└── weaveworks-nginx
    ├── artifacts
    │   ├── nested-profile
    │   │   └── nginx-server
    │   │       ├── helm-chart
    │   │       │   ├── HelmRelease.yaml
    │   │       │   ├── kustomization.yaml
    │   │       │   └── nginx
    │   │       │       └── chart
    │   │       │           ├── Chart.lock
    │   │       │           ├── Chart.yaml
    │   │       │           ├── ...
    │   │       ├── kustomization.yaml
    │   │       └── kustomize-flux.yaml
    │   ├── nginx-chart
    │   │   ├── helm-chart
    │   │   │   ├── ConfigMap.yaml
    │   │   │   ├── HelmRelease.yaml
    │   │   │   └── HelmRepository.yaml
    │   │   ├── kustomization.yaml
    │   │   └── kustomize-flux.yaml
    │   └── nginx-deployment
    │       ├── kustomization.yaml
    │       ├── kustomize-flux.yaml
    │       └── nginx
    │           └── deployment
    │               └── deployment.yaml
    └── profile-installation.yaml
```

The `profile-installation.yaml` is the top-level Profile installation object. It describes the profile installation. The artifact's
directory contains all the resources required for deploying the profile. Each of the artifacts corresponds to a
[Flux 2 resource](https://fluxcd.io/docs/components/). The kustomization.yaml files make sure that only the resources
related to the profile, are handled by flux directly. For example, it makes sure that the locally copied nginx/chart files
aren't installed by flux, but by the HelmRelease object.

This can be applied to the cluster by committing it to your flux repository. Since it contains non-kubernetes objects,
applying it by hand isn't supported at the moment. If you are using a flux repository, the `--create-pr` flags provides
an automated way for creating a PR against it. See `pctl install --help` for more details.

#### Add via URL

It's also possible to add from a specific location given a URL, branch and a path. For example, consider the following
`profiles` folder structure:

```
tree
.
├── README.md
├── bitnami-nginx
│   ├── README.md
│   ├── nginx
│   │   └── chart
│   │       ├── Chart.lock
│   │       ├── ...
│   │       └── values.yaml
│   └── profile.yaml
└── weaveworks-nginx
    ├── README.md
    ├── nginx
    │   └── deployment
    │       └── deployment.yaml
    └── profile.yaml
```

Given a development branch called `devel` to add the `bitnami-nginx` profile from this repository, call `add`
with the following parameters:

```
pctl add --name pctl-profile \
             --namespace default \
             --profile-branch devel \
             --profile-url https://github.com/<usr>/<repo> \
             --profile-path bitnami-nginx \
             --out <optional sub-folder inside the flux repository>
```

It's the user's responsibility to make sure that the local `git` setup has access to the url provided with `profile-url`.
It can be any form of url as long as `git clone` understands it.


#### Configuring a profile
You can configure a set of helm values for the charts inside your profile. For example if you add the `weaveworks-nginx` profile
that contains two artifacts that are charts, `nginx-chart` and `nginx-server` you would configure them by creating a config map as
follows:

```
apiVersion: v1
kind: ConfigMap
metadata:
  name: my-profile-values
  namespace: default
data:
  nginx-chart: |
    replicaCount: 3
  nginx-server: |
    service:
      type: ClusterIP
```

The key in the `data` is the name of the chart artifact, and the value is the values.yaml file you want to have provided. You can
pass this config map into pctl add by providing the `--config-map my-profile-values` in during `pctl add`

#### Cloning repository resources

pctl clones all repository local resources and puts them into the target flux repository. This is done so that the user doesn't
have to include access credentials for all private repositories, including nested profiles. However, for repository local
resources to work, the user has to provide the location of the GitRepository object that flux creates when bootstrapping or
creating a flux repository. This resource is usually under the namespace `flux-system` named `flux-system` and of type
GitRepository.

Provide the following information when running add: `--git-repository <namespace>/<name>`.

This looks as follows if installing via a URL:

```
pctl add --name pctl-profile \
             --namespace [default] \
             --profile-branch [main] \
             --profile-url git@github.com:org/private-profile-repo \
             --profile-path <profile-name> \
             --git-repository <namespace>/<name> \
             --out <location of my flux repository>
```

This will result in all the private resources which aren't available from the outside, downloaded locally and added
into the flux repository.

#### Architecture
The below diagram illustrates how pctl add works:

<!--
To update this diagram go to https://miro.com/app/board/o9J_lI2seIg=/
edit, export, save as image (size small) and commit. Easy.
-->
![](/docs/assets/pctl_install.png)


### Prepare

pctl can set up a cluster with all necessary components for `profiles` to work.
To do that, run the following:

```
pctl prepare
```

This will take the latest manifests release under the profiles repository and install
them into the currently set cluster.

There are a number of options which can be set, such as: version, dry-run, context, kube-config.
Please run `pctl help` for all options and defaults.

#### Pre-Flight check

`prepare` will also check whether some needed components are already present in the cluster or not.
The main component which needs to be present is [flux](https://github.com/fluxcd/flux2). This is
checked by looking for some specific CRDs which needs to be present in order for `profiles` to work.
These are as follows:

- buckets.source.toolkit.fluxcd.io
- gitrepositories.source.toolkit.fluxcd.io
- helmcharts.source.toolkit.fluxcd.io
- helmreleases.helm.toolkit.fluxcd.io
- helmrepositories.source.toolkit.fluxcd.io
- kustomizations.kustomize.toolkit.fluxcd.io

### Catalog service options

The catalog service options can be configured via `--catalog-service-name`, `--catalog-service-port` and `--catalog-service-namespace`

## Development

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

#### Using dev tag pin

- Open a new branch in `profiles` and create new code
- Push new branch to fork and open a pull request
- `profiles` creates a dev-tag for that branch in the format of: `<latestReleasedTag>-<branch-name>`
- In `pctl`, do an update like this: `go get github.com/weaveworks/profiles@dev-tag`
- If there are more changes for the `profiles` side, just keep pushing and repeat `go get`. The tag will be updated with the
  new code.
- Work on the changes and open a PR
- Release `profiles` and run `go get github.com/weaveworks/profiles` which should get the latest
- Update remote code

This approach is convenient, however, it should be avoided for the sole reason that it's possible to forget running
the `go get` again to update to the latest version of profiles. The better way is using a new Makefile target called
`update-modules` and we explain why next:

#### Using doki and the Makefile targets

@Deprecated -- this approach isn't supported until we figure out a way to create dev tags for forks on main.
There is a convenient tool for all of these operations called [Doki](https://github.com/weaveworks/doki) and a new
`make` target called `update-modules` which works together nicely. Also, the new make target allows for an extra check
to be executed by CI so the developer doesn't forget to update to latest before merging new code.

The same process as above using `doki` would be:

- Open a new branch in `profiles` and create new code
- Push new branch to fork and open a pull request
- `profiles` creates a dev-tag
- run `doki get dev tag` in `profiles` which should display something like this:
```
➜  profiles git:(keep_in_sync) doki get dev tag
v0.0.4-keep_in_sync
```
- go to `pctl` and edit in `Makefile` this target:
```Makefile
.PHONY: update-modules
	go get \
		$(shell doki mod latest \
			github.com/weaveworks/profiles \
		)
	go mod tidy
```

`github.com/weaveworks/profiles \` => `github.com/weaveworks/profiles@dev-tag \`
- run `make update-modules` which should also synchronise other dependencies
- Write the code and push pctl and create a PR
- This will run a new Action called `Check pinned version`. It will check for version pins in the Makefile
  and fail if there are any. This is to ensure that a `pctl` change which requires new profile code is not accidentally merged
- Once coding has been finished, release `profiles` and remove the pin from the Makefile and run `make update-modules` again. Doki
  will automatically fetch the new released tag.
- Push new code and everything should be green

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

1. PR the release notes into main.
_Note_ that you should also update the `var Version` in `pkg/version/release.go` file with the same tag which is created in the following step
This should be automated with #233

1. Create and push a tag with the new version:
   ```sh
   git tag <version>
   git push origin <version>
   ```

1. The `Create release` action should run. Verify that:
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
