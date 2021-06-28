## Approach
BASE=v0.0.1
LOCAL=v0.0.1-user-modified
REMOTE=v0.0.2

BASE:
```bash
├── configmap.yaml
├── deployment.yaml
└── namespace.yaml
```

LOCAL:
```bash
├── configmap.yaml # modified
├── deployment.yaml # modified
└── namespace.yaml
```

REMOTE:
```bash
├── nginx.yaml # merged configmap.yaml into deployment.yaml, also some modifications
└── namespace.yaml
```

1. load all three files into memory as `runtime.Object`
- var base map[string]runtime.Object // file name to resource
- var local map[string]runtime.Object
- var remote map[string]runtime.Object
2.
  a. An object exists across all three. Proceed with the current kubectl patch logic
  b. An object exists only in base and local
   i. and local modified it. CONFLICT
   ii. and local did not modify it. DELETE
  c. An object exists only in base and remote
   i. and remote modified it. CONFLICT
   ii. and remote did not modify it. DELETE
  d. An object exists only in local and remote
   i. if they are identical no conflict. DELETE
   ii. if they are different CONFLICT
3. Where does it put the output?
  conflict in deployment resource:
  - do we create a new deployment.yaml with the conflicts?
  - do we put the conflicted resource in the nginx.yaml?
    ```
    search across all three maps, find any object with the same name/namespace/kind/apiVersion
    BASE: deployment.yaml. Some object Deployment
    BASE: deployment.yaml. Some object Deployment
    BASE: nginx.yaml. Some object Deployment
    ```
  ANSWER: put it in the same file the resource exists in the REMOTE
