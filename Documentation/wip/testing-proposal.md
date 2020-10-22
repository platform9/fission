# Testing Proposal

This proposal was initially started as a upgrade testing proposal but soon problems that were posed resulted in a bigger proposal.
### Kinds of testing

Most of current integration tests are CLI driven. Fission CLI is used to test execute various test cases. In future we would have to also focus on API level testing as a UI is built for Fission. 
## Needs & patterns

This section only explains the problems/best practices without going into tooling and language used for implementation.
### Separating the test & data

Separating the tests from test data has two aspects - one is separation of concerns and second is scaling the tests without touching the test logic. The test data is a simple data structure which holds all information and test can take data and execute the logic.

As an example today we test "Hello world" for nodejs environment with a simple hello.js like this:

```
fission env create --name nodejs --image fission/node-env
fission fn create --name $fn --env nodejs --code $ROOT/examples/nodejs/hello.js
fission route create --function $fn --url /$fn --method GET
response=$(curl http://$FISSION_ROUTER/$fn)
```
The variables here are environment image, function code & route URL.

If tomorrow if we had to scale this test for all environments, we will have to repeat ourselves. (Violate DRY principle). Instead of that if we encapsulate the test setup & test in a simple function:

```
test_hello_env(envImage, codePath, routeURL){

}
```

And feed it with a dictionary which has all possible combination of tests:

```
{
    node: {"fission/node-env", "test/hello.js", "/hellonode"},
    python: {"fission/python-env", "test/hello.py", "/hellopy"},
    golang: {"fission/go-env", "test/hello.go", "/hellogo"},
    binary: {"fission/binary-env", "test/hello.sh", "/hellobinary"},
}

```
This would achieve a few things:

- Separate the test execution logic from the data it needs clearly.
- For adding new kind of environments, you just need to add one more entry into data structure.

Testing all environments may not be most apt example for this, but there can be potential use cases like this.
### Separating the test & setup/teardown

When we run a test there are typically three distinct phases:

- Setup (Create env, fn, route)
- Test (Curl the function)
- Cleanup (Delete fn, route & env)

It should be possible to separate the before test and after test parts from actual tests at two levels:
- Each test
- A whole test suite

The ability to have clean and separate before and after blocks, apart from separation of concerns, enables:

- Running a suite of tests for same setup (See tagging for suite of tests)
### Tagging tests & running a selection

Over a period of time as tests grow, there will be unit, smoke, integration, performance, soak tests and so on. Ability to run a particular test suite only or a combination of them makes it easy to run for specific purpose. 
### Measuring test times

[Good to have, not a must] Measuring time for tests and reporting somewhere helps over time to monitor trends. Although this job is better done by performance/benchmark tests so it is not a strict requirement

### Cleaner Logging
It would be good to have cleaner/relevant logging as part of build & test. For example something that Ginkgo framework does is it shows error logs only for failed tests.
### Tests in Parallel 

It would be good to be able to run tests in parallel.

## Evaluating the tools/alternatives
### BATS

Bash Automated Testing System is like a enhanced version of bash with support for @test tags and before and after steps & ability to skip tests etc. While it enhances the bash to certain extent, the overall improvement is only marginal.

```
#!/usr/bin/env bats

@test "addition using bc" {
  result="$(echo 2+2 | bc)"
  [ "$result" -eq 4 ]
}

$ bats addition.bats
 ✓ addition using bc
 ✓ addition using dc

2 tests, 0 failures

```
#### Links
- Bats repo: https://github.com/sstephenson/bats
- Runc uses Bats https://github.com/opencontainers/runc/tree/master/tests/integration
### Go Test

The testing package of Go also is quite feature rich for most of the use cases we need. Go 1.7 onwards there is support for setup & teardown parts and parallelism etc. 
#### Go - Testing package

- Support for setup and teardown based on https://golang.org/pkg/testing/#hdr-Main
- Go testing already supports and has examples of table driven tests (Separating test & data), measuring test times and parallel tests
#### Shell execution: Go's Exec Library
GO provides a built in Exec library for working with CLI commands. The package seems good enough for us to work, though a few working examples will help decide better

