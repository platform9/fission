# 0.8.0
[Documentation](https://docs.fission.io/0.8.0/)
## Downloads for 0.8.0


filename | sha256 hash
-------- | -----------
[fission-all-0.8.0.tgz](https://github.com/fission/fission/releases/download/0.8.0/fission-all-0.8.0.tgz) | `bf643aab4642c1d9f73235ffe10c51b74e46474d765cefc29139a560c702e515`
[fission-core-0.8.0.tgz](https://github.com/fission/fission/releases/download/0.8.0/fission-core-0.8.0.tgz) | `5473832a696b1e4e3316878f84dad813e3fbcc7b270563d11a0b02345f1b0370`
[fission-cli-linux](https://github.com/fission/fission/releases/download/0.8.0/fission-cli-linux) | `db6d66aae513fd1bc8f969c4e781e7f5bd9369dfcf5389edc3b6afb48522b387`
[fission-cli-osx](https://github.com/fission/fission/releases/download/0.8.0/fission-cli-osx) | `3b2cff596402d4a35a5347cb1d0cc8c90e09cdc88c6a0a33f99cc510c7ccbb8b`
[fission-cli-windows.exe](https://github.com/fission/fission/releases/download/0.8.0/fission-cli-windows.exe) | `41fce28b3f5d34e63910882d8a7fbee6241a8644e1a65a133a30714bb666d7f9`

# Change Log

## [0.8.0](https://github.com/fission/fission/tree/0.8.0) (2018-06-05)
[Full Changelog](https://github.com/fission/fission/compare/0.7.2...0.8.0)

**Merged pull requests:**

- Pre-install/pre-upgrade hooks to verify func references and assign restricted role bindings [\#717](https://github.com/fission/fission/pull/717) ([smruthi2187](https://github.com/smruthi2187))
- Logger daemonset's update strategy [\#714](https://github.com/fission/fission/pull/714) ([vishal-biyani](https://github.com/vishal-biyani))
- Check spec directory exists before reading spec files [\#709](https://github.com/fission/fission/pull/709) ([life1347](https://github.com/life1347))
- Formatted specifiers are not compatible with variables [\#706](https://github.com/fission/fission/pull/706) ([xiekeyang](https://github.com/xiekeyang))
- Indicate HTTP status code by library const [\#703](https://github.com/fission/fission/pull/703) ([xiekeyang](https://github.com/xiekeyang))
- docker-distribution version bump for windows compatibility [\#691](https://github.com/fission/fission/pull/691) ([thejosephstevens](https://github.com/thejosephstevens))
- Java environment Design & considerations [\#642](https://github.com/fission/fission/pull/642) ([vishal-biyani](https://github.com/vishal-biyani))
- Working version of Ingress integration [\#688](https://github.com/fission/fission/pull/688) ([vishal-biyani](https://github.com/vishal-biyani))
- Update k8s dependencies to 1.10 [\#687](https://github.com/fission/fission/pull/687) ([life1347](https://github.com/life1347))
- Add time trigger cron spec examination tool [\#680](https://github.com/fission/fission/pull/680) ([life1347](https://github.com/life1347))
- Fission metrics integration [\#677](https://github.com/fission/fission/pull/677) ([soamvasani](https://github.com/soamvasani))
- Replace Werkzeug  with Bjoern as underlying WSGI server [\#672](https://github.com/fission/fission/pull/672) ([life1347](https://github.com/life1347))
- Enabling multi-tenancy for fission objects. [\#655](https://github.com/fission/fission/pull/655) ([smruthi2187](https://github.com/smruthi2187))

## [0.7.2](https://github.com/fission/fission/tree/0.7.2) (2018-05-05)
[Full Changelog](https://github.com/fission/fission/compare/0.7.1...0.7.2)

**Merged pull requests:**

- Add benchmark script [\#666](https://github.com/fission/fission/pull/666) ([life1347](https://github.com/life1347))
- Fixed the issue with update wiping values [\#663](https://github.com/fission/fission/pull/663) ([vishal-biyani](https://github.com/vishal-biyani))
- Fix newdeploy backend failed to delete deployment due to incorrect resource version [\#657](https://github.com/fission/fission/pull/657) ([life1347](https://github.com/life1347))
- Function update should be possible without change to code [\#652](https://github.com/fission/fission/pull/652) ([vishal-biyani](https://github.com/vishal-biyani))
- Fixes the issue with fn test and adds relevant test cases, fixes \#650 [\#651](https://github.com/fission/fission/pull/651) ([vishal-biyani](https://github.com/vishal-biyani))
- Fix test cases occasional failure [\#647](https://github.com/fission/fission/pull/647) ([life1347](https://github.com/life1347))
- Change time precision for fluentd influxdb plugin to nano second [\#646](https://github.com/fission/fission/pull/646) ([life1347](https://github.com/life1347))
- Setting buildStatus to pending when function's source archive is updated. [\#637](https://github.com/fission/fission/pull/637) ([smruthi2187](https://github.com/smruthi2187))
- Fix SEGFAULT issue when buildmgr failed to update package [\#635](https://github.com/fission/fission/pull/635) ([life1347](https://github.com/life1347))
- Fix executor does not reap specialized function pod when env no longer exists [\#633](https://github.com/fission/fission/pull/633) ([life1347](https://github.com/life1347))
- Update readme to point to the proper link [\#628](https://github.com/fission/fission/pull/628) ([jgall](https://github.com/jgall))
- Changes needed for release 0.7.1 [\#622](https://github.com/fission/fission/pull/622) ([smruthi2187](https://github.com/smruthi2187))
- Add default value to cli flag [\#619](https://github.com/fission/fission/pull/619) ([life1347](https://github.com/life1347))
- Remove port forward in tests for router, controller and nats pods  [\#611](https://github.com/fission/fission/pull/611) ([smruthi2187](https://github.com/smruthi2187))
- meaningful error message when fetch request is received for a package when build is not successful. [\#661](https://github.com/fission/fission/pull/661) ([smruthi2187](https://github.com/smruthi2187))
- Delete deployment with proper delete propagation policy [\#630](https://github.com/fission/fission/pull/630) ([life1347](https://github.com/life1347))
- Fix buildmgr SEGFAULT when it failed to update package [\#626](https://github.com/fission/fission/pull/626) ([life1347](https://github.com/life1347))
- Fission upgrade tests [\#605](https://github.com/fission/fission/pull/605) ([vishal-biyani](https://github.com/vishal-biyani))
- Removed the fn pods functionality [\#594](https://github.com/fission/fission/pull/594) ([vishal-biyani](https://github.com/vishal-biyani))
- Testing proposal: Requirements and frameworks exploration [\#581](https://github.com/fission/fission/pull/581) ([vishal-biyani](https://github.com/vishal-biyani))

## [0.7.1](https://github.com/fission/fission/tree/0.7.1) (2018-04-10)
[Full Changelog](https://github.com/fission/fission/compare/0.7.0...0.7.1)

**Merged pull requests:**

- Prevent releasing idle connections because transport is shared. [\#609](https://github.com/fission/fission/pull/609) ([smruthi2187](https://github.com/smruthi2187))
- Fix components crash before crds creation [\#602](https://github.com/fission/fission/pull/602) ([life1347](https://github.com/life1347))
- updates to changelog. [\#598](https://github.com/fission/fission/pull/598) ([smruthi2187](https://github.com/smruthi2187))
- changes needed for release 0.7.0 [\#597](https://github.com/fission/fission/pull/597) ([smruthi2187](https://github.com/smruthi2187))
- `fission X create --spec` flags for env and trigger create commands [\#607](https://github.com/fission/fission/pull/607) ([soamvasani](https://github.com/soamvasani))
- Updating releasing guideliness with a few more details. [\#599](https://github.com/fission/fission/pull/599) ([smruthi2187](https://github.com/smruthi2187))
- Add deprecated message to subcommand pods [\#592](https://github.com/fission/fission/pull/592) ([life1347](https://github.com/life1347))
- Add validate function to crd resource and do validate before creation/update [\#580](https://github.com/fission/fission/pull/580) ([life1347](https://github.com/life1347))
- Invalidate stale router cache entry with podIP's for deleted pods. [\#546](https://github.com/fission/fission/pull/546) ([smruthi2187](https://github.com/smruthi2187))
- Use a separate controller loop to watch functions change and create a service [\#544](https://github.com/fission/fission/pull/544) ([life1347](https://github.com/life1347))
- E2E test for NATS-streaming trigger [\#338](https://github.com/fission/fission/pull/338) ([soamvasani](https://github.com/soamvasani))

## [0.7.0](https://github.com/fission/fission/tree/0.7.0) (2018-04-02)
[Full Changelog](https://github.com/fission/fission/compare/0.6.1...0.7.0)

**Merged pull requests:**

- bug fix: spec dir flag [\#595](https://github.com/fission/fission/pull/595) ([xiekeyang](https://github.com/xiekeyang))
- Add steps to set FISSION\_ROUTER env variable & update docs [\#593](https://github.com/fission/fission/pull/593) ([life1347](https://github.com/life1347))
- Adding routerUrl parameter for kubewatch, timer, message queue trigge… [\#591](https://github.com/fission/fission/pull/591) ([smruthi2187](https://github.com/smruthi2187))
- Uses proper way to get server URL [\#587](https://github.com/fission/fission/pull/587) ([vishal-biyani](https://github.com/vishal-biyani))
- Check if the requested file already exists in fetcher and skip fetch [\#584](https://github.com/fission/fission/pull/584) ([smruthi2187](https://github.com/smruthi2187))
- Add golang example to installation guide [\#578](https://github.com/fission/fission/pull/578) ([clee](https://github.com/clee))
- Fixes the issue \#559 with env versions [\#569](https://github.com/fission/fission/pull/569) ([vishal-biyani](https://github.com/vishal-biyani))
- Add post-upgrade-job to track fission upgrade [\#564](https://github.com/fission/fission/pull/564) ([life1347](https://github.com/life1347))
- Prepending a slash to user input url if missing. [\#547](https://github.com/fission/fission/pull/547) ([smruthi2187](https://github.com/smruthi2187))
- Add verbosity flag and verbose logs for portforwarder [\#575](https://github.com/fission/fission/pull/575) ([soamvasani](https://github.com/soamvasani))
- Spec validator, better errors, apply waits for previous build  [\#560](https://github.com/fission/fission/pull/560) ([soamvasani](https://github.com/soamvasani))
- Tests for function update [\#550](https://github.com/fission/fission/pull/550) ([vishal-biyani](https://github.com/vishal-biyani))
- Show fission deployment version with cli [\#538](https://github.com/fission/fission/pull/538) ([life1347](https://github.com/life1347))

## [0.6.1](https://github.com/fission/fission/tree/0.6.1) (2018-03-22)
[Full Changelog](https://github.com/fission/fission/compare/0.6.0...0.6.1)

**Merged pull requests:**

- This change fixes an error in a yaml file in the fission-core chart. [\#563](https://github.com/fission/fission/pull/563) ([associationrule](https://github.com/associationrule))
- \[ci skip\] update release number [\#561](https://github.com/fission/fission/pull/561) ([appleboy](https://github.com/appleboy))
- Fixes \#537 - warning should not be given when updating to newdeploy [\#545](https://github.com/fission/fission/pull/545) ([vishal-biyani](https://github.com/vishal-biyani))
- Docs update [\#542](https://github.com/fission/fission/pull/542) ([soamvasani](https://github.com/soamvasani))
- Release script updates [\#541](https://github.com/fission/fission/pull/541) ([soamvasani](https://github.com/soamvasani))
- Show warning when trying to create a route with non-existent function \(\#238\) [\#539](https://github.com/fission/fission/pull/539) ([life1347](https://github.com/life1347))
- Fix executor failed to clean cache & kubeobjs after function deleted \(\#533\) [\#534](https://github.com/fission/fission/pull/534) ([life1347](https://github.com/life1347))
- Delete healthz log [\#525](https://github.com/fission/fission/pull/525) ([smruthi2187](https://github.com/smruthi2187))
- Always retry when istio is enabled. [\#536](https://github.com/fission/fission/pull/536) ([life1347](https://github.com/life1347))
- Fix executor tries to create a new deployment when a function is updated [\#524](https://github.com/fission/fission/pull/524) ([life1347](https://github.com/life1347))
- Add container spec config options  to \(build\) environments [\#413](https://github.com/fission/fission/pull/413) ([erwinvaneyk](https://github.com/erwinvaneyk))

## [0.6.0](https://github.com/fission/fission/tree/0.6.0) (2018-03-01)
[Full Changelog](https://github.com/fission/fission/compare/latest...0.6.0)

**Merged pull requests:**

- Detect fission namespace in cli [\#519](https://github.com/fission/fission/pull/519) ([soamvasani](https://github.com/soamvasani))
- Default values for FISSION\_\* env vars [\#518](https://github.com/fission/fission/pull/518) ([soamvasani](https://github.com/soamvasani))
- Add chart version to job name [\#516](https://github.com/fission/fission/pull/516) ([soamvasani](https://github.com/soamvasani))
- Fix CLI not update function's secret/configmap correctly [\#512](https://github.com/fission/fission/pull/512) ([life1347](https://github.com/life1347))
- Adds latest tags and pushes to dockerhub for fetcher and fission-bundle [\#509](https://github.com/fission/fission/pull/509) ([vishal-biyani](https://github.com/vishal-biyani))
- Fixes the backward compatibility with older environment versions [\#508](https://github.com/fission/fission/pull/508) ([vishal-biyani](https://github.com/vishal-biyani))
- Update Fn: Executor New Deployment [\#504](https://github.com/fission/fission/pull/504) ([vishal-biyani](https://github.com/vishal-biyani))
- Adds default resources for fetcher pod [\#500](https://github.com/fission/fission/pull/500) ([vishal-biyani](https://github.com/vishal-biyani))
- Documentation Revamp [\#496](https://github.com/fission/fission/pull/496) ([vishal-biyani](https://github.com/vishal-biyani))
- Delete and list orphan pkgs [\#468](https://github.com/fission/fission/pull/468) ([smruthi2187](https://github.com/smruthi2187))
- Service type ClusterIP - Controller port forward through CLI [\#431](https://github.com/fission/fission/pull/431) ([prithviramesh](https://github.com/prithviramesh))
- Istio integration [\#421](https://github.com/fission/fission/pull/421) ([life1347](https://github.com/life1347))
- Implement support for Azure storage message queue triggers [\#371](https://github.com/fission/fission/pull/371) ([peterhuene](https://github.com/peterhuene))

## [latest](https://github.com/fission/fission/tree/latest) (2018-03-01)
[Full Changelog](https://github.com/fission/fission/compare/0.5.0...latest)

**Merged pull requests:**

- Release checklist [\#522](https://github.com/fission/fission/pull/522) ([soamvasani](https://github.com/soamvasani))
- Fix post-install-job container failure due to command not found [\#514](https://github.com/fission/fission/pull/514) ([life1347](https://github.com/life1347))
- Replace the release with the latest tag. [\#513](https://github.com/fission/fission/pull/513) ([smruthi2187](https://github.com/smruthi2187))
- Go: Set image to right version, update example readme [\#497](https://github.com/fission/fission/pull/497) ([soamvasani](https://github.com/soamvasani))
- Remove a noisy log from router [\#495](https://github.com/fission/fission/pull/495) ([soamvasani](https://github.com/soamvasani))
- Improve release script [\#494](https://github.com/fission/fission/pull/494) ([life1347](https://github.com/life1347))
- Update SHA256 HASH in CHANGELOG.md due to binaries update [\#493](https://github.com/fission/fission/pull/493) ([life1347](https://github.com/life1347))
- Go builder for single file functions [\#492](https://github.com/fission/fission/pull/492) ([soamvasani](https://github.com/soamvasani))
- CI modifications [\#491](https://github.com/fission/fission/pull/491) ([smruthi2187](https://github.com/smruthi2187))
- Add upgrade guide from 0.4.x to 0.5.0 [\#490](https://github.com/fission/fission/pull/490) ([life1347](https://github.com/life1347))
- Version -\> 0.5.0 [\#489](https://github.com/fission/fission/pull/489) ([life1347](https://github.com/life1347))

## [0.5.0](https://github.com/fission/fission/tree/0.5.0) (2018-02-07)
[Full Changelog](https://github.com/fission/fission/compare/0.4.1...0.5.0)

**Merged pull requests:**

- Migrate project.json to dotnet.csproj & do build in dotnet container [\#488](https://github.com/fission/fission/pull/488) ([life1347](https://github.com/life1347))
- Fix binary environment build failure due to package not found [\#487](https://github.com/fission/fission/pull/487) ([life1347](https://github.com/life1347))
- Fix possible context leak problem [\#483](https://github.com/fission/fission/pull/483) ([life1347](https://github.com/life1347))
- Removed limit on max number of channels in NATS Streaming deployment [\#482](https://github.com/fission/fission/pull/482) ([erwinvaneyk](https://github.com/erwinvaneyk))
- Add glide flag to strip nested vendor [\#480](https://github.com/fission/fission/pull/480) ([life1347](https://github.com/life1347))
- Extend perl examples to use more http features [\#479](https://github.com/fission/fission/pull/479) ([LittleFox94](https://github.com/LittleFox94))
- Fluentd image tag issue in tests - an additional tag was appended [\#469](https://github.com/fission/fission/pull/469) ([vishal-biyani](https://github.com/vishal-biyani))
- Fix broken redirect in python example [\#467](https://github.com/fission/fission/pull/467) ([soamvasani](https://github.com/soamvasani))
- Add readiness probe to go env [\#461](https://github.com/fission/fission/pull/461) ([life1347](https://github.com/life1347))
- Fix fission bundle build failure [\#456](https://github.com/fission/fission/pull/456) ([life1347](https://github.com/life1347))
- Convert build.sh to a multi-stage Dockerfile. [\#452](https://github.com/fission/fission/pull/452) ([justinbarrick](https://github.com/justinbarrick))
- NewDeploy Doc [\#432](https://github.com/fission/fission/pull/432) ([vishal-biyani](https://github.com/vishal-biyani))
- Add go vet check [\#430](https://github.com/fission/fission/pull/430) ([life1347](https://github.com/life1347))
- Fix potential nil pointer problem [\#485](https://github.com/fission/fission/pull/485) ([life1347](https://github.com/life1347))
- Add simple usage doc for accessing secret/configmap in function [\#484](https://github.com/fission/fission/pull/484) ([life1347](https://github.com/life1347))
- Helm hook bugfixes: run on upgrade, delete on completion [\#473](https://github.com/fission/fission/pull/473) ([soamvasani](https://github.com/soamvasani))
- Archive pruner [\#471](https://github.com/fission/fission/pull/471) ([smruthi2187](https://github.com/smruthi2187))
- Build and push fluentd image on release; update chart to use that image [\#462](https://github.com/fission/fission/pull/462) ([soamvasani](https://github.com/soamvasani))
- Installation instructions for Fission Workflows [\#453](https://github.com/fission/fission/pull/453) ([erwinvaneyk](https://github.com/erwinvaneyk))
- Block build requests before environment builder is ready [\#437](https://github.com/fission/fission/pull/437) ([life1347](https://github.com/life1347))
- Show warning when user tries to create a function with a non-existed environment [\#436](https://github.com/fission/fission/pull/436) ([life1347](https://github.com/life1347))
- Declarative application specifications for Fission [\#422](https://github.com/fission/fission/pull/422) ([soamvasani](https://github.com/soamvasani))
- Functions have access to secrets/configmaps specified by the user [\#399](https://github.com/fission/fission/pull/399) ([prithviramesh](https://github.com/prithviramesh))
- Newdeploy backend [\#387](https://github.com/fission/fission/pull/387) ([vishal-biyani](https://github.com/vishal-biyani))

## [0.4.1](https://github.com/fission/fission/tree/0.4.1) (2018-01-20)
[Full Changelog](https://github.com/fission/fission/compare/0.4.0...0.4.1)

**Merged pull requests:**

- Fix python environment failed to launch [\#451](https://github.com/fission/fission/pull/451) ([life1347](https://github.com/life1347))
- use time.Since instead of time.Now\(\).Sub [\#449](https://github.com/fission/fission/pull/449) ([wgliang](https://github.com/wgliang))
- Fix fission function logs [\#448](https://github.com/fission/fission/pull/448) ([prithviramesh](https://github.com/prithviramesh))
- Integration test improvements [\#447](https://github.com/fission/fission/pull/447) ([soamvasani](https://github.com/soamvasani))
- Use storageClassName in Helm Charts \(\#444\) [\#445](https://github.com/fission/fission/pull/445) ([agrahamlincoln](https://github.com/agrahamlincoln))
- Fscache support for multiple kubernetes objects [\#435](https://github.com/fission/fission/pull/435) ([vishal-biyani](https://github.com/vishal-biyani))
- Improve travi-ci test scripts [\#434](https://github.com/fission/fission/pull/434) ([life1347](https://github.com/life1347))
- Fix glide failed to check out github.com/dsnet/compress [\#429](https://github.com/fission/fission/pull/429) ([life1347](https://github.com/life1347))
- Golang v2 environment -- runtime and builder [\#427](https://github.com/fission/fission/pull/427) ([soamvasani](https://github.com/soamvasani))
- \[Issue 423\] build logs not saved on build error [\#426](https://github.com/fission/fission/pull/426) ([life1347](https://github.com/life1347))
- Add support for httproute Host matching [\#425](https://github.com/fission/fission/pull/425) ([ajbouh](https://github.com/ajbouh))
- Removed openshift specifics as they are no longer necessary [\#424](https://github.com/fission/fission/pull/424) ([karmab](https://github.com/karmab))
- Overwrite request host with internal host to prevent request rejection [\#419](https://github.com/fission/fission/pull/419) ([life1347](https://github.com/life1347))
- Fix pool manager crash problem if failed at http call [\#418](https://github.com/fission/fission/pull/418) ([life1347](https://github.com/life1347))
- Update nats dependencies [\#411](https://github.com/fission/fission/pull/411) ([life1347](https://github.com/life1347))
- Prepare Fission for IPv6 uses [\#408](https://github.com/fission/fission/pull/408) ([valentin2105](https://github.com/valentin2105))
- Executor API panics if there is err in getting function from backends [\#407](https://github.com/fission/fission/pull/407) ([vishal-biyani](https://github.com/vishal-biyani))
- fission function logs returns logs in correct order now [\#405](https://github.com/fission/fission/pull/405) ([prithviramesh](https://github.com/prithviramesh))
- Fetcher retry [\#403](https://github.com/fission/fission/pull/403) ([vishal-biyani](https://github.com/vishal-biyani))
- Add fission/builder image [\#397](https://github.com/fission/fission/pull/397) ([erwinvaneyk](https://github.com/erwinvaneyk))
- Changed podName to a generic objectReference in cache implementation [\#391](https://github.com/fission/fission/pull/391) ([vishal-biyani](https://github.com/vishal-biyani))
- Add package command [\#385](https://github.com/fission/fission/pull/385) ([life1347](https://github.com/life1347))
- Executor abstraction [\#384](https://github.com/fission/fission/pull/384) ([vishal-biyani](https://github.com/vishal-biyani))

## [0.4.0](https://github.com/fission/fission/tree/0.4.0) (2017-11-15)
[Full Changelog](https://github.com/fission/fission/compare/0.4.0rc...0.4.0)

**Merged pull requests:**

- Added python example to demonstrate status codes. [\#395](https://github.com/fission/fission/pull/395) ([c0dyhi11](https://github.com/c0dyhi11))
- created weather.js in node.js examples, modified README.md [\#394](https://github.com/fission/fission/pull/394) ([svicenteruiz](https://github.com/svicenteruiz))
- Delete failed helm releases to prevent test case failure [\#393](https://github.com/fission/fission/pull/393) ([life1347](https://github.com/life1347))
- Added AWS to install cloud setup [\#392](https://github.com/fission/fission/pull/392) ([joshkelly](https://github.com/joshkelly))
- Fix functionReferenceResolver return out-of-date function metadata [\#390](https://github.com/fission/fission/pull/390) ([life1347](https://github.com/life1347))
- changes made to FluentD configuration to circumvent Logger daemonset [\#380](https://github.com/fission/fission/pull/380) ([prithviramesh](https://github.com/prithviramesh))

## [0.4.0rc](https://github.com/fission/fission/tree/0.4.0rc) (2017-11-08)
[Full Changelog](https://github.com/fission/fission/compare/0.3.0...0.4.0rc)

**Merged pull requests:**

- Use store to sync functions/triggers for fast synchronization [\#382](https://github.com/fission/fission/pull/382) ([life1347](https://github.com/life1347))
- Switch from ThirdPartyResources to CustomResourceDefinitions [\#381](https://github.com/fission/fission/pull/381) ([life1347](https://github.com/life1347))
- changed helm install pullPolicy from Always to IfNotPresent when building local docker image [\#378](https://github.com/fission/fission/pull/378) ([prithviramesh](https://github.com/prithviramesh))
- Reduce function resolving time [\#376](https://github.com/fission/fission/pull/376) ([life1347](https://github.com/life1347))
- Fix builder manager issues [\#367](https://github.com/fission/fission/pull/367) ([life1347](https://github.com/life1347))
- Test functions 236 [\#355](https://github.com/fission/fission/pull/355) ([vishal-biyani](https://github.com/vishal-biyani))
- Make default node-env use alpine. List envs in documentation. [\#354](https://github.com/fission/fission/pull/354) ([rapitable](https://github.com/rapitable))
- Update k8s client version to 4.0.0 [\#351](https://github.com/fission/fission/pull/351) ([life1347](https://github.com/life1347))

## [0.3.0](https://github.com/fission/fission/tree/0.3.0) (2017-09-29)
[Full Changelog](https://github.com/fission/fission/compare/0.3.0-rc...0.3.0)

**Merged pull requests:**

- dotnet20 build fixes [\#365](https://github.com/fission/fission/pull/365) ([soamvasani](https://github.com/soamvasani))
- Add experimental deploy script [\#364](https://github.com/fission/fission/pull/364) ([erwinvaneyk](https://github.com/erwinvaneyk))
- Fix workflow apiserver proxy [\#363](https://github.com/fission/fission/pull/363) ([erwinvaneyk](https://github.com/erwinvaneyk))
- Differentiate by environment in fscache eviction [\#361](https://github.com/fission/fission/pull/361) ([soamvasani](https://github.com/soamvasani))

## [0.3.0-rc](https://github.com/fission/fission/tree/0.3.0-rc) (2017-09-27)
[Full Changelog](https://github.com/fission/fission/compare/buildmgr-preview-20170922...0.3.0-rc)

**Merged pull requests:**

- Dump package resources at the end of tests [\#357](https://github.com/fission/fission/pull/357) ([soamvasani](https://github.com/soamvasani))
- Use Containers to find matched storage containers \(\#350\) [\#353](https://github.com/fission/fission/pull/353) ([life1347](https://github.com/life1347))
- Fix storage service failed to start after restarting it [\#352](https://github.com/fission/fission/pull/352) ([life1347](https://github.com/life1347))
- Add bodyparser for text/plain to node-env [\#349](https://github.com/fission/fission/pull/349) ([erwinvaneyk](https://github.com/erwinvaneyk))
- Fix unsupported checksum type \(issue 342\) [\#343](https://github.com/fission/fission/pull/343) ([life1347](https://github.com/life1347))
- Fission workflow env integration [\#336](https://github.com/fission/fission/pull/336) ([erwinvaneyk](https://github.com/erwinvaneyk))
- Add builder manager support [\#308](https://github.com/fission/fission/pull/308) ([life1347](https://github.com/life1347))

## [buildmgr-preview-20170922](https://github.com/fission/fission/tree/buildmgr-preview-20170922) (2017-09-22)
[Full Changelog](https://github.com/fission/fission/compare/buildmgr-preview-20170921...buildmgr-preview-20170922)

**Merged pull requests:**

- Multiple Trigger Definitions Fix [\#341](https://github.com/fission/fission/pull/341) ([jsturtevant](https://github.com/jsturtevant))

## [buildmgr-preview-20170921](https://github.com/fission/fission/tree/buildmgr-preview-20170921) (2017-09-21)
[Full Changelog](https://github.com/fission/fission/compare/v0.2.1...buildmgr-preview-20170921)

**Merged pull requests:**

- Update a dependency in the package.json [\#339](https://github.com/fission/fission/pull/339) ([watilde](https://github.com/watilde))
- Fission dotnet 2.0 env [\#337](https://github.com/fission/fission/pull/337) ([joalmeid](https://github.com/joalmeid))
- Fix internal route setup bug [\#335](https://github.com/fission/fission/pull/335) ([soamvasani](https://github.com/soamvasani))
- Tag and push the latest environment images [\#333](https://github.com/fission/fission/pull/333) ([y-taka-23](https://github.com/y-taka-23))
- Function service cache partial support for multiple specialization [\#332](https://github.com/fission/fission/pull/332) ([soamvasani](https://github.com/soamvasani))
- Upgrade Node Environment to 8.x [\#329](https://github.com/fission/fission/pull/329) ([MylesBorins](https://github.com/MylesBorins))
- Removed deprecated k8s templates [\#327](https://github.com/fission/fission/pull/327) ([erwinvaneyk](https://github.com/erwinvaneyk))
- Post-install hook to poke analytics function [\#325](https://github.com/fission/fission/pull/325) ([soamvasani](https://github.com/soamvasani))
- update readme with latest install instructions [\#324](https://github.com/fission/fission/pull/324) ([soamvasani](https://github.com/soamvasani))

## [v0.2.1](https://github.com/fission/fission/tree/v0.2.1) (2017-09-12)
[Full Changelog](https://github.com/fission/fission/compare/v0.2.1-rc2...v0.2.1)

**Merged pull requests:**

- Upgrade tool for 0.1 -\> 0.2.1 [\#320](https://github.com/fission/fission/pull/320) ([soamvasani](https://github.com/soamvasani))
- Release automation script -- attach helm charts, tag env images [\#318](https://github.com/fission/fission/pull/318) ([soamvasani](https://github.com/soamvasani))

## [v0.2.1-rc2](https://github.com/fission/fission/tree/v0.2.1-rc2) (2017-09-10)
[Full Changelog](https://github.com/fission/fission/compare/v0.2.1-rc...v0.2.1-rc2)

## [v0.2.1-rc](https://github.com/fission/fission/tree/v0.2.1-rc) (2017-09-09)
[Full Changelog](https://github.com/fission/fission/compare/v0.2.0-20170901...v0.2.1-rc)

**Merged pull requests:**

- Hugo-based documentation site [\#317](https://github.com/fission/fission/pull/317) ([soamvasani](https://github.com/soamvasani))
- Use latest function metadata to check cached function service. [\#316](https://github.com/fission/fission/pull/316) ([life1347](https://github.com/life1347))
- Storage service helm chart integration + bugfixes [\#315](https://github.com/fission/fission/pull/315) ([soamvasani](https://github.com/soamvasani))
- Added perl environment [\#311](https://github.com/fission/fission/pull/311) ([LittleFox94](https://github.com/LittleFox94))
- Move builds to package level [\#297](https://github.com/fission/fission/pull/297) ([soamvasani](https://github.com/soamvasani))

## [v0.2.0-20170901](https://github.com/fission/fission/tree/v0.2.0-20170901) (2017-09-01)
[Full Changelog](https://github.com/fission/fission/compare/nightly20170705...v0.2.0-20170901)

**Merged pull requests:**

- Large functions: API proxy for storage svc, upload support in the CLI [\#304](https://github.com/fission/fission/pull/304) ([soamvasani](https://github.com/soamvasani))
- Unarchive zip file after fetcher downloads the package [\#301](https://github.com/fission/fission/pull/301) ([life1347](https://github.com/life1347))
- Storage service and client [\#300](https://github.com/fission/fission/pull/300) ([soamvasani](https://github.com/soamvasani))
- Add link to the logs section of INSTALL.md [\#299](https://github.com/fission/fission/pull/299) ([ly798](https://github.com/ly798))
- Add Environment v2 Builder [\#298](https://github.com/fission/fission/pull/298) ([life1347](https://github.com/life1347))
- Add env builder & srcpkg through cli [\#296](https://github.com/fission/fission/pull/296) ([life1347](https://github.com/life1347))
- Split out the Package type into a first class Kubernetes resource [\#295](https://github.com/fission/fission/pull/295) ([soamvasani](https://github.com/soamvasani))
- Helm chart bugfixes + end to end test bugfixes [\#293](https://github.com/fission/fission/pull/293) ([soamvasani](https://github.com/soamvasani))
- Minor documentation fix for the Go example [\#292](https://github.com/fission/fission/pull/292) ([georgebuckerfield](https://github.com/georgebuckerfield))
- Improve error message if an older CLI attempts to make a request [\#291](https://github.com/fission/fission/pull/291) ([rapitable](https://github.com/rapitable))
- Update list of environments currently in README [\#289](https://github.com/fission/fission/pull/289) ([erwinvaneyk](https://github.com/erwinvaneyk))
- Fix fetcher failed to access TPR if RBAC is enabled [\#288](https://github.com/fission/fission/pull/288) ([life1347](https://github.com/life1347))
- Fix bug that causes us to skip our new e2e tests [\#285](https://github.com/fission/fission/pull/285) ([soamvasani](https://github.com/soamvasani))
- Parse metadata.Name before creating tpr resource [\#284](https://github.com/fission/fission/pull/284) ([life1347](https://github.com/life1347))
- Remove etcd deployment & svc [\#282](https://github.com/fission/fission/pull/282) ([life1347](https://github.com/life1347))
- End to end test runner [\#281](https://github.com/fission/fission/pull/281) ([soamvasani](https://github.com/soamvasani))
- Set fetcher image through poolmgr env [\#280](https://github.com/fission/fission/pull/280) ([life1347](https://github.com/life1347))
- Set message content-type based on the trigger.Spec.ContentType [\#279](https://github.com/fission/fission/pull/279) ([life1347](https://github.com/life1347))
- Helm chart updates [\#273](https://github.com/fission/fission/pull/273) ([soamvasani](https://github.com/soamvasani))
- Kubernetes access for Travis CI tests [\#272](https://github.com/fission/fission/pull/272) ([soamvasani](https://github.com/soamvasani))
- V2 types and TPR [\#266](https://github.com/fission/fission/pull/266) ([soamvasani](https://github.com/soamvasani))
- Fix logger prints wrong log [\#263](https://github.com/fission/fission/pull/263) ([life1347](https://github.com/life1347))
- Fix nats trigger replies message to non-existing response topic [\#260](https://github.com/fission/fission/pull/260) ([life1347](https://github.com/life1347))
- Binary Environment [\#256](https://github.com/fission/fission/pull/256) ([erwinvaneyk](https://github.com/erwinvaneyk))
- fix typo funtion -\> function [\#252](https://github.com/fission/fission/pull/252) ([sbfaulkner](https://github.com/sbfaulkner))
- Ruby logger [\#251](https://github.com/fission/fission/pull/251) ([sbfaulkner](https://github.com/sbfaulkner))
- Update/Add fission-core & fission-all helm charts [\#239](https://github.com/fission/fission/pull/239) ([life1347](https://github.com/life1347))
- Fix unstoppable kubewatcher [\#208](https://github.com/fission/fission/pull/208) ([life1347](https://github.com/life1347))

## [nightly20170705](https://github.com/fission/fission/tree/nightly20170705) (2017-07-06)
[Full Changelog](https://github.com/fission/fission/compare/nightly20170621...nightly20170705)

**Merged pull requests:**

- include path parameters in params hash for ruby environment [\#249](https://github.com/fission/fission/pull/249) ([sbfaulkner](https://github.com/sbfaulkner))
- Fission update must require at least one change to function [\#241](https://github.com/fission/fission/pull/241) ([life1347](https://github.com/life1347))
- Add message queue trigger support [\#218](https://github.com/fission/fission/pull/218) ([life1347](https://github.com/life1347))

## [nightly20170621](https://github.com/fission/fission/tree/nightly20170621) (2017-06-21)
[Full Changelog](https://github.com/fission/fission/compare/alpha20170124...nightly20170621)

**Merged pull requests:**

- Fix creating redundant pods on heavyload coldstart [\#232](https://github.com/fission/fission/pull/232) ([yqf3139](https://github.com/yqf3139))
- Aggregate tap service request in interval [\#229](https://github.com/fission/fission/pull/229) ([yqf3139](https://github.com/yqf3139))
- Specify full golang version in Dockerfiles and build helper script [\#227](https://github.com/fission/fission/pull/227) ([soamvasani](https://github.com/soamvasani))
- Retrieve URL params in functions \(\#158\) [\#226](https://github.com/fission/fission/pull/226) ([yqf3139](https://github.com/yqf3139))
- Fix s/Sirupsen/sirupsen/ for logrus [\#224](https://github.com/fission/fission/pull/224) ([n1koo](https://github.com/n1koo))
- add ruby-env [\#223](https://github.com/fission/fission/pull/223) ([sbfaulkner](https://github.com/sbfaulkner))
- Fix pool contains wrong environment metadata [\#221](https://github.com/fission/fission/pull/221) ([life1347](https://github.com/life1347))
- Added support for pods and replication controllers to watchers [\#216](https://github.com/fission/fission/pull/216) ([javierbq](https://github.com/javierbq))
- Fix two links in Roadmap doc [\#213](https://github.com/fission/fission/pull/213) ([markpeek](https://github.com/markpeek))
- Fix http response body not closed correctly & return immediately when error occurred [\#210](https://github.com/fission/fission/pull/210) ([life1347](https://github.com/life1347))
- Print log when timetrigger is removed [\#209](https://github.com/fission/fission/pull/209) ([life1347](https://github.com/life1347))
- Retrieve function logs from controller [\#207](https://github.com/fission/fission/pull/207) ([life1347](https://github.com/life1347))
- Adding fission-rbac.yml for  [\#183](https://github.com/fission/fission/pull/183) ([gamefiend](https://github.com/gamefiend))
- Add OpenShift INSTALL.md docs [\#179](https://github.com/fission/fission/pull/179) ([tiny-dancer](https://github.com/tiny-dancer))
- Lighten up the python3 base image \(alpine\) [\#171](https://github.com/fission/fission/pull/171) ([syassami](https://github.com/syassami))
- Make the chart work with helm 2.2 [\#170](https://github.com/fission/fission/pull/170) ([apenney](https://github.com/apenney))
- Add OpenShift support \(\#107\) [\#168](https://github.com/fission/fission/pull/168) ([methadata](https://github.com/methadata))
- Go build helper script [\#163](https://github.com/fission/fission/pull/163) ([soamvasani](https://github.com/soamvasani))
- Add Time Trigger API and client \(\#153\) [\#161](https://github.com/fission/fission/pull/161) ([yqf3139](https://github.com/yqf3139))
- Add fission-ui intro in readme [\#159](https://github.com/fission/fission/pull/159) ([yqf3139](https://github.com/yqf3139))
- Drop Go 1.7, use Go 1.8 [\#157](https://github.com/fission/fission/pull/157) ([soamvasani](https://github.com/soamvasani))
- Add README for Node.js examples [\#155](https://github.com/fission/fission/pull/155) ([RobertHerhold](https://github.com/RobertHerhold))
- Upgrade node environment to Node.js 7.6.0+ [\#151](https://github.com/fission/fission/pull/151) ([RobertHerhold](https://github.com/RobertHerhold))
- use fmt.Errorf instead of error.New\(\) [\#149](https://github.com/fission/fission/pull/149) ([maxwell92](https://github.com/maxwell92))
- Return 201 for created resources [\#148](https://github.com/fission/fission/pull/148) ([RobertHerhold](https://github.com/RobertHerhold))
- Set correct Content-Type in the http response [\#147](https://github.com/fission/fission/pull/147) ([lingxiankong](https://github.com/lingxiankong))
- Make it more clear where to clone this repo [\#145](https://github.com/fission/fission/pull/145) ([RobertHerhold](https://github.com/RobertHerhold))
- Fix function delete with uid [\#142](https://github.com/fission/fission/pull/142) ([yqf3139](https://github.com/yqf3139))
- Fixed pod has no ip \(\#139\) [\#141](https://github.com/fission/fission/pull/141) ([life1347](https://github.com/life1347))
- fix\(kubeEventsSlack\): typo and wrong variable name [\#140](https://github.com/fission/fission/pull/140) ([Pindar](https://github.com/Pindar))
- Ignore the vendor folder [\#137](https://github.com/fission/fission/pull/137) ([RobertHerhold](https://github.com/RobertHerhold))
- Fix Markdown table [\#136](https://github.com/fission/fission/pull/136) ([RobertHerhold](https://github.com/RobertHerhold))
- Symlink user function's node\_modules to server's node\_modules [\#133](https://github.com/fission/fission/pull/133) ([soamvasani](https://github.com/soamvasani))
- Add function logs support \(\#53\) [\#131](https://github.com/fission/fission/pull/131) ([life1347](https://github.com/life1347))
- Remove redundant hello.js from charts directory [\#130](https://github.com/fission/fission/pull/130) ([ssudake21](https://github.com/ssudake21))
- Handle errors in filestore init \(\#108\) [\#127](https://github.com/fission/fission/pull/127) ([soamvasani](https://github.com/soamvasani))
- \[WIP\] Golang runtime [\#125](https://github.com/fission/fission/pull/125) ([nouney](https://github.com/nouney))
- Modify the stock example to show how to change the Content-Type [\#124](https://github.com/fission/fission/pull/124) ([gonrial](https://github.com/gonrial))
- Improve command-line client error output [\#122](https://github.com/fission/fission/pull/122) ([tobias](https://github.com/tobias))
- Report KeyNotFound from etcd as a 404 [\#121](https://github.com/fission/fission/pull/121) ([tobias](https://github.com/tobias))
- Use latest for stable release of minikube [\#120](https://github.com/fission/fission/pull/120) ([r2d4](https://github.com/r2d4))
- Fixed failed to delete function when function's file is not exist [\#118](https://github.com/fission/fission/pull/118) ([life1347](https://github.com/life1347))
- Update gitignore to include dev artifacts [\#117](https://github.com/fission/fission/pull/117) ([tobias](https://github.com/tobias))
- Better convey duplicate name errors to client [\#116](https://github.com/fission/fission/pull/116) ([tobias](https://github.com/tobias))
- Don't wait for ready pod in MakeGenericPool [\#114](https://github.com/fission/fission/pull/114) ([soamvasani](https://github.com/soamvasani))
- Allow unique HTTP route & method \(\#102\) [\#111](https://github.com/fission/fission/pull/111) ([kphatak](https://github.com/kphatak))
- Minor improvements to build instructions in README [\#110](https://github.com/fission/fission/pull/110) ([tobias](https://github.com/tobias))
- Make build an actual sh script [\#109](https://github.com/fission/fission/pull/109) ([tobias](https://github.com/tobias))
- Fixing validations of fn actions [\#106](https://github.com/fission/fission/pull/106) ([kphatak](https://github.com/kphatak))
- Http request support [\#105](https://github.com/fission/fission/pull/105) ([ktrance](https://github.com/ktrance))
- function code download using HTTP URL [\#100](https://github.com/fission/fission/pull/100) ([kphatak](https://github.com/kphatak))
- Error when env name/image not provided [\#98](https://github.com/fission/fission/pull/98) ([lcrisci](https://github.com/lcrisci))
- Add initial support for PHP7 [\#97](https://github.com/fission/fission/pull/97) ([dgoujard](https://github.com/dgoujard))
- kubewatcher example: send watch updates to slack [\#96](https://github.com/fission/fission/pull/96) ([soamvasani](https://github.com/soamvasani))
- bugfix \(cli\) Update the URL check to work with https [\#94](https://github.com/fission/fission/pull/94) ([andrewstuart](https://github.com/andrewstuart))
- Primary Helm chart for fission [\#90](https://github.com/fission/fission/pull/90) ([ssudake21](https://github.com/ssudake21))
- Wait for Pod IP while waiting for pod ready [\#89](https://github.com/fission/fission/pull/89) ([soamvasani](https://github.com/soamvasani))
- Added support for running C\# code in a dotnet core environment [\#84](https://github.com/fission/fission/pull/84) ([ktrance](https://github.com/ktrance))

## [alpha20170124](https://github.com/fission/fission/tree/alpha20170124) (2017-01-24)
[Full Changelog](https://github.com/fission/fission/compare/kubecon...alpha20170124)

**Merged pull requests:**

- Make go vet happy [\#87](https://github.com/fission/fission/pull/87) ([AlekSi](https://github.com/AlekSi))
- Ignore glide cache in gofmt check [\#86](https://github.com/fission/fission/pull/86) ([soamvasani](https://github.com/soamvasani))
- Bugfix for internal routes [\#81](https://github.com/fission/fission/pull/81) ([soamvasani](https://github.com/soamvasani))
- Bug fix for handling a route's HTTP method in router [\#79](https://github.com/fission/fission/pull/79) ([soamvasani](https://github.com/soamvasani))
- fission-bundle: allow setting the namespace [\#77](https://github.com/fission/fission/pull/77) ([frodenas](https://github.com/frodenas))
- Delete generic pools when environments are deleted [\#75](https://github.com/fission/fission/pull/75) ([soamvasani](https://github.com/soamvasani))
- Poolmgr: fix pod leak bugs on specializePod failure [\#70](https://github.com/fission/fission/pull/70) ([soamvasani](https://github.com/soamvasani))
- Poolmgr: ensure orphaned resources are cleaned up [\#69](https://github.com/fission/fission/pull/69) ([soamvasani](https://github.com/soamvasani))
- Implement 'fission route update' [\#68](https://github.com/fission/fission/pull/68) ([soamvasani](https://github.com/soamvasani))
- Update router cache on new function version [\#67](https://github.com/fission/fission/pull/67) ([soamvasani](https://github.com/soamvasani))
- Changed Package Names to Match new Github Organization [\#66](https://github.com/fission/fission/pull/66) ([jgavinray](https://github.com/jgavinray))
- Add HTTP route create params to function create command [\#65](https://github.com/fission/fission/pull/65) ([soamvasani](https://github.com/soamvasani))
- Add kubectl download to install instructions [\#61](https://github.com/fission/fission/pull/61) ([soamvasani](https://github.com/soamvasani))
- Readme minikube instructions [\#60](https://github.com/fission/fission/pull/60) ([soamvasani](https://github.com/soamvasani))
- Check for name in 'function delete' [\#59](https://github.com/fission/fission/pull/59) ([soamvasani](https://github.com/soamvasani))
- adding go-report card and fixing minor typo in README [\#57](https://github.com/fission/fission/pull/57) ([kphatak](https://github.com/kphatak))
- Kubewatcher: trigger functions from Kubernetes Watch callbacks [\#56](https://github.com/fission/fission/pull/56) ([soamvasani](https://github.com/soamvasani))
- adding commonly used python libraries [\#49](https://github.com/fission/fission/pull/49) ([kphatak](https://github.com/kphatak))
- Setup app.logger for python environment [\#48](https://github.com/fission/fission/pull/48) ([soamvasani](https://github.com/soamvasani))
- Add build badge [\#46](https://github.com/fission/fission/pull/46) ([soamvasani](https://github.com/soamvasani))
- Install and run etcd on travis [\#45](https://github.com/fission/fission/pull/45) ([soamvasani](https://github.com/soamvasani))
- Bugfix in functionServiceCache test  [\#44](https://github.com/fission/fission/pull/44) ([soamvasani](https://github.com/soamvasani))
- Fix cache test [\#43](https://github.com/fission/fission/pull/43) ([soamvasani](https://github.com/soamvasani))
- \#25 Continuous Testing [\#42](https://github.com/fission/fission/pull/42) ([jgavinray](https://github.com/jgavinray))
- add travis integration test [\#41](https://github.com/fission/fission/pull/41) ([carmark](https://github.com/carmark))
- Fix `environment` command typo. [\#39](https://github.com/fission/fission/pull/39) ([pirogoeth](https://github.com/pirogoeth))
- Edit readme [\#38](https://github.com/fission/fission/pull/38) ([soamvasani](https://github.com/soamvasani))
- Updated README to include protocol scheme for FISSION\_URL prefix [\#36](https://github.com/fission/fission/pull/36) ([efexen](https://github.com/efexen))
- Add minikube example in readme [\#34](https://github.com/fission/fission/pull/34) ([johscheuer](https://github.com/johscheuer))
- README: use kubectl create -f http [\#32](https://github.com/fission/fission/pull/32) ([philips](https://github.com/philips))
- Python environment improvements [\#30](https://github.com/fission/fission/pull/30) ([soamvasani](https://github.com/soamvasani))
- Readme updates [\#29](https://github.com/fission/fission/pull/29) ([soamvasani](https://github.com/soamvasani))
- Add "fission function edit \<function\>" command [\#28](https://github.com/fission/fission/pull/28) ([soamvasani](https://github.com/soamvasani))
- Move client-go dependency to 1.5 [\#27](https://github.com/fission/fission/pull/27) ([soamvasani](https://github.com/soamvasani))

## [kubecon](https://github.com/fission/fission/tree/kubecon) (2016-11-11)
**Merged pull requests:**

- Reap idle pods [\#20](https://github.com/fission/fission/pull/20) ([soamvasani](https://github.com/soamvasani))
- Fission CLI [\#19](https://github.com/fission/fission/pull/19) ([soamvasani](https://github.com/soamvasani))
- Fix resource store errors on empty db [\#18](https://github.com/fission/fission/pull/18) ([soamvasani](https://github.com/soamvasani))
- fission-bundle: executable package for router, controller, poolmgr [\#17](https://github.com/fission/fission/pull/17) ([soamvasani](https://github.com/soamvasani))
- Router integration with poolmgr and controller [\#16](https://github.com/fission/fission/pull/16) ([soamvasani](https://github.com/soamvasani))
- Poolmgr -- manage generic containers and their specialization [\#15](https://github.com/fission/fission/pull/15) ([soamvasani](https://github.com/soamvasani))
- Fetcher is a helper for function run containers [\#14](https://github.com/fission/fission/pull/14) ([soamvasani](https://github.com/soamvasani))
- Cache -- simple threadsafe map [\#13](https://github.com/fission/fission/pull/13) ([soamvasani](https://github.com/soamvasani))
- Change controller and router exports to make them usable as libraries [\#12](https://github.com/fission/fission/pull/12) ([soamvasani](https://github.com/soamvasani))
- Add API version to URLs [\#11](https://github.com/fission/fission/pull/11) ([soamvasani](https://github.com/soamvasani))
- Nodejs improvements [\#10](https://github.com/fission/fission/pull/10) ([soamvasani](https://github.com/soamvasani))
- Base64 encode the code in json objects. [\#9](https://github.com/fission/fission/pull/9) ([soamvasani](https://github.com/soamvasani))
- API for environments [\#8](https://github.com/fission/fission/pull/8) ([soamvasani](https://github.com/soamvasani))
- Add HTTP trigger API and client [\#7](https://github.com/fission/fission/pull/7) ([soamvasani](https://github.com/soamvasani))
- Move some fission structs to top level package [\#6](https://github.com/fission/fission/pull/6) ([soamvasani](https://github.com/soamvasani))
- Controller [\#5](https://github.com/fission/fission/pull/5) ([soamvasani](https://github.com/soamvasani))
- Move packages to root dir from src/ [\#4](https://github.com/fission/fission/pull/4) ([soamvasani](https://github.com/soamvasani))
- Router [\#3](https://github.com/fission/fission/pull/3) ([soamvasani](https://github.com/soamvasani))
- NodeJS Function Run Container [\#2](https://github.com/fission/fission/pull/2) ([soamvasani](https://github.com/soamvasani))
- Initial docs commit [\#1](https://github.com/fission/fission/pull/1) ([soamvasani](https://github.com/soamvasani))



