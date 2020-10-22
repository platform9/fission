# Multi-tenancy in Fission

Multi-tenancy in Fission allows users to create Fission objects, i.e functions, packages, environments and triggers in different namespaces.
It mandates that a function reference secrets, configmaps and its package (if explicitly referenced during function create/update operation) to be present in the same namespace as the function.
This allows user separation and prevents in-advertent access to sensitive data of other users sharing the same cluster.
However, users are allowed and encouraged to share environments to ensure optimal utilization of cluster resources. To achieve this, users can create all the necessary environments in a ns, say ns1 and then go on to create functions in different namespaces and refer to env in ns1.
Users that prefer complete isolation can create their env, functions in the same ns.  

## Roles and privileges

1. Cluster-Admin Role : Fission's services need cluster-admin privileges to monitor, create, update and delete resources across namespaces.
2. Package-getter Role : This role has privileges to do a get, watch and list on fission package objects.
3. Secret-Configmap-getter Role : This role has privileges to do a get, watch and list on secrets and configmaps.

## Service Accounts 

1. fission-fetcher

This SA is created in every namespace that a user creates runtime environments in.
Also created in function namespaces where a user creates functions that use NewDeploy executor backend.

2. fission-builder

This SA is created in every namespace that a user creates builder environments in.

## Role-bindings

1. Package-getter-binding

Every time a user creates a package explicitly in a namespace, this role binding is created in package's namespace (which is also function's namespace). This grants package-getter role to fission-fetcher SA present in the referenced environment's namespace.
If the package is a source package, then, fission-builder SA present in the environment namespace is also added to this role binding.

Next when the user creates a function in the same namespace, if the function's executor type is newdeploy, then, the fission-fetcher SA present in function namespace is also added to the same role binding.

Note : For functions that have executor type poolmgr, the env pods are created in the namespace that env object is created. Whereas, for those functions that have executor type New deploy mgr, the function pods are created in the namespace that function object is created in.

This is because, poolmgr allows env sharing and optimal resource utilization. so generic env pools are created in a different namespace and all functions that prefer sharing this env pool can reference these pools.
If users require strict isolation, they can either create functions with new deploy backend, or, create envs in different namespaces and not share them across functions.

2. Secret-Configmap-getter-binding

Every time a user creates a function in a namespace, Secret-Configmap-getter-binding is created in the same namespace, granting secret-configmap-getter role to fission-fetcher SA present in the referenced environment's namespace in case the executor type is poolmgr.
If the executor type is newdeploymgr, then the same role binding is created in the same namespace as the function, granting the same secret-configmap-getter role to fission-fetcher SA present in the function namespace.

## Examples

1. create a generic python runtime env in ns 1 and function with poolmgr executor type in ns 2 that references it.

```bash
$ fission env create --name python --image fission/python-env --envns ns1
$ fission function create --name func1 --env python --code hello.py --fns ns2
```

2. create a builder and runtime environment in ns3, a source pkg in ns3 and a function referring to this src pkg also in ns3. (for complete isolation, all objects are in ns3)

```bash
$ fission env create --name python-builder-env --builder fission/python-builder --image fission/python-env --ns3
$ fission package create --src src-pkg.zip --env python-builder-env --buildcmd "./build.sh" --pkgns ns3
$ fission fn create --name func3 --fns ns3 --pkg $pkg --entrypoint "user.main"
```

## Note

1. To maintain backward compatibility, fission objects that are created without the ns flags are created in default namespace. Also, the run time env pods in such a case will continue to live in fission-function ns and builder env pods in fission-builder ns
2. Since all envs in a namespace have the same fission-fetcher SA mounted in them, even though multiple envs are created in a namespace and referenced by functions in different namespaces, the SA will have privileges to view those function's secrets if any.
3. Similarly, if there are multiple functions in different namespaces but all sharing an env in one namespace, the fission-fetcher SA in that namespace will have privileges to see all of their secrets.