https://golang.org/pkg/os/exec 

### Using CLI package

Currently we build a CLI and then execute the tests. The tests basically call one of functions from CLI package. If we decide to use a go lang based framework, then we can import the CLI package and then call those functions  by providing them context. This is as good as calling the Fission from CLI, with added benefit of programmability of Go language.

```
func TestSomething(t *testing.T) {
    // Build the Cli context with flags etc.
    ctx := cli.Context{}

    // Pass the ctx to create function
    fnCreate(ctx)
}
```

Some of benefits of using above pattern are:
- We can build a small framework around above core where we can pass various flag combinations etc. and exercise all flags in great detail
- We can use rest of Go testing library and other libraries to build matchers, looping, parallelism etc.
- It allows us to exercise the logic in CLI as well as validate the API at the same time.

### Ginkgo & Gomega

Ginkgo is a BDD framework which works with Gomega matcher library. I will state relevant portions of these two frameworks which can be utilized:

From Ginkgo:

- Global `BeforeSuite` and after `AfterSuite` can be used to have global setup and tear down phases
- For tests `BeforeEach` and `AfterEach` and more such variants to do before and after test tasks.

From Gomega:

Gomega is a matcher library but the `gexec` library makes it really easy to interact with OS execution environment. Some working examples:

- Build and cleanup the Fission CLI before & after the tests
```
var fissionCli string

BeforeSuite(func() {
    var err error
    fissionCli, err = gexec.Build("github.com/fission/fission")
    Ω(err).ShouldNot(HaveOccurred())
})

AfterSuite(func() {
    gexec.CleanupBuildArtifacts()
})
```

- Following will run Fission commands with Fission CLI and print error if there is one (verbosity is configurable)

```
command := exec.Command(fissionCli, "fission env create --name nodejs --image fission/node-env")
session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
Ω(err).ShouldNot(HaveOccurred())
```

- Use Fission CLI's output to validate test results

```
Eventually(session.Out).Should(gbytes.Say("hello [A-Za-z], world"))
```
#### Gomega Matchers

- Gomega provides quite a few built in matchers - so you don't have to code those small usual checks, for example:
```
Ω(ACTUAL).Should(BeTrue()) // The output should be true

Ω(ACTUAL).Should(BeAnExistingFile()) //  The file should already exist

```
There are many more matchers which cane be found here: http://onsi.github.io/gomega/#provided-matchers 

- We can build custom matchers in Go language for reusable logic.

#### Links
Ginkgo: http://onsi.github.io/ginkgo/
Gomega: http://onsi.github.io/gomega/
## Thoughts & Next actions

Based on the discussion with team, here are current thoughts and next action items:

### Thoughts
- As far as possible we should stick to Go's built in testing package
- Ginkgo's cleaner logging feature (Onlu log if there are errors) - is very useful. We can decide to incorporate this in future.
- Gomega's (gexec)[http://onsi.github.io/gomega/#gexec-testing-external-processes] is really neat and some matchers can be used if necessary

### Action items
- How will upgrade test for Fission fit in the framework?
- How will migration of tests happen over time:
  - Aim is to keep existing tests around so that enough validation is in place
  - May be migrate one test at a time
  - How much of current setup etc. will move into framework? For example it is clear that helm commands should be part of test framework as part of setup/teardown. But other sections may or may not be. A RCA needs to be done to analyze and come up with clear demarcation.

## References

- EngineYard uses BATS: https://www.engineyard.com/blog/bats-test-command-line-tools
- AWS CLI Tests, written in Python (CLI itself is also in Python): https://github.com/aws/aws-cli/tree/develop/tests
- Kubernetes uses Ginkgo and Gomega extensively: https://github.com/kubernetes/kubernetes/search?l=Go&q=onsi&type=
- Hashicorp's Mitchell's talk on Advanced testing with go talks about some good patterns to use: https://www.youtube.com/watch?v=8hQG7QlcLBk 
