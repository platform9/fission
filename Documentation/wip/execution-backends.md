
# Fission Pool manager

Fission's currently uses a pool of running "environments" and specialized them for execution of a function. This design served the cold start use cases well but this is not the only strategy for creation and execution of functions. For example requirements for a new execution backend have been discussed in https://github.com/fission/fission/issues/193. This document aims to discuss the currently under development "newdeploy" backend and related thoughts

# Executor

A new layer - executor now sits between the router and actual backends are responsible for all of heavy lifting for execution of functions. Executor layer is responsible for accepting requests from router and checking with cache before calling on a backend for execution of a function.

# Backend

A backend is responsible for execution of a function - which can involve provisioning appropriate objects in Kubernetes. So with the new design Pool manager becomes one of the backends. As of this writing there are two backends which are described as:

### Pool Manager Backend

Pool manager backend uses a pool of environment pods and specialized them when a function is invoked. The specialized pods are cleaned up if not in use after a few minutes. More details on Pool manager can be found here: https://github.com/fission/fission/blob/5c470735185b980c1f7987921db360e91c65573b/Documentation/Architecture.md

### New Deploy Backend

New Deploy backend create a Kubernetes deployment, a Kubernetes Service for a given function. It additionally creates a HorizontalPodAutoscaler if scale parameters are provided. The creation of deployment and service can be eager or lazy based on input. 

### Execution Strategy

While this is still a WIP, parameters that affect execution behavior of function are based on `InvokeStrategy`. A invoke strategy defines the `strategyType` and actual strategy parameters encapsulated in the strategy object. 

```
InvokeStrategy struct {
		ExecutionStrategy ExecutionStrategy
		StrategyType      StrategyType
	}
```  
For example in above case the strategy type is `ExecutionStrategy` and the corresponding parameters are listed below.

```
	ExecutionStrategy struct {
		Backend       BackendType
		MinScale      int
		MaxScale      int
		EagerCreation bool
	}
  ```

In future there could be more strategies for different use cases.

## Dispatch to backend

As of now one of the backends is chosen based on a simple flag in `ExecutionStrategy`. In future there might be a intelligent/hybrid ways of choosing a backend. For example initial requests of a function could be served from a pool manager while later scaling could be served by a NewDeploy backend
