# pctl
pctl is a cli for interacting with [Profiles](https://github.com/weaveworks/profiles)

## Commands

### Search
pctl can be used to search a catalog for profiles, example:
```
$ pctl search --catalog-url=$CATALOG_URL nginx
searching for profiles matching "nginx":
weaveworks-nginx: This installs nginx.
```

## Local testing

In order to test the CLI you need a profiles catalog controller up and running along with its API.
To get a local setup clone the [Profiles repo](https://github.com/weaveworks/profiles) and run `make local-env`.
This will deploy a local kind cluster with the catalog controller and API running. Once the environment is setup
run the following to use pctl against it:

1. Create your catalog, for example there is a `examples/profile-catalog-source.yaml` file in the profiles repo
`kubectl apply -f profiles/examples/profile-catalog-source.yaml`
1. In a separate terminal run `kubectl -n profiles-system port-forward <profiles-controller-pod-name> 8000:8000` to enable access to the API
1. Run `pctl search --catalog-url http://localhost:8000 <profile-name>` to search for your profile
