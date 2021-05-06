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
  - [List](#list)
  - [Get](#get)
  - [Prepare](#prepare)
  - [Catalog service options](#catalog-service-options)
- [Development](#development)
  - [Tests](#tests)
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

pctl can be used to install a profile subscription for a profile, example:

```
pctl install nginx-catalog/weaveworks-nginx
generating subscription for profile nginx-catalog/weaveworks-nginx:
```

Then the result will be in profile-subscription.yaml file.

### List
pctl can be used to list the profile subscriptions in a cluster, example:
```
pctl list
NAMESPACE       NAME                    READY
default         nginx-profile-test      False
```

### Get
pctl can be used to get a profile subscriptions in a cluster, example:
```
pctl get --namespace default --name nginx-profile-test
Subscription    nginx-profile-test
Namespace       default
Ready           False
Reason          error when reconciling profile artifacts
```

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

### Tests

1. Run `make integration` for integration tests _(This will set up the required env, no need to do anything beforehand.
   Note: if you have a `local-env` running and have created profile catalog sources in it, this will influence your tests.)_
1. Run `make unit` for unit tests
1. Run `make test` to run all tests

See `make help` for all development commands.
