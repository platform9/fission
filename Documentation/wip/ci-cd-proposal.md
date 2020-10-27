# Continuous Integration and Delivery of Fission functions

This document outlines a simple CI/CD process for Fission functions which can be extended to any CI/CD tool. Before we start, some level setting for terminology used as the terms are used rather broadly in industry.

## Continuous Integration

CI is made up of a series of broad areas:

- The first step is to compile the source code and convert into artifact which is pushed to artifact repository. Traditionally this has been building a Py or wheel package (Python) or Jar file (Java) but as containers became mainstream the container image became the package. The traditional artifact repositories were replaced by the Docker registries.

- Execution and reporting of unit testing has been a crucial part of the CI cycle and is done after the source code can be compiled successfully.

- Running a static/dynamic code analyzer is the next step in continuous integration. The static code analysis is usually used to measure and report quality metrics and dynamic code scanning/analysis for security.

In the draft version of this proposal we will only consider the source to artifact conversion part and will not dive into unit testing or code scanning/analysis cycles of the CI.

## Continuous Delivery

CD is also composed of a few broad areas focusing on different aspects:

- After CI cycle completes successfully - deploying the artifact to a Dev/Staging environment so that it can be tested by developers and QA teams.

- Once the tests & teams have verified that a function works, the same function should be promoted from Dev/Stage to higher/production environment. The number of environment that a organization maintains varies but the idea of promotion from one environment to higher environment does exist. There are very few organizations who deploy the newer versions of functions directly in production with a A/B setup but that is as of this writing is an exception and not the norm.

- Another aspect of promoting from one environment to higher environment is the configuration for both environments will be different. For ex. the DB connection string will be different for each environment. Or the "maxscale" property for production environment could be higher than that for Dev. The ability to store all these environment specific configurations in some sort of system (Github for normal values and some sort of KMS for sensitive data) and being able to combine the logic and configurations for each environment when deploying is important.

Beyond these points there are integration/automation points such as being able to call a test suite after deployment is done etc. but we will skip for now for brevity.

## 1 Fission specs in a container

Let's start with a simple Fission function which uses specs. A typical directory structure looks like below:

```
.
├── multifile
│   ├── README.md
│   ├── __init__.py
│   ├── main.py
│   ├── message.txt
│   └── readfile.py
└── specs
    ├── README
    ├── env-python.yaml
    ├── fission-deployment-config.yaml
    └── function-pyz.yaml
```

Irrespective of if source code that needs to be built or not, there is a simple command to fire to update the function, which will build (if applicable) and deploy the function:

```
$ fission spec apply
```

If we look at this from CI/CD perspective this process requires:
  1. Source code & specs 
  2. A github push event when any one of the two change
  3. Fission CLI
  4. Kubernetes Config so that the apply command can be run

So if we build a container - which has the above requirements met as installed software (Ex. Fission & Kubectl CLI) or available as environment variable (Github pull token or Kubeconfig), the container can be used as part of CI workflow in any tool such as - Jenkins, Argo, Github Actions, GitLab etc.

The idea is to build a generic container with Fission CLI, Kubernetes CLI and a way to read Github token and Kubeconfig from env variable/mounted files and being able to run `fission spec apply` command.

### 1.1

Instead of building a container in previous section - the same can be achieved by a function. The Github webhook can call a function endpoint which in turn can execute the process similar to inside the container.

## 2 Environment Configurations

There are use cases and reasons to have environment configuration different for each environment such as Dev/Staging etc. Let's assume that we want to vary the `maxscale` in functions and `DB_CONNECTION` in environment 

```
spec:
  InvokeStrategy:
    ExecutionStrategy:
      ExecutorType: newdeploy
      MaxScale: 2 // <-- Varies based on environment deployed in
      MinScale: 1
```

```
    container:
      env:
      - name: DB_CONNECTION
        value: "http://database.url" // <-- Varies based on environment deployed in
```

Without changing anything in Fission spec it is possible to change these things from environment to environment and some of strategies used by people are:

  1. Generate and maintain specs for each environment. This is not a best practice as it leads to drift in code and configuration between environment over time.
  2. Use placeholder variables (i.e. $DB_CONNECTION_VALUE) and replace them for each environment before deploying. This is better in the sense that you are combining changes specific to each environment with spec code but is still a work around sort of.

For environment specific configurations, it is possible to use some sort of templates or overlay mechanism. One of interesting projects using overlays is [Kustomize](https://github.com/kubernetes-sigs/kustomize). In any case as of today the fission spec command does not have a way to use template or modify values using overlay and it is worth exploring this approach for fission spec.

## 3 Promotion from one environment to another

This necessarily does not fall in the area of Fission per se but it would be fairly easy to build a pipeline in the the tool used for CI/CD if we have container mentioned in (1) and even work around mentioned in (2).

## Action Items

As a first step it would be good to build a simple container mentioned in (1) and use it in various tools to understand the value it adds and any unknowns. The next steps would be to build a full end to end pipeline from source to production.
