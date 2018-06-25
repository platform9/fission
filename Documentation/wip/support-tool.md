# Fission Support Tool
Fission now has rich functionality supported by multiple services, however, it brings the complexity of troubleshooting.
This proposal tends to give a picture of fission support tool that can help both user and developer to locate the problem in short time.
To achieve this, the support tool will dump related kubernetes objects, fission resources and pod logs from the given cluster.

# Functionality

## Environment Information Collection

Before troubleshooting, some of the basic information is needed to give others an overview of kubernetes/fission user test with so that we can locate the problem in short time.

* Fission version
    * Client/Server version

* Kubernetes cluster version
    * Cluster version (i.e v1.9.7-gke.0)
    * Running environment (i.e GKE, AKS and minikube)
    * Nodes version and other information

## Service Logs Collection

The component logs and the logs of interaction between components are important for people to understand what really happened in cluster. Following are components need to collect logs from.

* All fission component pods
* Function pods
* Builder pods
* Environment pods

## Object dumping

Fission is deeply coupled with Kubernetes, most of the objects are created and maintained by it. There is two major type of objects need to be dumped from kubernetes:

* K8S objects
* CRD resources

All objects should be dumped into a readable file format. It will be great if people can reproduce similar environment with these files.

## Information upload

Upload dump files to the specific backend server for support channel to analysis

# CLI Interface

```
$ fission support collect
  NAME:
     fission support collect - Collect pod logs, fission resources and related kubernetes objects for troubleshooting
  
  USAGE:
     fission support collect [command options] [arguments...]
  
  OPTIONS:
     --dumpdir value    Directory to save dump kubernetes objects and fission resources (default: "fission-dump")
     --fissionns value  Namespace of fission installation (default: "fission")
     --builderns value  Namespace of fission package builder (default: "fission-builder")
     --funcns value     Namespace of fission function pod (default: "fission-function")
```

# Thoughts?

1. What to do with sensitive objects like secrets and configmap? Ignore the dump for such objects?
2. The functionality is necessary but not listed above?
