# pctl
pctl is a cli tool for interacting with [Profiles](https://github.com/weaveworks/profiles)

<!--
To update the TOC, install https://github.com/kubernetes-sigs/mdtoc
and run: mdtoc -inplace README.md
-->

 <!-- toc -->
- [Usage](#usage)
  - [Search](#search)
  - [Show](#show)
  - [Install](#install)
    - [Architecture](#architecture)
  - [List](#list)
  - [Prepare](#prepare)
    - [Pre-Flight check](#pre-flight-check)
  - [Catalog service options](#catalog-service-options)
- [Development](#development)
- [Release process](#release-process)
  - [Tests](#tests)
    - [Configuring Integration Tests](#configuring-integration-tests)
<!-- /toc -->

## Usage

For more information on all commands, run `pctl --help` or `pctl <subcommand> --help`.

### Search
pctl can be used to search a catalog for profiles, example:
```sh
$ pctl search nginx
CATALOG/PROFILE                         VERSION DESCRIPTION
nginx-catalog-1/weaveworks-nginx        0.0.1   This installs nginx.
nginx-catalog-1/some-other-nginx        1.0.1   This installs some other nginx.
```

### Show

pctl can be used to get more information about a specific profile, example:

```
$ pctl show nginx-catalog-1/weaveworks-nginx
Catalog         nginx-catalog-1
Name            weaveworks-nginx
Version         0.0.1
Description     This installs nginx.
URL             https://github.com/weaveworks/nginx-profile
Maintainer      weaveworks (https://github.com/weaveworks/profiles)
Prerequisites   Kubernetes 1.18+
```

### Install

pctl can be used to install a profile, example:

```
pctl install nginx-catalog/weaveworks-nginx/v0.1.0
```

you can omit the version and pctl will install the latest by default, example:
```
pctl install nginx-catalog/weaveworks-nginx
```


This results in a profile installation folder being created (defaults to the name of the profile). Example:

```
$ pctl install nginx-catalog/weaveworks-nginx/v0.1.0
generating a profile installation for nginx-catalog/weaveworks-nginx:

$ tree weaveworks-nginx
weaveworks-nginx
├── artifacts
│   ├── dokuwiki
│   │   ├── HelmRelease.yaml
│   │   └── HelmRepository.yaml
│   ├── nested-profile
│   │   └── nginx-server
│   │       ├── GitRepository.yaml
│   │       └── HelmRelease.yaml
│   └── nginx-deployment
│       ├── GitRepository.yaml
│       └── Kustomization.yaml
└── profile.yaml

```

The `profile.yaml` is the top-level Profile installation object. It describes the profile installation. The artifacts
directory contains all of the resources required for deploying the profile. Each of the artifacts corresponds to a
[Flux 2 resource](https://fluxcd.io/docs/components/).


This can be applied directly to the cluster `kubectl apply -R -f weaveworks-nginx/` or by comitting it to your
flux repository. If you are using a flux repository the `--create-pr` flags provides an automated way for creating a PR
against your flux repository. See `pctl install --help` for more details.

#### Architecture
The below diagram illustrates how pctl install works:

<!--
To update this diagram go to https://miro.com/app/board/o9J_lI2seIg=/
edit, export, save as image (size small) and commit. Easy.
-->
![](/docs/assets/pctl_install.png)



### List
pctl can be used to list the profile installed in a cluster, example:
```
pctl list
NAMESPACE       NAME            SOURCE                                  AVAILABLE UPDATES
default         pctl-profile    nginx-catalog/weaveworks-nginx/v0.1.0   v0.1.1
```

It also includes profiles installed through a direct link and a branch:

```
pctl list
NAMESPACE       NAME            SOURCE                                                                          AVAILABLE UPDATES
default         pctl-profile    nginx-catalog/weaveworks-nginx/v0.1.0                                           -
default         update-profile  https://github.com/weaveworks/profiles-examples:branch-and-url:bitnami-nginx    -
```

The source, in case of a branch install, is put together as follows: `url:branch:profile-name`.

Available updates can be viewed in case of profiles which have been installed through a catalog.
If that catalog contains an earlier version, `AVAILABLE UPDATES` section will list them.

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
