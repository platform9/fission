# 1.11.1

Install Guide: docs.fission.io/installation
Release Highlight: docs.fission.io/releases/1.11.1
Full Changelog: /CHANGELOG.md@master

filename | sha256 hash
-------- | -----------
[fission-core-1.11.1-minikube.yaml](https://github.com/fission/fission/releases/download/1.11.1/fission-core-1.11.1-minikube.yaml) | `eea5b535a52b0304aa0fe05d533b83f2a2e0baf6c92c07647b0b29cca6f1933d`
[fission-all-1.11.1-minikube.yaml](https://github.com/fission/fission/releases/download/1.11.1/fission-all-1.11.1-minikube.yaml) | `f2124664c3ffc3b25d393a197e81a05c9f8ec677d73dced9c1f2882601e49916`
[fission-core-1.11.1-openshift.yaml](https://github.com/fission/fission/releases/download/1.11.1/fission-core-1.11.1-openshift.yaml) | `4c9d428cfa25ce2e43e1f3b4f6aab05af2eda45e2bf896262689c1e92bb34e2a`
[fission-core-1.11.1.yaml](https://github.com/fission/fission/releases/download/1.11.1/fission-core-1.11.1.yaml) | `07851e8cfe3a7629cbbce42624af1da82d7ba0882d546586d1501a48084c0d3b`
[fission-all-1.11.1-openshift.yaml](https://github.com/fission/fission/releases/download/1.11.1/fission-all-1.11.1-openshift.yaml) | `873cca25848920cb702e1fc6f5d9234df17389955cce36d67347a78bb79eeccc`
[fission-all-1.11.1.yaml](https://github.com/fission/fission/releases/download/1.11.1/fission-all-1.11.1.yaml) | `122054aa2777467511a3f2bf804bc0bd7d1fed11ebdeaebfb8b73990783e56b8`
[fission-all-1.11.1.tgz](https://github.com/fission/fission/releases/download/1.11.1/fission-all-1.11.1.tgz) | `2f370f40a8799dab8e3ac12b861b18c1eab264c90eb641f9855051bd534f5c29`
[fission-core-1.11.1.tgz](https://github.com/fission/fission/releases/download/1.11.1/fission-core-1.11.1.tgz) | `dc2c6df660974c893433b570e8711c59adf0f7450b4eeef965f44c907f03458c`
[fission-cli-osx](https://github.com/fission/fission/releases/download/1.11.1/fission-cli-osx) | `8a3aa0d4f0f2b564a274e3c96bb0ba41ca4f91481abb906913bf93086d9cd005`
[fission-cli-linux](https://github.com/fission/fission/releases/download/1.11.1/fission-cli-linux) | `3840ed221f14c8f70b20a2b8be2881305423b32d4705cb92349e24d4b627cc9a`
[fission-cli-windows.exe](https://github.com/fission/fission/releases/download/1.11.1/fission-cli-windows.exe) | `3c85b3ec9f0d2d6d4f35a63e1fcea755ed7c0ee66e906c66cb41c99c5bf73140`

# Changelog

## [1.11.1](https://github.com/fission/fission/tree/1.11.1) (2020-10-14)

[Full Changelog](https://github.com/fission/fission/compare/v1.11.0...1.11.1)

**Merged pull requests:**

- Added field to preserve fields during CRD validation [\#1818](https://github.com/fission/fission/pull/1818) ([therahulbhati](https://github.com/therahulbhati))
- fix\(docs\): Update Readme [\#1742](https://github.com/fission/fission/pull/1742) ([iamdarshshah](https://github.com/iamdarshshah))
- Update README.md - fix typo [\#1737](https://github.com/fission/fission/pull/1737) ([Parikshit-Hooda](https://github.com/Parikshit-Hooda))
- \[skip ci\] fix contributing.md link on readme [\#1731](https://github.com/fission/fission/pull/1731) ([mrturkmencom](https://github.com/mrturkmencom))
- Added concurrency field to schema validation [\#1727](https://github.com/fission/fission/pull/1727) ([therahulbhati](https://github.com/therahulbhati))
- Improving contributing docs [\#1726](https://github.com/fission/fission/pull/1726) ([vishal-biyani](https://github.com/vishal-biyani))
- Added code to prevent deletion of active fn pod [\#1724](https://github.com/fission/fission/pull/1724) ([therahulbhati](https://github.com/therahulbhati))
- Release 1.11.0 [\#1716](https://github.com/fission/fission/pull/1716) ([vishal-biyani](https://github.com/vishal-biyani))
- Added flag for insecureSkipVerfiy [\#1829](https://github.com/fission/fission/pull/1829) ([therahulbhati](https://github.com/therahulbhati))
- Update aws-kinesis image name in values.yaml [\#1817](https://github.com/fission/fission/pull/1817) ([girishg4t](https://github.com/girishg4t))
- Improvements from scale testing [\#1812](https://github.com/fission/fission/pull/1812) ([therahulbhati](https://github.com/therahulbhati))
- Update aws-sqs image name in values.yaml [\#1714](https://github.com/fission/fission/pull/1714) ([therahulbhati](https://github.com/therahulbhati))

## [1.11.0](https://github.com/fission/fission/tree/1.11.0) (2020-09-16)

[Full Changelog](https://github.com/fission/fission/compare/1.10.0...1.11.0)

**Merged pull requests:**

- Bumping up release of Helm for release script [\#1715](https://github.com/fission/fission/pull/1715) ([vishal-biyani](https://github.com/vishal-biyani))
- Skaffold: Typo in Helm values for Prometheus [\#1709](https://github.com/fission/fission/pull/1709) ([vishal-biyani](https://github.com/vishal-biyani))
- logs: change timestamp to ISO [\#1708](https://github.com/fission/fission/pull/1708) ([sahil-lakhwani](https://github.com/sahil-lakhwani))
- Bump jetty-server from 9.0.4.v20130625 to 9.4.17.v20190418 in /environments/jvm-jersey [\#1706](https://github.com/fission/fission/pull/1706) ([dependabot[bot]](https://github.com/apps/dependabot))
- Addition of openapischemav3 to fission CRDs to support kubectl explain [\#1702](https://github.com/fission/fission/pull/1702) ([ankitjain235](https://github.com/ankitjain235))
- Skaffold kind [\#1700](https://github.com/fission/fission/pull/1700) ([vishal-biyani](https://github.com/vishal-biyani))
- Adding Concurrency in Pool Manager [\#1698](https://github.com/fission/fission/pull/1698) ([therahulbhati](https://github.com/therahulbhati))
- Adding NodeJS version 12 env [\#1683](https://github.com/fission/fission/pull/1683) ([vishal-biyani](https://github.com/vishal-biyani))
- Removed code related to mqtrigger [\#1680](https://github.com/fission/fission/pull/1680) ([therahulbhati](https://github.com/therahulbhati))
- jvm-jersey-env [\#1677](https://github.com/fission/fission/pull/1677) ([sahil-lakhwani](https://github.com/sahil-lakhwani))
- Update values.yaml with mqt keda configuration [\#1670](https://github.com/fission/fission/pull/1670) ([therahulbhati](https://github.com/therahulbhati))
- Release 1.10.0 [\#1658](https://github.com/fission/fission/pull/1658) ([vishal-biyani](https://github.com/vishal-biyani))
- \(fix\) Dependencies and steps for building examples/java locally [\#1653](https://github.com/fission/fission/pull/1653) ([rahulchheda](https://github.com/rahulchheda))
- Allow user to override nats-streaming image [\#1645](https://github.com/fission/fission/pull/1645) ([funkypenguin](https://github.com/funkypenguin))
- Allow user to define busybox image [\#1643](https://github.com/fission/fission/pull/1643) ([funkypenguin](https://github.com/funkypenguin))
- Add headers to Kafka MQT error topics [\#1701](https://github.com/fission/fission/pull/1701) ([ankitjain235](https://github.com/ankitjain235))
- Non-root users access Secrets and ConfigMaps  [\#1697](https://github.com/fission/fission/pull/1697) ([atsai1220](https://github.com/atsai1220))
- Added prefix tag in fluentbit filter stanza [\#1671](https://github.com/fission/fission/pull/1671) ([therahulbhati](https://github.com/therahulbhati))
- 1665: Python env sentry integration [\#1669](https://github.com/fission/fission/pull/1669) ([vir-mir](https://github.com/vir-mir))
- Fission MQT integration with keda [\#1657](https://github.com/fission/fission/pull/1657) ([therahulbhati](https://github.com/therahulbhati))
- Add serviceaccount for nats-streaming [\#1646](https://github.com/fission/fission/pull/1646) ([funkypenguin](https://github.com/funkypenguin))
- Allow user to override value of influxdb image [\#1642](https://github.com/fission/fission/pull/1642) ([funkypenguin](https://github.com/funkypenguin))

## [v1.10.0](https://github.com/fission/fission/tree/v1.10.0) (2020-06-29)

[Full Changelog](https://github.com/fission/fission/compare/v1.9.0...v1.10.0)

**Merged pull requests:**

- Make mergePodSpec pick up enableServiceLinks [\#1601](https://github.com/fission/fission/pull/1601) ([darkworon](https://github.com/darkworon))
- Bump rack from 2.0.8 to 2.1.4 in /environments/ruby [\#1654](https://github.com/fission/fission/pull/1654) ([dependabot[bot]](https://github.com/apps/dependabot))
- For fixing staticcheck issue [\#1652](https://github.com/fission/fission/pull/1652) ([vishal-biyani](https://github.com/vishal-biyani))
- Python env changes for pip3 [\#1633](https://github.com/fission/fission/pull/1633) ([agiwalpooja20](https://github.com/agiwalpooja20))
- S3 backend for storage service [\#1629](https://github.com/fission/fission/pull/1629) ([vishal-biyani](https://github.com/vishal-biyani))
- Fixing verify-staticcheck.sh [\#1622](https://github.com/fission/fission/pull/1622) ([rajalokan](https://github.com/rajalokan))
- Added support for setting bodyParser limit param via environment variable [\#1618](https://github.com/fission/fission/pull/1618) ([therahulbhati](https://github.com/therahulbhati))
- Skaffold registry [\#1617](https://github.com/fission/fission/pull/1617) ([vishal-biyani](https://github.com/vishal-biyani))
- \[WIP\] \(feat\) Added Exponential BackOff for Retry in builder [\#1614](https://github.com/fission/fission/pull/1614) ([rahulchheda](https://github.com/rahulchheda))
- Added codecov badge [\#1604](https://github.com/fission/fission/pull/1604) ([vishal-biyani](https://github.com/vishal-biyani))
- Update issue templates [\#1602](https://github.com/fission/fission/pull/1602) ([vishal-biyani](https://github.com/vishal-biyani))
- Release 1.9.0 [\#1597](https://github.com/fission/fission/pull/1597) ([vishal-biyani](https://github.com/vishal-biyani))
- Added support for kube-context flag, to specify kubernetes context [\#1595](https://github.com/fission/fission/pull/1595) ([therahulbhati](https://github.com/therahulbhati))

## [1.9.0](https://github.com/fission/fission/tree/1.9.0) (2020-05-10)

[Full Changelog](https://github.com/fission/fission/compare/1.8.0...1.9.0)

**Merged pull requests:**

- Kind config  [\#1587](https://github.com/fission/fission/pull/1587) ([vishal-biyani](https://github.com/vishal-biyani))
- Show short flag on CLI usage [\#1580](https://github.com/fission/fission/pull/1580) ([life1347](https://github.com/life1347))
- External nats streaming [\#1576](https://github.com/fission/fission/pull/1576) ([vishal-biyani](https://github.com/vishal-biyani))
- Skaffold Default Repo [\#1575](https://github.com/fission/fission/pull/1575) ([vishal-biyani](https://github.com/vishal-biyani))
- Python Env Build issue due to gevent [\#1572](https://github.com/fission/fission/pull/1572) ([vishal-biyani](https://github.com/vishal-biyani))
- create fission environment for go version 1.14 [\#1570](https://github.com/fission/fission/pull/1570) ([Jared-Prime](https://github.com/Jared-Prime))
- \[chart\] Add PSP for logger [\#1568](https://github.com/fission/fission/pull/1568) ([LittleFox94](https://github.com/LittleFox94))
- \[WIP\] Skaffold Fix  [\#1567](https://github.com/fission/fission/pull/1567) ([vishal-biyani](https://github.com/vishal-biyani))
- Adding community meeting link and document [\#1563](https://github.com/fission/fission/pull/1563) ([vishal-biyani](https://github.com/vishal-biyani))
- Return Kubernetes full error message [\#1560](https://github.com/fission/fission/pull/1560) ([life1347](https://github.com/life1347))
- Use stock InfluxDB image [\#1557](https://github.com/fission/fission/pull/1557) ([life1347](https://github.com/life1347))
- Bump Python environment to latest Alpine [\#1547](https://github.com/fission/fission/pull/1547) ([delucca](https://github.com/delucca))
- Bump nokogiri from 1.10.4 to 1.10.8 in /examples/ruby/parse [\#1544](https://github.com/fission/fission/pull/1544) ([dependabot[bot]](https://github.com/apps/dependabot))
- Avoid exposing sensitive data to client [\#1543](https://github.com/fission/fission/pull/1543) ([life1347](https://github.com/life1347))
- Retry querying package info when "not found" [\#1540](https://github.com/fission/fission/pull/1540) ([life1347](https://github.com/life1347))
- Fix function test timeout doesnt works [\#1539](https://github.com/fission/fission/pull/1539) ([life1347](https://github.com/life1347))
- Support Function-level idle timeout setting [\#1538](https://github.com/fission/fission/pull/1538) ([life1347](https://github.com/life1347))
- Add message queue service factory [\#1537](https://github.com/fission/fission/pull/1537) ([life1347](https://github.com/life1347))
- Update NATS-Streaming dependencies version [\#1533](https://github.com/fission/fission/pull/1533) ([life1347](https://github.com/life1347))
- Fix Git issue on case-insensitive OS [\#1532](https://github.com/fission/fission/pull/1532) ([life1347](https://github.com/life1347))
- Reorganize message queue trigger directory structure [\#1531](https://github.com/fission/fission/pull/1531) ([life1347](https://github.com/life1347))
- Append Environment labels to function pod labels [\#1530](https://github.com/fission/fission/pull/1530) ([life1347](https://github.com/life1347))
- Place package deploy archive to fix path [\#1529](https://github.com/fission/fission/pull/1529) ([life1347](https://github.com/life1347))
- Fix github\_changelog\_generator error [\#1527](https://github.com/fission/fission/pull/1527) ([life1347](https://github.com/life1347))
- Release 1.8.0 [\#1526](https://github.com/fission/fission/pull/1526) ([life1347](https://github.com/life1347))

## [v1.8.0](https://github.com/fission/fission/tree/v1.8.0) (2020-02-03)

[Full Changelog](https://github.com/fission/fission/compare/v1.7.1...v1.8.0)

**Merged pull requests:**

- Revert "Temporarily disable building JVM image during CI builds" [\#1525](https://github.com/fission/fission/pull/1525) ([life1347](https://github.com/life1347))
- Update TerminationGracePeriod usage text [\#1524](https://github.com/fission/fission/pull/1524) ([life1347](https://github.com/life1347))
- Set initial package status if its empty [\#1522](https://github.com/fission/fission/pull/1522) ([life1347](https://github.com/life1347))
- Update slack invitation link [\#1521](https://github.com/fission/fission/pull/1521) ([life1347](https://github.com/life1347))
- Fix Java spec example [\#1519](https://github.com/fission/fission/pull/1519) ([life1347](https://github.com/life1347))
- Fix executor wrongly deletes rolebindings [\#1517](https://github.com/fission/fission/pull/1517) ([life1347](https://github.com/life1347))
- Show global options in usage [\#1516](https://github.com/fission/fission/pull/1516) ([life1347](https://github.com/life1347))
- Add fake client for local command operation [\#1515](https://github.com/fission/fission/pull/1515) ([life1347](https://github.com/life1347))
- Set prometheus.enabled=false in core chart [\#1513](https://github.com/fission/fission/pull/1513) ([life1347](https://github.com/life1347))
- Java env: Add XML dependency [\#1512](https://github.com/fission/fission/pull/1512) ([sahil-lakhwani](https://github.com/sahil-lakhwani))
- Fix poolmanager wrongly delete env pool [\#1511](https://github.com/fission/fission/pull/1511) ([life1347](https://github.com/life1347))
- Use patch for robust pod metadata update in poolmgr [\#1509](https://github.com/fission/fission/pull/1509) ([life1347](https://github.com/life1347))
- Optimize CI build steps to reduce waiting time [\#1508](https://github.com/fission/fission/pull/1508) ([life1347](https://github.com/life1347))
- Add Go 1.13 support and upgrade Go 1.12 version [\#1507](https://github.com/fission/fission/pull/1507) ([life1347](https://github.com/life1347))
- Add resource exists error on spec validate [\#1506](https://github.com/fission/fission/pull/1506) ([life1347](https://github.com/life1347))
- Add --dry option to view the generated spec without saving [\#1504](https://github.com/fission/fission/pull/1504) ([life1347](https://github.com/life1347))
- Let unit tests run in different namespaces to avoid resource conflict [\#1502](https://github.com/fission/fission/pull/1502) ([life1347](https://github.com/life1347))
- Move to new CI cluster [\#1500](https://github.com/fission/fission/pull/1500) ([life1347](https://github.com/life1347))
- Add sponsor logos [\#1499](https://github.com/fission/fission/pull/1499) ([life1347](https://github.com/life1347))
- Follow kubernetes APIs directory structure [\#1497](https://github.com/fission/fission/pull/1497) ([life1347](https://github.com/life1347))
- Codebase cleanup & optimization [\#1493](https://github.com/fission/fission/pull/1493) ([life1347](https://github.com/life1347))
- Use code-generator to generate clientset/informer/lister [\#1492](https://github.com/fission/fission/pull/1492) ([life1347](https://github.com/life1347))
- Update analytics URL [\#1490](https://github.com/fission/fission/pull/1490) ([life1347](https://github.com/life1347))
- Add config for fetcher resource requests&limits [\#1489](https://github.com/fission/fission/pull/1489) ([life1347](https://github.com/life1347))
- Push php-builder to dockerhub [\#1477](https://github.com/fission/fission/pull/1477) ([life1347](https://github.com/life1347))
- Fix terminationGracePeriod is 0 due to wrong flag type [\#1476](https://github.com/fission/fission/pull/1476) ([life1347](https://github.com/life1347))
- Fix hello-spec-example [\#1474](https://github.com/fission/fission/pull/1474) ([sahil-lakhwani](https://github.com/sahil-lakhwani))
- Add message queue nats-streaming example [\#1472](https://github.com/fission/fission/pull/1472) ([life1347](https://github.com/life1347))
- Add failed/success output to spec validate [\#1471](https://github.com/fission/fission/pull/1471) ([anubhav6663](https://github.com/anubhav6663))
- Bump rack from 2.0.6 to 2.0.8 in /environments/ruby [\#1470](https://github.com/fission/fission/pull/1470) ([dependabot[bot]](https://github.com/apps/dependabot))
- Fix go-server failed to load plugin [\#1469](https://github.com/fission/fission/pull/1469) ([life1347](https://github.com/life1347))
- Add spec list feature [\#1468](https://github.com/fission/fission/pull/1468) ([anubhav6663](https://github.com/anubhav6663))
- Add controller API client interface [\#1467](https://github.com/fission/fission/pull/1467) ([life1347](https://github.com/life1347))
- Remove FISSION\_ROUTER for fn test [\#1465](https://github.com/fission/fission/pull/1465) ([sahil-lakhwani](https://github.com/sahil-lakhwani))
- Add the not present cmname while fn create in err message \[CLI-UX\] [\#1462](https://github.com/fission/fission/pull/1462) ([viveksinghggits](https://github.com/viveksinghggits))
- Fission 1.7.1 [\#1461](https://github.com/fission/fission/pull/1461) ([life1347](https://github.com/life1347))

## [1.7.1](https://github.com/fission/fission/tree/1.7.1) (2019-12-09)

[Full Changelog](https://github.com/fission/fission/compare/1.7.0...1.7.1)

**Merged pull requests:**

- Fix name conflict when buildermanager merges podspec [\#1460](https://github.com/fission/fission/pull/1460) ([life1347](https://github.com/life1347))
- Not to exclude hidden file when creating archive [\#1458](https://github.com/fission/fission/pull/1458) ([life1347](https://github.com/life1347))
- Fission 1.7.0 [\#1457](https://github.com/fission/fission/pull/1457) ([life1347](https://github.com/life1347))

## [v1.7.0](https://github.com/fission/fission/tree/v1.7.0) (2019-12-02)

[Full Changelog](https://github.com/fission/fission/compare/v1.7.0-rc.2...v1.7.0)

**Merged pull requests:**

- Fix release script not uploads OpenShift deploy YAML file [\#1456](https://github.com/fission/fission/pull/1456) ([life1347](https://github.com/life1347))
- Let executor type manages how to do cleanup for old kubeobjects [\#1455](https://github.com/fission/fission/pull/1455) ([life1347](https://github.com/life1347))
- Prevent deployment from rolling update due to different instance-id [\#1454](https://github.com/fission/fission/pull/1454) ([life1347](https://github.com/life1347))
- Make AdoptExistingResources optional [\#1453](https://github.com/fission/fission/pull/1453) ([life1347](https://github.com/life1347))
- Prevent newdeploy updates deployment if no resources changed [\#1452](https://github.com/fission/fission/pull/1452) ([life1347](https://github.com/life1347))
- Fix CLI unable to get pod logs from controller [\#1451](https://github.com/fission/fission/pull/1451) ([life1347](https://github.com/life1347))
- Ignore hidden file when creating archive file [\#1450](https://github.com/fission/fission/pull/1450) ([life1347](https://github.com/life1347))
- Fix spec init overrides existing deploymentconfig [\#1449](https://github.com/fission/fission/pull/1449) ([life1347](https://github.com/life1347))
- Fix spec shows source archive is not used [\#1448](https://github.com/fission/fission/pull/1448) ([life1347](https://github.com/life1347))
- Fix adopted deployment uses old fetcher image [\#1447](https://github.com/fission/fission/pull/1447) ([life1347](https://github.com/life1347))
- Improve executor bootstrap speed [\#1446](https://github.com/fission/fission/pull/1446) ([life1347](https://github.com/life1347))
- Fission 1.7.0-rc.2 [\#1445](https://github.com/fission/fission/pull/1445) ([life1347](https://github.com/life1347))

## [1.7.0-rc.2](https://github.com/fission/fission/tree/1.7.0-rc.2) (2019-11-27)

[Full Changelog](https://github.com/fission/fission/compare/1.7.0-rc.1...1.7.0-rc.2)

**Merged pull requests:**

- Push extra tag to fit go module semver tag format [\#1444](https://github.com/fission/fission/pull/1444) ([life1347](https://github.com/life1347))
- Adopt existing orphan kubernetes resources when executor starts up [\#1443](https://github.com/fission/fission/pull/1443) ([life1347](https://github.com/life1347))
- Revert "Try to fix flaky canary test" [\#1442](https://github.com/fission/fission/pull/1442) ([life1347](https://github.com/life1347))
- Try to fix flaky canary test [\#1441](https://github.com/fission/fission/pull/1441) ([life1347](https://github.com/life1347))
- Fix router tries to update ingress when createIngress is false [\#1440](https://github.com/fission/fission/pull/1440) ([life1347](https://github.com/life1347))
- Fix poolmanager sets 0 timeout for function specialization [\#1439](https://github.com/fission/fission/pull/1439) ([life1347](https://github.com/life1347))
- Add huge response body test [\#1437](https://github.com/fission/fission/pull/1437) ([life1347](https://github.com/life1347))
- Return error when specialization failed [\#1436](https://github.com/fission/fission/pull/1436) ([life1347](https://github.com/life1347))
-  Fix poolmanager terminates running function pod periodically [\#1435](https://github.com/fission/fission/pull/1435) ([life1347](https://github.com/life1347))
- Allow to tap multiple function services at one time [\#1434](https://github.com/fission/fission/pull/1434) ([life1347](https://github.com/life1347))
- Collect function metrics after finishing request [\#1433](https://github.com/fission/fission/pull/1433) ([life1347](https://github.com/life1347))
- Fix poolmanager crashes when failed to list environment [\#1432](https://github.com/fission/fission/pull/1432) ([life1347](https://github.com/life1347))
- Ability to pull builder image from private registry [\#1431](https://github.com/fission/fission/pull/1431) ([life1347](https://github.com/life1347))
- Add checksum and insecure flag for user to skip checksum generation [\#1430](https://github.com/fission/fission/pull/1430) ([life1347](https://github.com/life1347))
- Support to set imagePullSecret when creating environment [\#1429](https://github.com/fission/fission/pull/1429) ([life1347](https://github.com/life1347))
- Fix no kubeobjs get created if fn created before env creation [\#1428](https://github.com/fission/fission/pull/1428) ([life1347](https://github.com/life1347))
- Fix verbosity flag not found in subcommand [\#1425](https://github.com/fission/fission/pull/1425) ([life1347](https://github.com/life1347))
- Improve compatibility with Openshift [\#1424](https://github.com/fission/fission/pull/1424) ([life1347](https://github.com/life1347))
- Fix truncated body returned from router [\#1420](https://github.com/fission/fission/pull/1420) ([life1347](https://github.com/life1347))
- Fission 1.7.0-rc.1 [\#1419](https://github.com/fission/fission/pull/1419) ([life1347](https://github.com/life1347))

## [v1.7.0-rc.1](https://github.com/fission/fission/tree/v1.7.0-rc.1) (2019-11-18)

[Full Changelog](https://github.com/fission/fission/compare/v1.6.0...v1.7.0-rc.1)

**Merged pull requests:**

- Update release-builder go and helm version [\#1418](https://github.com/fission/fission/pull/1418) ([life1347](https://github.com/life1347))
- Fix failed to find init\_tools.sh [\#1417](https://github.com/fission/fission/pull/1417) ([life1347](https://github.com/life1347))
- Support semantic version tags [\#1416](https://github.com/fission/fission/pull/1416) ([life1347](https://github.com/life1347))
- Show warning when referencing nonexistent resources in spec [\#1415](https://github.com/fission/fission/pull/1415) ([life1347](https://github.com/life1347))
- Add resource info to output message when saving spec file [\#1414](https://github.com/fission/fission/pull/1414) ([life1347](https://github.com/life1347))
- Always embed given URL in the archive [\#1413](https://github.com/fission/fission/pull/1413) ([life1347](https://github.com/life1347))
- Fix wrong command and flag usage description [\#1412](https://github.com/fission/fission/pull/1412) ([life1347](https://github.com/life1347))
- Add --spec to package command [\#1411](https://github.com/fission/fission/pull/1411) ([life1347](https://github.com/life1347))
- Drop unreleased features \(record & replay\) [\#1406](https://github.com/fission/fission/pull/1406) ([life1347](https://github.com/life1347))
- Build error formatting on fission spec apply --wait [\#1403](https://github.com/fission/fission/pull/1403) ([life1347](https://github.com/life1347))
- Refactor controller client package [\#1402](https://github.com/fission/fission/pull/1402) ([life1347](https://github.com/life1347))
- Fix fn test failed to query logs from log database [\#1401](https://github.com/fission/fission/pull/1401) ([life1347](https://github.com/life1347))
- Skip trace for router healthz endpoint [\#1400](https://github.com/fission/fission/pull/1400) ([life1347](https://github.com/life1347))
- Set jaeger collector endpoint as an environment variable [\#1399](https://github.com/fission/fission/pull/1399) ([life1347](https://github.com/life1347))
- Replace deprecated serviceAccount with serviceAccountName [\#1398](https://github.com/fission/fission/pull/1398) ([life1347](https://github.com/life1347))
- Fix helm pre-upgrade check failure problem [\#1397](https://github.com/fission/fission/pull/1397) ([life1347](https://github.com/life1347))
- Prettify console output message [\#1396](https://github.com/fission/fission/pull/1396) ([life1347](https://github.com/life1347))
- fix typo [\#1395](https://github.com/fission/fission/pull/1395) ([jjmengze](https://github.com/jjmengze))
- Reorder command flags and add missing flags [\#1394](https://github.com/fission/fission/pull/1394) ([life1347](https://github.com/life1347))
- Fix CLI exits with status 0 when error occurs [\#1393](https://github.com/fission/fission/pull/1393) ([life1347](https://github.com/life1347))
- Poolmanager wait for function pod till specialization timeout [\#1392](https://github.com/fission/fission/pull/1392) ([life1347](https://github.com/life1347))
- Replace flag text with const [\#1391](https://github.com/fission/fission/pull/1391) ([life1347](https://github.com/life1347))
- Update prometheus version and disable it from the default installation [\#1389](https://github.com/fission/fission/pull/1389) ([life1347](https://github.com/life1347))
- Fix helm shows "Not a table" issue when install Fission [\#1387](https://github.com/fission/fission/pull/1387) ([life1347](https://github.com/life1347))
- Fix githook not aborting push if error occurs [\#1386](https://github.com/fission/fission/pull/1386) ([life1347](https://github.com/life1347))
- Migrate from urfave/cli to cobra [\#1385](https://github.com/fission/fission/pull/1385) ([life1347](https://github.com/life1347))
- Update maintainer info [\#1383](https://github.com/fission/fission/pull/1383) ([life1347](https://github.com/life1347))
- Update Makefile and add git pre-push hook [\#1382](https://github.com/fission/fission/pull/1382) ([life1347](https://github.com/life1347))
- Update staticcheck version and fix all warnings [\#1381](https://github.com/fission/fission/pull/1381) ([life1347](https://github.com/life1347))
- Make CLI functions return error instead of fatal out [\#1379](https://github.com/fission/fission/pull/1379) ([life1347](https://github.com/life1347))
- Refactor record command [\#1378](https://github.com/fission/fission/pull/1378) ([life1347](https://github.com/life1347))
- Fix reverse proxy shows 404 not found when Istio enabled [\#1377](https://github.com/fission/fission/pull/1377) ([life1347](https://github.com/life1347))
- Refactor time trigger command [\#1376](https://github.com/fission/fission/pull/1376) ([life1347](https://github.com/life1347))
- Refactor mqtrigger command [\#1375](https://github.com/fission/fission/pull/1375) ([life1347](https://github.com/life1347))
- added example for builder podspec [\#1374](https://github.com/fission/fission/pull/1374) ([viveksinghggits](https://github.com/viveksinghggits))
- Refactor function command [\#1372](https://github.com/fission/fission/pull/1372) ([life1347](https://github.com/life1347))
- Allow to set API type for tensorflow serving environment [\#1371](https://github.com/fission/fission/pull/1371) ([life1347](https://github.com/life1347))
- Refactor canary config command [\#1370](https://github.com/fission/fission/pull/1370) ([life1347](https://github.com/life1347))
- PodSpec support in environment builder [\#1369](https://github.com/fission/fission/pull/1369) ([viveksinghggits](https://github.com/viveksinghggits))
- Fix utility function uses the wrong flag text to get value [\#1368](https://github.com/fission/fission/pull/1368) ([life1347](https://github.com/life1347))
- Refactor HTTP trigger command [\#1367](https://github.com/fission/fission/pull/1367) ([life1347](https://github.com/life1347))
- Update READEME link and add back the basic usage [\#1366](https://github.com/fission/fission/pull/1366) ([life1347](https://github.com/life1347))
- Refactor kubewatch command [\#1365](https://github.com/fission/fission/pull/1365) ([life1347](https://github.com/life1347))
- Fix accidentally removed timestamp when listing package [\#1364](https://github.com/fission/fission/pull/1364) ([life1347](https://github.com/life1347))
- Move route creation to function [\#1362](https://github.com/fission/fission/pull/1362) ([life1347](https://github.com/life1347))
- Remove UID from CLI output [\#1361](https://github.com/fission/fission/pull/1361) ([life1347](https://github.com/life1347))
- Allow using URL as archive source when creating functions [\#1360](https://github.com/fission/fission/pull/1360) ([life1347](https://github.com/life1347))
- Refactor plugin & version subcommands [\#1359](https://github.com/fission/fission/pull/1359) ([life1347](https://github.com/life1347))
- Provide secrets and configmaps while updating the functions [\#1358](https://github.com/fission/fission/pull/1358) ([viveksinghggits](https://github.com/viveksinghggits))
- Update fission architecture doc [\#1356](https://github.com/fission/fission/pull/1356) ([life1347](https://github.com/life1347))
- calling the function that handles kafka messages, asynchronously [\#1355](https://github.com/fission/fission/pull/1355) ([viveksinghggits](https://github.com/viveksinghggits))
- Fix release script tags wrong image name & sed problem on Linux [\#1353](https://github.com/fission/fission/pull/1353) ([life1347](https://github.com/life1347))
- Update CHANGELOG for 1.6.0 [\#1352](https://github.com/fission/fission/pull/1352) ([life1347](https://github.com/life1347))
- Replace `AlwaysSample` with `ProbabilitySampler` in router \(\#1215\) [\#1348](https://github.com/fission/fission/pull/1348) ([ccamel](https://github.com/ccamel))
- Refactor package CLI command [\#1345](https://github.com/fission/fission/pull/1345) ([life1347](https://github.com/life1347))

## [1.6.0](https://github.com/fission/fission/tree/1.6.0) (2019-10-10)

[Full Changelog](https://github.com/fission/fission/compare/1.5.0...1.6.0)

**Merged pull requests:**

- Fission 1.6.0 [\#1351](https://github.com/fission/fission/pull/1351) ([life1347](https://github.com/life1347))
- Move statefulset to apps/v1 [\#1350](https://github.com/fission/fission/pull/1350) ([life1347](https://github.com/life1347))
- Fix newdeploy failed to find serviceEntry in cache [\#1349](https://github.com/fission/fission/pull/1349) ([life1347](https://github.com/life1347))
- Support encoded path in router [\#1347](https://github.com/fission/fission/pull/1347) ([life1347](https://github.com/life1347))
- Support custom private go vendor [\#1346](https://github.com/fission/fission/pull/1346) ([life1347](https://github.com/life1347))
- Fix the namespace mismatch problem when deploying with a single YAML file [\#1344](https://github.com/fission/fission/pull/1344) ([life1347](https://github.com/life1347))
- Move charts executor service to the right place [\#1343](https://github.com/fission/fission/pull/1343) ([life1347](https://github.com/life1347))
- Allow to deploy router as DaemonSet [\#1342](https://github.com/fission/fission/pull/1342) ([life1347](https://github.com/life1347))
- add package filter feature [\#1341](https://github.com/fission/fission/pull/1341) ([jjmengze](https://github.com/jjmengze))
- Bump jvm builder JDK version [\#1340](https://github.com/fission/fission/pull/1340) ([life1347](https://github.com/life1347))
- Fix executor doesn't apply user-configured container spec correctly [\#1339](https://github.com/fission/fission/pull/1339) ([life1347](https://github.com/life1347))
- Allow to add annotations to router service in helm chart [\#1338](https://github.com/fission/fission/pull/1338) ([prabhu43](https://github.com/prabhu43))
- Remove too verbose and unhelpful debug log [\#1336](https://github.com/fission/fission/pull/1336) ([life1347](https://github.com/life1347))
- package will now be listed, sorted by lastupdatedtime [\#1334](https://github.com/fission/fission/pull/1334) ([viveksinghggits](https://github.com/viveksinghggits))
- Fix router health route not found if no HTTP triggers created [\#1333](https://github.com/fission/fission/pull/1333) ([life1347](https://github.com/life1347))
- Fix spec doesn't update status of failed package when applying spec files [\#1332](https://github.com/fission/fission/pull/1332) ([life1347](https://github.com/life1347))
- Move Deployment API group from extensions/v1beta1 to apps/v1 [\#1331](https://github.com/fission/fission/pull/1331) ([life1347](https://github.com/life1347))
- Fix empty host value when list http triggers [\#1328](https://github.com/fission/fission/pull/1328) ([life1347](https://github.com/life1347))
- List/Delete HTTP triggers by function [\#1327](https://github.com/fission/fission/pull/1327) ([life1347](https://github.com/life1347))
- Add Ingress TLS support [\#1326](https://github.com/fission/fission/pull/1326) ([life1347](https://github.com/life1347))
- Add Ingress host, path and annotations support [\#1325](https://github.com/fission/fission/pull/1325) ([life1347](https://github.com/life1347))
- Set maxSurge to 20% for safe rolling upgrade [\#1321](https://github.com/fission/fission/pull/1321) ([life1347](https://github.com/life1347))
- Disable revision history for newdeploy [\#1319](https://github.com/fission/fission/pull/1319) ([life1347](https://github.com/life1347))
- Pre-built docker go module dependency cache image [\#1318](https://github.com/fission/fission/pull/1318) ([life1347](https://github.com/life1347))
- Fix newdeploy doesn't handle error properly [\#1316](https://github.com/fission/fission/pull/1316) ([life1347](https://github.com/life1347))
- Fix failed to update specialization timeout [\#1315](https://github.com/fission/fission/pull/1315) ([life1347](https://github.com/life1347))
- Add codecov config file [\#1314](https://github.com/fission/fission/pull/1314) ([life1347](https://github.com/life1347))
- Fix potential nil pointer problem when using multierror pkg [\#1311](https://github.com/fission/fission/pull/1311) ([life1347](https://github.com/life1347))
- Use ErrorHandler to handle proxy error [\#1310](https://github.com/fission/fission/pull/1310) ([life1347](https://github.com/life1347))
- Integrate with Codecov [\#1308](https://github.com/fission/fission/pull/1308) ([life1347](https://github.com/life1347))
- Fix executor unable to list secrets/configmaps [\#1307](https://github.com/fission/fission/pull/1307) ([life1347](https://github.com/life1347))
- Add specialization timeout default value to CLI [\#1305](https://github.com/fission/fission/pull/1305) ([life1347](https://github.com/life1347))
- Update CHANGELOG for 1.5.0 [\#1304](https://github.com/fission/fission/pull/1304) ([life1347](https://github.com/life1347))
- Fission 1.5.0 [\#1303](https://github.com/fission/fission/pull/1303) ([life1347](https://github.com/life1347))
- Implement TLS authentication for kafka mqt [\#1300](https://github.com/fission/fission/pull/1300) ([vadasambar](https://github.com/vadasambar))
-  Issue 1229: Function level timeout [\#1284](https://github.com/fission/fission/pull/1284) ([parauliya](https://github.com/parauliya))
- Fix kafka producer and consumer logs show empty objects [\#1281](https://github.com/fission/fission/pull/1281) ([vadasambar](https://github.com/vadasambar))

## [1.5.0](https://github.com/fission/fission/tree/1.5.0) (2019-09-09)

[Full Changelog](https://github.com/fission/fission/compare/1.4.1...1.5.0)

**Merged pull requests:**

- Refactor support dump CLI [\#1301](https://github.com/fission/fission/pull/1301) ([life1347](https://github.com/life1347))
- \[fission-cli\]\[feature\] support reverse query function for query log [\#1298](https://github.com/fission/fission/pull/1298) ([moluzhang](https://github.com/moluzhang))
- Fix unit tests failure [\#1297](https://github.com/fission/fission/pull/1297) ([life1347](https://github.com/life1347))
- Support nodejs function with 0 argument [\#1295](https://github.com/fission/fission/pull/1295) ([life1347](https://github.com/life1347))
- Update go environment errors for specializeHandlerV2 [\#1294](https://github.com/fission/fission/pull/1294) ([e-nikolov](https://github.com/e-nikolov))
- Add troubleshooting guide link [\#1292](https://github.com/fission/fission/pull/1292) ([life1347](https://github.com/life1347))
- Bump nokogiri from 1.10.1 to 1.10.4 in /examples/ruby/parse [\#1291](https://github.com/fission/fission/pull/1291) ([dependabot[bot]](https://github.com/apps/dependabot))
- Bump Go version to 1.12 [\#1290](https://github.com/fission/fission/pull/1290) ([life1347](https://github.com/life1347))
- Fix typo extraCoreComponmentPodConfig -\> extraCoreComponentPodConfig [\#1287](https://github.com/fission/fission/pull/1287) ([life1347](https://github.com/life1347))
- Improve CI integration test script [\#1285](https://github.com/fission/fission/pull/1285) ([life1347](https://github.com/life1347))
- support for providing multiple CMs and secrets in fn create [\#1282](https://github.com/fission/fission/pull/1282) ([viveksinghggits](https://github.com/viveksinghggits))
- Fix typo "consumer" =\> "producer" [\#1278](https://github.com/fission/fission/pull/1278) ([vadasambar](https://github.com/vadasambar))
- Fix go 1.9.2 & 1.11.4 shows "go mod not found" [\#1276](https://github.com/fission/fission/pull/1276) ([life1347](https://github.com/life1347))
- Fix typo "specialing" -\> "specializing" [\#1275](https://github.com/fission/fission/pull/1275) ([vadasambar](https://github.com/vadasambar))
- Added community meetings and some other changes [\#1273](https://github.com/fission/fission/pull/1273) ([vishal-biyani](https://github.com/vishal-biyani))
- fix a panic bug caused by err.Error\(\) [\#1271](https://github.com/fission/fission/pull/1271) ([cocoifly10](https://github.com/cocoifly10))
- Use fuzzy version for openJDK dependency [\#1269](https://github.com/fission/fission/pull/1269) ([life1347](https://github.com/life1347))
- Set full URL path to request header [\#1266](https://github.com/fission/fission/pull/1266) ([life1347](https://github.com/life1347))
- Refactor environment CLI command [\#1265](https://github.com/fission/fission/pull/1265) ([life1347](https://github.com/life1347))
- Added correct imagePullPolicy [\#1263](https://github.com/fission/fission/pull/1263) ([msshroff](https://github.com/msshroff))
- Make NewDeployment wait timeout configurable [\#1260](https://github.com/fission/fission/pull/1260) ([vadasambar](https://github.com/vadasambar))
- Issue \#1258: Allow empty repository tag in chart values.yaml [\#1259](https://github.com/fission/fission/pull/1259) ([parauliya](https://github.com/parauliya))
- Fix release script tags wrong image name [\#1257](https://github.com/fission/fission/pull/1257) ([life1347](https://github.com/life1347))
- Update tensorflow serving image name in env README [\#1256](https://github.com/fission/fission/pull/1256) ([life1347](https://github.com/life1347))
- V1.4.1 [\#1255](https://github.com/fission/fission/pull/1255) ([vishal-biyani](https://github.com/vishal-biyani))
- Fix environment version validation [\#1253](https://github.com/fission/fission/pull/1253) ([davidsmf](https://github.com/davidsmf))
- Allow to set deployment config uid during initialization [\#1249](https://github.com/fission/fission/pull/1249) ([life1347](https://github.com/life1347))
- Add swagger \(OpenAPI 2.0\) support [\#1245](https://github.com/fission/fission/pull/1245) ([life1347](https://github.com/life1347))
- Update go dependencies [\#1240](https://github.com/fission/fission/pull/1240) ([life1347](https://github.com/life1347))

## [1.4.1](https://github.com/fission/fission/tree/1.4.1) (2019-07-29)

[Full Changelog](https://github.com/fission/fission/compare/1.4.0...1.4.1)

**Merged pull requests:**

- Fix wrongly replace spec api version [\#1254](https://github.com/fission/fission/pull/1254) ([life1347](https://github.com/life1347))
- Revert change of product name in  README [\#1250](https://github.com/fission/fission/pull/1250) ([davidsmf](https://github.com/davidsmf))
- Analytics env fix in chart [\#1247](https://github.com/fission/fission/pull/1247) ([vishal-biyani](https://github.com/vishal-biyani))
- Fix CI unable to start test due to the same travis build ID [\#1246](https://github.com/fission/fission/pull/1246) ([life1347](https://github.com/life1347))
- V1.4.0 [\#1244](https://github.com/fission/fission/pull/1244) ([vishal-biyani](https://github.com/vishal-biyani))

## [1.4.0](https://github.com/fission/fission/tree/1.4.0) (2019-07-24)

[Full Changelog](https://github.com/fission/fission/compare/1.3.0...1.4.0)

**Merged pull requests:**

- Update flask version to resolve CVE alert [\#1243](https://github.com/fission/fission/pull/1243) ([life1347](https://github.com/life1347))
- Fix typo where "spec" is spelt "sepc" in some dump directories. [\#1239](https://github.com/fission/fission/pull/1239) ([davidsmf](https://github.com/davidsmf))
- Fix fetcher client doesn't handle error properly [\#1238](https://github.com/fission/fission/pull/1238) ([life1347](https://github.com/life1347))
- Fix build failed due to script unable to find configmap [\#1237](https://github.com/fission/fission/pull/1237) ([life1347](https://github.com/life1347))
- Enable concurrent CI builds [\#1236](https://github.com/fission/fission/pull/1236) ([life1347](https://github.com/life1347))
- Change log-level for better performance and less annoying logs [\#1231](https://github.com/fission/fission/pull/1231) ([life1347](https://github.com/life1347))
- Replace localhost with 127.0.0.1 to prevent DNS resolving problem [\#1227](https://github.com/fission/fission/pull/1227) ([life1347](https://github.com/life1347))
- Fix release script wrongly tags builder as env image [\#1226](https://github.com/fission/fission/pull/1226) ([life1347](https://github.com/life1347))
- Make router keep-alive setting configurable [\#1225](https://github.com/fission/fission/pull/1225) ([life1347](https://github.com/life1347))
- Function Update if config/secret changes [\#1224](https://github.com/fission/fission/pull/1224) ([vishal-biyani](https://github.com/vishal-biyani))
- Fix nil pointer when CLI unable to get server version [\#1223](https://github.com/fission/fission/pull/1223) ([life1347](https://github.com/life1347))
- Reuse go docker build cache [\#1218](https://github.com/fission/fission/pull/1218) ([life1347](https://github.com/life1347))
- Allow to set log level through environment variable [\#1217](https://github.com/fission/fission/pull/1217) ([life1347](https://github.com/life1347))
- Fix roundtripper doesn't increase request timeout setting after each retry [\#1216](https://github.com/fission/fission/pull/1216) ([life1347](https://github.com/life1347))
- Configmaps/secrets in function exist check [\#1214](https://github.com/fission/fission/pull/1214) ([vishal-biyani](https://github.com/vishal-biyani))
- Add experimental environment: tensorflow-serving [\#1212](https://github.com/fission/fission/pull/1212) ([life1347](https://github.com/life1347))
- Fix executor panic problem when specialization timeout [\#1211](https://github.com/fission/fission/pull/1211) ([life1347](https://github.com/life1347))
- Fix fetcher of newdeploy pod error out when starting up. [\#1210](https://github.com/fission/fission/pull/1210) ([life1347](https://github.com/life1347))
- Fix Istio-proxy cannot collect HTTP-level information [\#1209](https://github.com/fission/fission/pull/1209) ([life1347](https://github.com/life1347))
- Fix poolmanager specializes a function pod repeatedly if istio is enabled [\#1208](https://github.com/fission/fission/pull/1208) ([life1347](https://github.com/life1347))
- Reuse docker maven build cache in JVM environment build process [\#1207](https://github.com/fission/fission/pull/1207) ([life1347](https://github.com/life1347))
- modify code to make log collection comprehensive [\#1206](https://github.com/fission/fission/pull/1206) ([cocoifly10](https://github.com/cocoifly10))
- Fallback to get user home directory from env [\#1203](https://github.com/fission/fission/pull/1203) ([life1347](https://github.com/life1347))
- Fission v1.3.0 [\#1202](https://github.com/fission/fission/pull/1202) ([life1347](https://github.com/life1347))
- \[bugfix\] Fix CLI drops controller URL path when querying logs [\#1201](https://github.com/fission/fission/pull/1201) ([moluzhang](https://github.com/moluzhang))
- Enable go module support for go environment [\#1152](https://github.com/fission/fission/pull/1152) ([life1347](https://github.com/life1347))

## [1.3.0](https://github.com/fission/fission/tree/1.3.0) (2019-06-03)

[Full Changelog](https://github.com/fission/fission/compare/1.2.1...1.3.0)

**Merged pull requests:**

- Check fission CLI & server git commit SHA before test [\#1200](https://github.com/fission/fission/pull/1200) ([life1347](https://github.com/life1347))
- Add readiness/liveness probes to nat-streaming [\#1199](https://github.com/fission/fission/pull/1199) ([life1347](https://github.com/life1347))
- Update bug issue templates [\#1198](https://github.com/fission/fission/pull/1198) ([life1347](https://github.com/life1347))
- Add static code analysis to CI test [\#1197](https://github.com/fission/fission/pull/1197) ([life1347](https://github.com/life1347))
- Analytics bugfix [\#1195](https://github.com/fission/fission/pull/1195) ([soamvasani](https://github.com/soamvasani))
- Add Terraform configuration and upgrade helm version [\#1194](https://github.com/fission/fission/pull/1194) ([darkgerm](https://github.com/darkgerm))
- Show warning message if spec alters poolsize while env version \< 3 [\#1193](https://github.com/fission/fission/pull/1193) ([life1347](https://github.com/life1347))
- Move packages to proejct/pkg to follow go project folder structure convention  [\#1190](https://github.com/fission/fission/pull/1190) ([life1347](https://github.com/life1347))
- router analytics -- close http response body [\#1180](https://github.com/fission/fission/pull/1180) ([soamvasani](https://github.com/soamvasani))
- Remove prometheus server connectivity test during controller initialization [\#1179](https://github.com/fission/fission/pull/1179) ([life1347](https://github.com/life1347))
- V1.2.1 [\#1178](https://github.com/fission/fission/pull/1178) ([vishal-biyani](https://github.com/vishal-biyani))
- Skaffold for Fission [\#1172](https://github.com/fission/fission/pull/1172) ([vishal-biyani](https://github.com/vishal-biyani))
- Add affinity support [\#1170](https://github.com/fission/fission/pull/1170) ([laurence-hudson-mindfoundry](https://github.com/laurence-hudson-mindfoundry))
- Refactor test framework [\#1128](https://github.com/fission/fission/pull/1128) ([darkgerm](https://github.com/darkgerm))
- Pod specs [\#1106](https://github.com/fission/fission/pull/1106) ([vishal-biyani](https://github.com/vishal-biyani))
- Include All Currently Supported Trigger Types [\#1043](https://github.com/fission/fission/pull/1043) ([gravypod](https://github.com/gravypod))
- Allow non-toplevel modules in python environment [\#1042](https://github.com/fission/fission/pull/1042) ([soamvasani](https://github.com/soamvasani))
- Created dotnet2.0 Builder Image and Added /v2/specialized Endpoint to dotnet2.0 Envrionment  [\#1001](https://github.com/fission/fission/pull/1001) ([paraspatidar](https://github.com/paraspatidar))

## [1.2.1](https://github.com/fission/fission/tree/1.2.1) (2019-05-09)

[Full Changelog](https://github.com/fission/fission/compare/1.2.0...1.2.1)

**Merged pull requests:**

- Fix dotnet example [\#1175](https://github.com/fission/fission/pull/1175) ([CanerPatir](https://github.com/CanerPatir))
- V1.2.0 [\#1171](https://github.com/fission/fission/pull/1171) ([vishal-biyani](https://github.com/vishal-biyani))
- Fixes broken config path for functions [\#1177](https://github.com/fission/fission/pull/1177) ([vishal-biyani](https://github.com/vishal-biyani))

## [1.2.0](https://github.com/fission/fission/tree/1.2.0) (2019-05-03)

[Full Changelog](https://github.com/fission/fission/compare/1.1.0...1.2.0)

**Merged pull requests:**

- DRY up fetcher configuration [\#1168](https://github.com/fission/fission/pull/1168) ([vishal-biyani](https://github.com/vishal-biyani))
- Add simple anonymous usage metrics [\#1167](https://github.com/fission/fission/pull/1167) ([soamvasani](https://github.com/soamvasani))
- Fix the logger not working [\#1166](https://github.com/fission/fission/pull/1166) ([darkgerm](https://github.com/darkgerm))
- Change log level in executor for better log reading/troubleshooting [\#1163](https://github.com/fission/fission/pull/1163) ([life1347](https://github.com/life1347))
- Fix TravisCI go environment version to avoid go bugs [\#1154](https://github.com/fission/fission/pull/1154) ([life1347](https://github.com/life1347))
- \#1132 nodejs environment, increase body size [\#1149](https://github.com/fission/fission/pull/1149) ([JannikZed](https://github.com/JannikZed))
- Added php builder to release script fixes \#1140 [\#1145](https://github.com/fission/fission/pull/1145) ([vishal-biyani](https://github.com/vishal-biyani))
- Using templated imagePullPolicy for containers in deployment.yaml [\#1137](https://github.com/fission/fission/pull/1137) ([msshroff](https://github.com/msshroff))
- Migrate from glide to official dependencies management tool: Go Module [\#1136](https://github.com/fission/fission/pull/1136) ([life1347](https://github.com/life1347))
- Fix misleading log when setup portforward [\#1134](https://github.com/fission/fission/pull/1134) ([life1347](https://github.com/life1347))
- V1.1.0 [\#1129](https://github.com/fission/fission/pull/1129) ([vishal-biyani](https://github.com/vishal-biyani))
- support KUBECONFIG with multiple kube config files [\#1126](https://github.com/fission/fission/pull/1126) ([grounded042](https://github.com/grounded042))
- Function update after change in env [\#1116](https://github.com/fission/fission/pull/1116) ([vishal-biyani](https://github.com/vishal-biyani))
- Add configurable timeout to fission function test [\#1091](https://github.com/fission/fission/pull/1091) ([erwinvaneyk](https://github.com/erwinvaneyk))
- Add links to examples for each Fission environment [\#1090](https://github.com/fission/fission/pull/1090) ([erwinvaneyk](https://github.com/erwinvaneyk))

## [1.1.0](https://github.com/fission/fission/tree/1.1.0) (2019-03-25)

[Full Changelog](https://github.com/fission/fission/compare/1.0.0...1.1.0)

**Merged pull requests:**

- Add connection lost handler for NATS-streaming [\#1125](https://github.com/fission/fission/pull/1125) ([life1347](https://github.com/life1347))
- Change RBAC api version to v1 [\#1124](https://github.com/fission/fission/pull/1124) ([vishal-biyani](https://github.com/vishal-biyani))
- Configurable zero pool size in case of newdeploy function [\#1121](https://github.com/fission/fission/pull/1121) ([vishal-biyani](https://github.com/vishal-biyani))
- use zap for logging [\#1112](https://github.com/fission/fission/pull/1112) ([grounded042](https://github.com/grounded042))
- Support --plugin parameter in Fission CLI [\#1111](https://github.com/fission/fission/pull/1111) ([erwinvaneyk](https://github.com/erwinvaneyk))
- PHP 7.3 v2 Specialization [\#1110](https://github.com/fission/fission/pull/1110) ([AlbertoLopezBenito](https://github.com/AlbertoLopezBenito))
- Fix canary config manager creation error in controller [\#1105](https://github.com/fission/fission/pull/1105) ([life1347](https://github.com/life1347))
- Python examples: Added a minimal 'getting started' [\#1103](https://github.com/fission/fission/pull/1103) ([erwinvaneyk](https://github.com/erwinvaneyk))
- Added support for Ruby v2 Specialization [\#1101](https://github.com/fission/fission/pull/1101) ([brendanstennett](https://github.com/brendanstennett))
- V1.0.0 [\#1100](https://github.com/fission/fission/pull/1100) ([vishal-biyani](https://github.com/vishal-biyani))
- Adding annotations for prometheus scraping to fission-core [\#1098](https://github.com/fission/fission/pull/1098) ([vishal-biyani](https://github.com/vishal-biyani))
- Switch from fluentd to fluentbit for log forwarding [\#1086](https://github.com/fission/fission/pull/1086) ([soamvasani](https://github.com/soamvasani))
- Added draft proposal for CI/CD [\#1084](https://github.com/fission/fission/pull/1084) ([vishal-biyani](https://github.com/vishal-biyani))
- \[Kafka MQT\] Add warning about Kafka version [\#1083](https://github.com/fission/fission/pull/1083) ([bhavin192](https://github.com/bhavin192))
- Bump base image version of Go environment to 1.11.4 [\#1026](https://github.com/fission/fission/pull/1026) ([life1347](https://github.com/life1347))

## [1.0.0](https://github.com/fission/fission/tree/1.0.0) (2019-02-13)

[Full Changelog](https://github.com/fission/fission/compare/1.0...1.0.0)

**Merged pull requests:**

- V1.0 [\#1094](https://github.com/fission/fission/pull/1094) ([vishal-biyani](https://github.com/vishal-biyani))

## [1.0](https://github.com/fission/fission/tree/1.0) (2019-02-08)

[Full Changelog](https://github.com/fission/fission/compare/1.0-rc2...1.0)

**Merged pull requests:**

- Fix unable to update the function value of route [\#1081](https://github.com/fission/fission/pull/1081) ([darkgerm](https://github.com/darkgerm))
- Consider Pod Phase in IsReadyPod [\#1080](https://github.com/fission/fission/pull/1080) ([bhavin192](https://github.com/bhavin192))
- Spec archive optimisation [\#1068](https://github.com/fission/fission/pull/1068) ([vishal-biyani](https://github.com/vishal-biyani))
- Fix helm charts blank line [\#1065](https://github.com/fission/fission/pull/1065) ([darkgerm](https://github.com/darkgerm))
- Update helm charts README [\#1064](https://github.com/fission/fission/pull/1064) ([darkgerm](https://github.com/darkgerm))
- Make extra configuration a sub heading [\#1062](https://github.com/fission/fission/pull/1062) ([bhavin192](https://github.com/bhavin192))
- Remove/Redirect out-of-date docs to fission doc site [\#1061](https://github.com/fission/fission/pull/1061) ([life1347](https://github.com/life1347))
- V1.0 rc2 [\#1056](https://github.com/fission/fission/pull/1056) ([vishal-biyani](https://github.com/vishal-biyani))
- Mac test utility [\#986](https://github.com/fission/fission/pull/986) ([vishal-biyani](https://github.com/vishal-biyani))
- Fix executor tries to create same name deployment [\#1082](https://github.com/fission/fission/pull/1082) ([life1347](https://github.com/life1347))
- OpenTracing for Fission [\#1079](https://github.com/fission/fission/pull/1079) ([vishal-biyani](https://github.com/vishal-biyani))
- Fix fluentd plugin version to prevent version incompatible problem [\#1076](https://github.com/fission/fission/pull/1076) ([life1347](https://github.com/life1347))
- Clear message in case of function/pod failure [\#1069](https://github.com/fission/fission/pull/1069) ([vishal-biyani](https://github.com/vishal-biyani))
- Adding check for requirements file [\#1067](https://github.com/fission/fission/pull/1067) ([vishal-biyani](https://github.com/vishal-biyani))
- Fix threads change value of http.DefaultTransport in router [\#1063](https://github.com/fission/fission/pull/1063) ([life1347](https://github.com/life1347))
- Bumped up default CPU for fetcher, fixes \#1058 [\#1059](https://github.com/fission/fission/pull/1059) ([vishal-biyani](https://github.com/vishal-biyani))
- Replace router svcAddrUpdateLocks with new throttler package for code readability&reusability [\#1047](https://github.com/fission/fission/pull/1047) ([life1347](https://github.com/life1347))

## [1.0-rc2](https://github.com/fission/fission/tree/1.0-rc2) (2019-01-14)

[Full Changelog](https://github.com/fission/fission/compare/1.0-rc1...1.0-rc2)

**Merged pull requests:**

- solve kubernetes/client-go nested vendor [\#1048](https://github.com/fission/fission/pull/1048) ([yesqiao](https://github.com/yesqiao))
- Update dotnet and perl environment docs for rebuilding env images [\#1035](https://github.com/fission/fission/pull/1035) ([life1347](https://github.com/life1347))
- \[python-env\] PEP8 Fixes for server.py [\#1034](https://github.com/fission/fission/pull/1034) ([bhavin192](https://github.com/bhavin192))
- Fix builder not using latest image during CI build [\#1033](https://github.com/fission/fission/pull/1033) ([life1347](https://github.com/life1347))
- Add link to values.yaml in charts' README.md [\#1023](https://github.com/fission/fission/pull/1023) ([bhavin192](https://github.com/bhavin192))
- V1.0 rc1 [\#1022](https://github.com/fission/fission/pull/1022) ([life1347](https://github.com/life1347))
- Draft proposal for annotations [\#992](https://github.com/fission/fission/pull/992) ([vishal-biyani](https://github.com/vishal-biyani))
- Refactor RoundTrip function for code reading [\#991](https://github.com/fission/fission/pull/991) ([life1347](https://github.com/life1347))
- Changed Kafka topic name validation  [\#1051](https://github.com/fission/fission/pull/1051) ([vishal-biyani](https://github.com/vishal-biyani))
- Makes router URL for Kafka trigger configurable [\#1045](https://github.com/fission/fission/pull/1045) ([vishal-biyani](https://github.com/vishal-biyani))
- New deploy should clean up objects it created if there are errors [\#1040](https://github.com/fission/fission/pull/1040) ([vishal-biyani](https://github.com/vishal-biyani))
- Fix cli create archive with nonexistent file [\#1036](https://github.com/fission/fission/pull/1036) ([life1347](https://github.com/life1347))
- Use Header.Set\(\) to override the existing header value [\#1032](https://github.com/fission/fission/pull/1032) ([life1347](https://github.com/life1347))
- Fix go env panic when trying to load plugin and failed [\#1031](https://github.com/fission/fission/pull/1031) ([life1347](https://github.com/life1347))
- Fix builder shows "http: multiple response.WriteHeader calls" [\#1029](https://github.com/fission/fission/pull/1029) ([life1347](https://github.com/life1347))
- Add support for Kafka record headers [\#1025](https://github.com/fission/fission/pull/1025) ([bhavin192](https://github.com/bhavin192))
- Fix requests are sent to unready function pod \(newdeploy\) [\#1005](https://github.com/fission/fission/pull/1005) ([life1347](https://github.com/life1347))
- Send the error message to user when enabling canary feature fails. [\#990](https://github.com/fission/fission/pull/990) ([smruthi2187](https://github.com/smruthi2187))
- Add fluentd.conf as a configmap [\#792](https://github.com/fission/fission/pull/792) ([erwinvaneyk](https://github.com/erwinvaneyk))

## [1.0-rc1](https://github.com/fission/fission/tree/1.0-rc1) (2018-12-11)

[Full Changelog](https://github.com/fission/fission/compare/0.12.0...1.0-rc1)

**Merged pull requests:**

- Use executor type as a delimiter to prevent deploy name conflict [\#1009](https://github.com/fission/fission/pull/1009) ([life1347](https://github.com/life1347))
- Upgrade environment dependencies for security alert [\#1006](https://github.com/fission/fission/pull/1006) ([life1347](https://github.com/life1347))
- Rename canary flag name from funcN/funcN-1 to newfunc/oldfunc [\#1003](https://github.com/fission/fission/pull/1003) ([life1347](https://github.com/life1347))
- Update formatting directive logic to unbreak tests [\#999](https://github.com/fission/fission/pull/999) ([life1347](https://github.com/life1347))
- Alpine OpenJDK not available anymore [\#985](https://github.com/fission/fission/pull/985) ([vishal-biyani](https://github.com/vishal-biyani))
- Show builder image when list all envs [\#971](https://github.com/fission/fission/pull/971) ([life1347](https://github.com/life1347))
- V0.12.0 [\#967](https://github.com/fission/fission/pull/967) ([smruthi2187](https://github.com/smruthi2187))
- Updating the compile documentation link [\#965](https://github.com/fission/fission/pull/965) ([gguttikonda](https://github.com/gguttikonda))
- Specs for JVM example [\#825](https://github.com/fission/fission/pull/825) ([soamvasani](https://github.com/soamvasani))
- handle duplicate archive and package specs; handle multifile archives better [\#1018](https://github.com/fission/fission/pull/1018) ([soamvasani](https://github.com/soamvasani))
- Validate command flag input by adding cli hook [\#1017](https://github.com/fission/fission/pull/1017) ([life1347](https://github.com/life1347))
- Fix MQ trigger \(NATS\) wrongly sends error message to response topic [\#1002](https://github.com/fission/fission/pull/1002) ([life1347](https://github.com/life1347))
- Added warning to fix \#946 [\#996](https://github.com/fission/fission/pull/996) ([vishal-biyani](https://github.com/vishal-biyani))
- Package info error should warn user if package does not exist [\#995](https://github.com/fission/fission/pull/995) ([vishal-biyani](https://github.com/vishal-biyani))
- Fix newdeploy re-creates deployment when only minscale changed [\#988](https://github.com/fission/fission/pull/988) ([life1347](https://github.com/life1347))
- Fix release script failed to generate yaml for nonhelm installation [\#978](https://github.com/fission/fission/pull/978) ([life1347](https://github.com/life1347))
- Fix the analytics jobs in the YAMLs \(remove duplicates\) [\#977](https://github.com/fission/fission/pull/977) ([soamvasani](https://github.com/soamvasani))
- Pre-create kubernetes resources for function with minScale=0 [\#976](https://github.com/fission/fission/pull/976) ([life1347](https://github.com/life1347))
- Shorten poolmgr deployment name [\#975](https://github.com/fission/fission/pull/975) ([life1347](https://github.com/life1347))
- Fix issues when specifying resources/scales during updating/creation process [\#970](https://github.com/fission/fission/pull/970) ([life1347](https://github.com/life1347))
- Properly render Helm charts [\#969](https://github.com/fission/fission/pull/969) ([sdake](https://github.com/sdake))
- Fix CLI not shows package name when creating a function [\#966](https://github.com/fission/fission/pull/966) ([life1347](https://github.com/life1347))
- Fix Read on Closed body error  [\#963](https://github.com/fission/fission/pull/963) ([smruthi2187](https://github.com/smruthi2187))
- Archive package user experience [\#927](https://github.com/fission/fission/pull/927) ([vishal-biyani](https://github.com/vishal-biyani))

## [0.12.0](https://github.com/fission/fission/tree/0.12.0) (2018-11-01)

[Full Changelog](https://github.com/fission/fission/compare/0.11.0...0.12.0)

**Merged pull requests:**

- Keep prometheus and canary deploy set to false in fission-core [\#964](https://github.com/fission/fission/pull/964) ([smruthi2187](https://github.com/smruthi2187))
- Update readme to include JVM [\#953](https://github.com/fission/fission/pull/953) ([david-mcgillicuddy-ovo](https://github.com/david-mcgillicuddy-ovo))
- Bump flask version  [\#942](https://github.com/fission/fission/pull/942) ([life1347](https://github.com/life1347))
- Adding JVM heap options to environment [\#936](https://github.com/fission/fission/pull/936) ([vishal-biyani](https://github.com/vishal-biyani))
- Demo script updates [\#934](https://github.com/fission/fission/pull/934) ([soamvasani](https://github.com/soamvasani))
- Fix flag not found problem when running canary demo scripts [\#914](https://github.com/fission/fission/pull/914) ([life1347](https://github.com/life1347))
- V0.11.0 [\#913](https://github.com/fission/fission/pull/913) ([vishal-biyani](https://github.com/vishal-biyani))
- Fix failed to pull influxdb image from dockerhub [\#957](https://github.com/fission/fission/pull/957) ([life1347](https://github.com/life1347))
- Kafka tests [\#944](https://github.com/fission/fission/pull/944) ([vishal-biyani](https://github.com/vishal-biyani))
- fix a few canary deployment issues [\#943](https://github.com/fission/fission/pull/943) ([smruthi2187](https://github.com/smruthi2187))
- Support for full url  \(base on aalubin 882 changes\) [\#941](https://github.com/fission/fission/pull/941) ([life1347](https://github.com/life1347))
- Remove version from release name since it contains illegal chars for names [\#939](https://github.com/fission/fission/pull/939) ([soamvasani](https://github.com/soamvasani))
- Feature flag to enable/disable canary + optional prometheus install [\#937](https://github.com/fission/fission/pull/937) ([smruthi2187](https://github.com/smruthi2187))
- Return the error on failed specializations with `fn test --debug`  [\#917](https://github.com/fission/fission/pull/917) ([smruthi2187](https://github.com/smruthi2187))
- Added build and push procedures for Nodejs builder environment [\#916](https://github.com/fission/fission/pull/916) ([vishal-biyani](https://github.com/vishal-biyani))
- Add X-Forwarded-Host to request header [\#890](https://github.com/fission/fission/pull/890) ([life1347](https://github.com/life1347))
- Optimize function latency when cache expired/invalid under high concurrency [\#856](https://github.com/fission/fission/pull/856) ([life1347](https://github.com/life1347))

## [0.11.0](https://github.com/fission/fission/tree/0.11.0) (2018-10-01)

[Full Changelog](https://github.com/fission/fission/compare/0.10.0...0.11.0)

**Merged pull requests:**

- Print status with the get option. [\#907](https://github.com/fission/fission/pull/907) ([smruthi2187](https://github.com/smruthi2187))
- Fixed the spec validation UX issue [\#898](https://github.com/fission/fission/pull/898) ([vishal-biyani](https://github.com/vishal-biyani))
- Check CRD creation error instead of doing return directly [\#897](https://github.com/fission/fission/pull/897) ([life1347](https://github.com/life1347))
- Fix failed to find release-builder dockerfile & push specific tag [\#870](https://github.com/fission/fission/pull/870) ([life1347](https://github.com/life1347))
- V0.10.0 [\#868](https://github.com/fission/fission/pull/868) ([life1347](https://github.com/life1347))
- Fixes \#758, uses v2 specialize for env versions 2 or higher [\#911](https://github.com/fission/fission/pull/911) ([vishal-biyani](https://github.com/vishal-biyani))
- Java env test - Maven verbosity reduction [\#902](https://github.com/fission/fission/pull/902) ([vishal-biyani](https://github.com/vishal-biyani))
- Canary deployments for fission functions. [\#892](https://github.com/fission/fission/pull/892) ([smruthi2187](https://github.com/smruthi2187))
- Fix fetcher not close file descriptor correctly [\#889](https://github.com/fission/fission/pull/889) ([life1347](https://github.com/life1347))
- Removes the spec helm command for now to fix \#881 [\#886](https://github.com/fission/fission/pull/886) ([vishal-biyani](https://github.com/vishal-biyani))
- FIX CleanupOldExecutorObjects in all namespaces [\#879](https://github.com/fission/fission/pull/879) ([ajbouh](https://github.com/ajbouh))
- Check pod container ready state [\#861](https://github.com/fission/fission/pull/861) ([life1347](https://github.com/life1347))
- Configurable namespace creation [\#855](https://github.com/fission/fission/pull/855) ([michaelgaida](https://github.com/michaelgaida))
- Add v2 interface support for nodes env [\#836](https://github.com/fission/fission/pull/836) ([garyyeap](https://github.com/garyyeap))
- Kafka integration [\#831](https://github.com/fission/fission/pull/831) ([vishal-biyani](https://github.com/vishal-biyani))
- Fission supportability: Add dump command to dump information for debugging [\#754](https://github.com/fission/fission/pull/754) ([life1347](https://github.com/life1347))

## [0.10.0](https://github.com/fission/fission/tree/0.10.0) (2018-08-17)

[Full Changelog](https://github.com/fission/fission/compare/0.9.2...0.10.0)

**Merged pull requests:**

- Fix CLI failed to setup port-forward caused by \#712 [\#867](https://github.com/fission/fission/pull/867) ([life1347](https://github.com/life1347))
- Replay recorded requests by ReqUID [\#864](https://github.com/fission/fission/pull/864) ([Amusement](https://github.com/Amusement))
- Add cleanup function to test scripts [\#863](https://github.com/fission/fission/pull/863) ([life1347](https://github.com/life1347))
- Fix newdeploy fail to update HPA, deployment of a function after function update [\#862](https://github.com/fission/fission/pull/862) ([life1347](https://github.com/life1347))
- Fix router not taps function services [\#860](https://github.com/fission/fission/pull/860) ([life1347](https://github.com/life1347))
- Do resources validation when validate spec files [\#840](https://github.com/fission/fission/pull/840) ([life1347](https://github.com/life1347))
- Fixed the name of JVM builder image name [\#824](https://github.com/fission/fission/pull/824) ([vishal-biyani](https://github.com/vishal-biyani))
- V0.9.2 [\#817](https://github.com/fission/fission/pull/817) ([vishal-biyani](https://github.com/vishal-biyani))
- Add retry subcommand to pkg command [\#808](https://github.com/fission/fission/pull/808) ([life1347](https://github.com/life1347))
- add gevent based Python server to benchmark test cases [\#794](https://github.com/fission/fission/pull/794) ([xiekeyang](https://github.com/xiekeyang))
- Add more meaningful error messages to executor when getServiceForFunction [\#752](https://github.com/fission/fission/pull/752) ([life1347](https://github.com/life1347))
- Fix for \#662: avoid unnecessary builds [\#866](https://github.com/fission/fission/pull/866) ([smruthi2187](https://github.com/smruthi2187))
- Fix newdeploy not updates deployment after function's entrypoint changed [\#838](https://github.com/fission/fission/pull/838) ([life1347](https://github.com/life1347))
- Fix spec failed to archive single directory [\#837](https://github.com/fission/fission/pull/837) ([life1347](https://github.com/life1347))
- Uses a real go project to showcase vendor example so glide works [\#828](https://github.com/fission/fission/pull/828) ([vishal-biyani](https://github.com/vishal-biyani))
- Recorder CRD, Records API, Redis deployment [\#818](https://github.com/fission/fission/pull/818) ([Amusement](https://github.com/Amusement))
- Fix router panic when trying to update route [\#811](https://github.com/fission/fission/pull/811) ([life1347](https://github.com/life1347))
- Add query options to `fission function test` [\#782](https://github.com/fission/fission/pull/782) ([erwinvaneyk](https://github.com/erwinvaneyk))
- Add go environment vendor directory support [\#781](https://github.com/fission/fission/pull/781) ([life1347](https://github.com/life1347))
- Scale deployment to zero when function is in idle state [\#775](https://github.com/fission/fission/pull/775) ([life1347](https://github.com/life1347))
- Update binary environment readme [\#773](https://github.com/fission/fission/pull/773) ([erwinvaneyk](https://github.com/erwinvaneyk))
- Added readme for JVM environment [\#768](https://github.com/fission/fission/pull/768) ([vishal-biyani](https://github.com/vishal-biyani))
- Fix spec command overrides existing archive's url of a package [\#764](https://github.com/fission/fission/pull/764) ([life1347](https://github.com/life1347))
- Fixed typos from from goreportcard [\#760](https://github.com/fission/fission/pull/760) ([vishal-biyani](https://github.com/vishal-biyani))
- Extensible Fission CLI [\#743](https://github.com/fission/fission/pull/743) ([erwinvaneyk](https://github.com/erwinvaneyk))
- Updating releasing notes with details and structure [\#738](https://github.com/fission/fission/pull/738) ([vishal-biyani](https://github.com/vishal-biyani))
- Update route without providing function reference [\#718](https://github.com/fission/fission/pull/718) ([vishal-biyani](https://github.com/vishal-biyani))
- Allow router round-trip to be configurable [\#713](https://github.com/fission/fission/pull/713) ([xiekeyang](https://github.com/xiekeyang))
- Fix CLI failed to set up port-forwarding when multiple controller pods exist in the same namespace [\#712](https://github.com/fission/fission/pull/712) ([life1347](https://github.com/life1347))

## [0.9.2](https://github.com/fission/fission/tree/0.9.2) (2018-07-25)

[Full Changelog](https://github.com/fission/fission/compare/0.9.1...0.9.2)

**Merged pull requests:**

- Helm lint check in Travis CI [\#799](https://github.com/fission/fission/pull/799) ([erwinvaneyk](https://github.com/erwinvaneyk))
- Spelling. [\#797](https://github.com/fission/fission/pull/797) ([WrathZA](https://github.com/WrathZA))
- change image pull policy of builder manager [\#793](https://github.com/fission/fission/pull/793) ([xiekeyang](https://github.com/xiekeyang))
- Delete namespace in background to reduce build time [\#791](https://github.com/fission/fission/pull/791) ([life1347](https://github.com/life1347))
- Break & Stop the build immediately if a non-zero exit code was returned [\#790](https://github.com/fission/fission/pull/790) ([life1347](https://github.com/life1347))
- Add changelog. [\#789](https://github.com/fission/fission/pull/789) ([smruthi2187](https://github.com/smruthi2187))
- changes needed for 0.9.1 [\#788](https://github.com/fission/fission/pull/788) ([smruthi2187](https://github.com/smruthi2187))
- Working version of Java builder with Maven [\#783](https://github.com/fission/fission/pull/783) ([vishal-biyani](https://github.com/vishal-biyani))

## [0.9.1](https://github.com/fission/fission/tree/0.9.1) (2018-07-07)

[Full Changelog](https://github.com/fission/fission/compare/0.9.0...0.9.1)

**Merged pull requests:**

- Committing changelog. [\#780](https://github.com/fission/fission/pull/780) ([smruthi2187](https://github.com/smruthi2187))
- Changes in charts for release 0.9.0 [\#778](https://github.com/fission/fission/pull/778) ([smruthi2187](https://github.com/smruthi2187))
- Change flag name to KeepArchive for backward compatibility [\#787](https://github.com/fission/fission/pull/787) ([life1347](https://github.com/life1347))
- Fix go env plugin [\#784](https://github.com/fission/fission/pull/784) ([life1347](https://github.com/life1347))
- Fix “rm: missing operand” in release script [\#779](https://github.com/fission/fission/pull/779) ([life1347](https://github.com/life1347))

## [0.9.0](https://github.com/fission/fission/tree/0.9.0) (2018-07-04)

[Full Changelog](https://github.com/fission/fission/compare/0.8.0...0.9.0)

**Merged pull requests:**

- Fix executor wrongly passes loop variable reference to function [\#751](https://github.com/fission/fission/pull/751) ([life1347](https://github.com/life1347))
- Python Environment: add gevent based WSGI server framework [\#750](https://github.com/fission/fission/pull/750) ([xiekeyang](https://github.com/xiekeyang))
- Temporarily disabling the tests so that other PRs can be worked on [\#737](https://github.com/fission/fission/pull/737) ([vishal-biyani](https://github.com/vishal-biyani))
- add build exe to gitignore [\#736](https://github.com/fission/fission/pull/736) ([xiekeyang](https://github.com/xiekeyang))
- ArchiveLiteralSizeLimit: Use Constant Instead Hard Code [\#731](https://github.com/fission/fission/pull/731) ([xiekeyang](https://github.com/xiekeyang))
- Environment warning message bugfix [\#725](https://github.com/fission/fission/pull/725) ([soamvasani](https://github.com/soamvasani))
- V0.8.0 [\#722](https://github.com/fission/fission/pull/722) ([vishal-biyani](https://github.com/vishal-biyani))
- Make fetcher resource requests and limits configurable [\#708](https://github.com/fission/fission/pull/708) ([xiekeyang](https://github.com/xiekeyang))
- Add steps to render & upload fission installation YAML [\#745](https://github.com/fission/fission/pull/745) ([life1347](https://github.com/life1347))
- Fix executor not reaps idle function pods for functions with executortype newdeploy [\#744](https://github.com/fission/fission/pull/744) ([life1347](https://github.com/life1347))
- Testing with keep alive settings for connections [\#742](https://github.com/fission/fission/pull/742) ([vishal-biyani](https://github.com/vishal-biyani))
- instead hard code by variable in error message [\#735](https://github.com/fission/fission/pull/735) ([xiekeyang](https://github.com/xiekeyang))
- envns should be availabe in message line [\#734](https://github.com/fission/fission/pull/734) ([xiekeyang](https://github.com/xiekeyang))
- Support annotations in environment specs [\#733](https://github.com/fission/fission/pull/733) ([erwinvaneyk](https://github.com/erwinvaneyk))
- Extract portforward to separate package [\#728](https://github.com/fission/fission/pull/728) ([erwinvaneyk](https://github.com/erwinvaneyk))
- Push NATS error messages to error queue [\#724](https://github.com/fission/fission/pull/724) ([Amusement](https://github.com/Amusement))
- Fix for Windows CLI Port Forwarding [\#715](https://github.com/fission/fission/pull/715) ([thejosephstevens](https://github.com/thejosephstevens))
- Router liveness [\#701](https://github.com/fission/fission/pull/701) ([smruthi2187](https://github.com/smruthi2187))
- Archives bigger than 256K size need env variable for uploading [\#697](https://github.com/fission/fission/pull/697) ([vishal-biyani](https://github.com/vishal-biyani))
- Convert go-env Dockerfile into a multi-stage build [\#683](https://github.com/fission/fission/pull/683) ([jgall](https://github.com/jgall))
- Move build process from host to docker container during release process [\#682](https://github.com/fission/fission/pull/682) ([life1347](https://github.com/life1347))
- Added a flag to control the extraction of archive based on user input [\#675](https://github.com/fission/fission/pull/675) ([vishal-biyani](https://github.com/vishal-biyani))
- Java env alpha [\#656](https://github.com/fission/fission/pull/656) ([vishal-biyani](https://github.com/vishal-biyani))

## [0.8.0](https://github.com/fission/fission/tree/0.8.0) (2018-06-05)

[Full Changelog](https://github.com/fission/fission/compare/0.7.2...0.8.0)

**Merged pull requests:**

- Pre-install/pre-upgrade hooks to verify func references and assign restricted role bindings [\#717](https://github.com/fission/fission/pull/717) ([smruthi2187](https://github.com/smruthi2187))
- Logger daemonset's update strategy [\#714](https://github.com/fission/fission/pull/714) ([vishal-biyani](https://github.com/vishal-biyani))
- Check spec directory exists before reading spec files [\#709](https://github.com/fission/fission/pull/709) ([life1347](https://github.com/life1347))
- Formatted specifiers are not compatible with variables [\#706](https://github.com/fission/fission/pull/706) ([xiekeyang](https://github.com/xiekeyang))
- Indicate HTTP status code by library const [\#703](https://github.com/fission/fission/pull/703) ([xiekeyang](https://github.com/xiekeyang))
- docker-distribution version bump for windows compatibility [\#691](https://github.com/fission/fission/pull/691) ([thejosephstevens](https://github.com/thejosephstevens))
- Version -\> 0.7.2 [\#670](https://github.com/fission/fission/pull/670) ([life1347](https://github.com/life1347))
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

[Full Changelog](https://github.com/fission/fission/compare/latest...0.6.1)

**Merged pull requests:**

- This change fixes an error in a yaml file in the fission-core chart. [\#563](https://github.com/fission/fission/pull/563) ([smartding](https://github.com/smartding))
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

## [latest](https://github.com/fission/fission/tree/latest) (2018-03-01)

[Full Changelog](https://github.com/fission/fission/compare/0.6.0...latest)

## [0.6.0](https://github.com/fission/fission/tree/0.6.0) (2018-03-01)

[Full Changelog](https://github.com/fission/fission/compare/0.5.0...0.6.0)

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
- use time.Since instead of time.Now\(\).Sub [\#449](https://github.com/fission/fission/pull/449) ([wgliang](https://github.com/wgliang))
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
- Multiple Trigger Definitions Fix [\#341](https://github.com/fission/fission/pull/341) ([jsturtevant](https://github.com/jsturtevant))
- Fission workflow env integration [\#336](https://github.com/fission/fission/pull/336) ([erwinvaneyk](https://github.com/erwinvaneyk))
- Add builder manager support [\#308](https://github.com/fission/fission/pull/308) ([life1347](https://github.com/life1347))

## [buildmgr-preview-20170922](https://github.com/fission/fission/tree/buildmgr-preview-20170922) (2017-09-22)

[Full Changelog](https://github.com/fission/fission/compare/buildmgr-preview-20170921...buildmgr-preview-20170922)

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
- Remove redundant hello.js from charts directory [\#130](https://github.com/fission/fission/pull/130) ([sanketsudake](https://github.com/sanketsudake))
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
- Primary Helm chart for fission [\#90](https://github.com/fission/fission/pull/90) ([sanketsudake](https://github.com/sanketsudake))
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

[Full Changelog](https://github.com/fission/fission/compare/90c14cfa0808aa1d63ea55ad87bee0f651f45091...kubecon)

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



