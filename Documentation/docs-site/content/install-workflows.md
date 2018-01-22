---
title: "Fission Workflows Installation Guide"
date: 2018-01-22T16:03:27.761Z 
draft: false
---


### Prerequisites

Fission Workflows requires the following components to be installed on your host machine:

- [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
- [helm](https://github.com/kubernetes/helm)


Fission Workflows is deployed on top of a Kubernetes cluster.
If you don't have a Kubernetes cluster, [here's a quick guide to set one up](../kubernetessetup).
It also requires a [Fission](https://github.com/fission/fission) deployment to be present on your Kubernetes cluster. 
If you do not have a Fission deployment, follow [Fission's installation guide](../install).

**(Note that Fission Workflows requires Fission 0.4.1 or higher, with the NATS component installed!)**

### Installing Fission Workflows

Fission Workflows is an add-on to Fission. You can install both
Fission and Fission Workflows using helm charts.

Assuming you have your Kubernetes cluster set up, run the following commands:

```bash
# Add the Fission charts repo
helm repo add fission-charts https://fission.github.io/fission-charts/
helm repo update

# Install Fission 
# This assumes that you do not have a Fission deployment yet, and are installing on a standard Minikube deployment.
# Otherwise see http://fission.io/docs/0.4.0/install/ for more detailed instructions
helm install --wait -n fission-all --namespace fission --set serviceType=NodePort --set analytics=false fission-charts/fission-all --version 0.4.1

# Install Fission Workflows
helm install --wait -n fission-workflows fission-charts/fission-workflows --version 0.2.0
```

### Creating your first workflow

After installing Fission and Workflows, you're all set to run a simple
test workflow. Clone this repository, and from its root directory, run:

```bash
# Fetch the required files, alternatively you could clone the fission-workflow repo
curl https://raw.githubusercontent.com/fission/fission-workflows/0.2.0/examples/whales/fortune.sh > fortune.sh
curl https://raw.githubusercontent.com/fission/fission-workflows/0.2.0/examples/whales/whalesay.sh > whalesay.sh
curl https://raw.githubusercontent.com/fission/fission-workflows/0.2.0/examples/whales/fortunewhale.wf.yaml > fortunewhale.wf.yaml

#
# Add binary environment and create two test functions on your Fission setup:
#
fission env create --name binary --image fission/binary-env
fission function create --name whalesay --env binary --deploy examples/whales/whalesay.sh
fission function create --name fortune --env binary --deploy examples/whales/fortune.sh

#
# Create a workflow that uses those two functions. A workflow is just
# a function that uses the "workflow" environment.
#
fission function create --name fortunewhale --env workflow --src examples/whales/fortunewhale.wf.yaml

#
# Map an HTTP GET to your new workflow function:
#
fission route create --method GET --url /fortunewhale --function fortunewhale

#
# Invoke the workflow with an HTTP request:
#
curl ${FISSION_ROUTER}/fortunewhale
``` 

This last command, the invocation of the workflow, should return a whale saying something wise 

```
 _______________________________________ 
/ A billion here, a couple of billion   \
| there -- first thing you know it adds |
| up to be real money.                  |
|                                       |
\ -- Senator Everett McKinley Dirksen   /
 --------------------------------------- 
    \
     \
      \
                    ##         .
              ## ## ##        ==
           ## ## ## ## ##    ===
       /"""""""""""""""""\___/ ===
      {                       /  ===-
       \______ O           __/
         \    \         __/
          \____\_______/
```

So what happened here?
Let's see what the workflow consists of (for example by running `cat fortunewhale.wf.yaml`): 

```yaml
# This whale shows of a basic workflow that combines both Fission Functions (fortune, whalesay) and internal functions (noop)
apiVersion: 1
output: WhaleWithFortune
tasks:
  InternalFuncShowoff:
    run: noop

  GenerateFortune:
    run: fortune
    requires:
    - InternalFuncShowoff

  WhaleWithFortune:
    run: whalesay
    inputs: "{$.Tasks.GenerateFortune.Output}"
    requires:
    - GenerateFortune
```

What you see is the [YAML](http://yaml.org/)-based workflow definition of the `fortunewhale` workflow.
A workflow consists out of multiple tasks, which are steps that it needs to complete.
Each task consists out of a unique identifier, such as `GenerateFortune`, a reference to a (Fission) function in the `run` field.
Optionally, it can contain `inputs` which allows you to specify inputs to the task, 
as well as contain `requires` which allows you to specify which tasks need to complete before this task can start.
Finally, at the top you will find the `output` field, which allows you to reference the task of which the output should used as the output of the workflow.

In this case, the `fortunewhale` workflow consists out of a sequence of 3 tasks:
```
InternalFuncShowoff -> GenerateFortune -> WhaleWithFortune
```
First, it starts with `InternalFuncShowoff` by running `noop`, which is an *internal function* in the workflow engine.
Internal functions are run inside of the workflow engine, which makes them run much faster at the cost of expressiveness and scalability.
So typically, light-weight functions, such as logic or control flow operations, are good candidates to be used as internal functions.
Besides, a minimal set of predefined internal functions, you can define internal function - there is nothing special about them.

After `InternalFuncShowff` completes, the `GenerateFortune` task can start as its `requires` has been fulfilled.
It runs the `fortune` Fission function, which outputs a random piece of wisdom.

After `GenerateFortune` completes, the `WhaleWithFortune` task can start.
This task uses a javascript expression in its `inputs` to reference the output of the `GenerateFortune` task.
In the inputs of a task you can reference anything in the workflow, such as outputs, inputs, and task definitions, or just provide a constant value.
The workflow engine invokes the `whalesay` fission function with as input the piece of wisdom, which outputs the ASCI whale that wraps the phrase.

Finally, with all tasks completed, the workflow engine uses the top-level `output` field to fetch the output of the `WhaleWithFortune` and return it to the user.
As the workflow engine adheres to the Fission function specification, a Fission workflow is just another Fission Function.
This means that you could use this workflow as a function in the `run` in other workflows.

### What's next?
To learn more about the Fission Workflows system and its advanced concepts, see the [documentation on Github](https://github.com/fission/fission-workflows/tree/master/Docs).

Or, check out the [examples](https://github.com/fission/fission-workflows/tree/0.2.0/examples) for more example workflows.

If something went wrong, we'd love to help -- please [drop by the slack channel](http://slack.fission.io) and ask for help.

