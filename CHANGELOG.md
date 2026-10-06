# Changelog

## [0.1.2](https://github.com/rshade/finfocus-plugin-opencost/compare/v0.1.1...v0.1.2) (2026-10-06)


### Bug Fixes

* **release:** build archives only and use a real git state field (OC-9.15) ([#88](https://github.com/rshade/finfocus-plugin-opencost/issues/88)) ([54de975](https://github.com/rshade/finfocus-plugin-opencost/commit/54de97530819368712272873864722ae59b7277d))

## [0.1.1](https://github.com/rshade/finfocus-plugin-opencost/compare/v0.1.0...v0.1.1) (2026-10-06)


### Bug Fixes

* **allocation:** filter controllers on controllerName (OC-9.3) ([#82](https://github.com/rshade/finfocus-plugin-opencost/issues/82)) ([1b5ecec](https://github.com/rshade/finfocus-plugin-opencost/commit/1b5ecec1b117ba1fb63ef5d0e95987de6bf60fa1))
* **release:** keep the component out of the release tag (OC-9.15) ([#86](https://github.com/rshade/finfocus-plugin-opencost/issues/86)) ([2da0fef](https://github.com/rshade/finfocus-plugin-opencost/commit/2da0fef02f2229b3298b426719d8a81388c47e2a))

## 0.1.0 (2026-10-05)


### Features

* add a kubernetes pulumi example (OC-6.2) ([20a7613](https://github.com/rshade/finfocus-plugin-opencost/commit/20a76139d89cfe0e8c943acb431fe3d7ed777a92))
* add stable OpenCost and Kubecost allocation profiles (OC-3.1) ([f9e6205](https://github.com/rshade/finfocus-plugin-opencost/commit/f9e620575f5bef2b083551ee78de1b24533831c1))
* adopt finfocus-spec v0.7.3 attributes ([fb3c25e](https://github.com/rshade/finfocus-plugin-opencost/commit/fb3c25efef615c9982d9805f9ec02c41af09dc7a))
* aggregate budget health into summary counts (OC-7.4) ([a8c59b3](https://github.com/rshade/finfocus-plugin-opencost/commit/a8c59b3c1eb7c9e10b68f502970b319a8fa2e4b6))
* carry allocation metadata on actual cost (OC-3.7) ([03228de](https://github.com/rshade/finfocus-plugin-opencost/commit/03228deb1fefc69872a56c1b24d88c095dcacda8))
* decode recorded OpenCost allocations into typed costs (OC-3.2) ([66b5d94](https://github.com/rshade/finfocus-plugin-opencost/commit/66b5d947dbfabfb0bcf66ad656076aa02e8fc16a))
* enhance Kubecost API client with complete allocation endpoint support ([8c19af4](https://github.com/rshade/finfocus-plugin-opencost/commit/8c19af4fce857d33686960f6148151be473f6816)), closes [#2](https://github.com/rshade/finfocus-plugin-opencost/issues/2)
* enhance Kubecost API client with complete allocation endpoint support ([#20](https://github.com/rshade/finfocus-plugin-opencost/issues/20)) ([43cd3f6](https://github.com/rshade/finfocus-plugin-opencost/commit/43cd3f62fd8ceca7c9cb16876a8f6c60296dbcb9)), closes [#2](https://github.com/rshade/finfocus-plugin-opencost/issues/2)
* enhance Kubecost API client with complete allocation endpoint support ([#22](https://github.com/rshade/finfocus-plugin-opencost/issues/22)) ([d0f8466](https://github.com/rshade/finfocus-plugin-opencost/commit/d0f84663bf66cea52df65f8f760c8fffc44516af)), closes [#2](https://github.com/rshade/finfocus-plugin-opencost/issues/2)
* estimate and batch costs in request order (OC-3.5) ([4ecb28f](https://github.com/rshade/finfocus-plugin-opencost/commit/4ecb28f28ca89dcbdbf6ed9f17f3fa57f5a1c9f8))
* estimate kubecost profile from spec cost (OC-7.1) ([d514642](https://github.com/rshade/finfocus-plugin-opencost/commit/d51464261712d561d76296903c1c425896562cc3))
* filter budgets by namespace tag (OC-7.3) ([05c4efa](https://github.com/rshade/finfocus-plugin-opencost/commit/05c4efa2ef17f80f4d72aa3aa2192d46d687817f))
* filter GetActualCost by recorded resource ids (OC-3.3) ([2cca24d](https://github.com/rshade/finfocus-plugin-opencost/commit/2cca24d669c586d6878fa464aed74bd8c65f59be))
* list namespace budgets from the budget api (OC-7.2) ([aec562e](https://github.com/rshade/finfocus-plugin-opencost/commit/aec562e917cb0a5bdf59708d3a890fc42335e29a))
* log the spec trace id on cost-source requests (OC-2.5) ([c5c8e0e](https://github.com/rshade/finfocus-plugin-opencost/commit/c5c8e0ed09c50017b28a589456cb3761e9b44f4b)), closes [#36](https://github.com/rshade/finfocus-plugin-opencost/issues/36)
* map allocation failures to gRPC statuses (OC-2.6) ([72f23b5](https://github.com/rshade/finfocus-plugin-opencost/commit/72f23b57aee39f287da05d660597e2dfc31d04e2)), closes [#9](https://github.com/rshade/finfocus-plugin-opencost/issues/9)
* price kubernetes resources in GetPricingSpec (OC-5.4) ([839b9ea](https://github.com/rshade/finfocus-plugin-opencost/commit/839b9ea2594d19de926d82dfd45c805336b46d64))
* probe backend health and log request counters (OC-4.3) ([20c9562](https://github.com/rshade/finfocus-plugin-opencost/commit/20c95628d10e71d40fcda0dcac3e4ac476f15f04))
* project one resource to a 730-hour month (OC-3.4) ([7ca6a81](https://github.com/rshade/finfocus-plugin-opencost/commit/7ca6a813c127d17a06819f00d02a44a5f0017336)), closes [#4](https://github.com/rshade/finfocus-plugin-opencost/issues/4)
* register the cost source through the spec SDK (OC-2.1) ([bc0c5e1](https://github.com/rshade/finfocus-plugin-opencost/commit/bc0c5e1c54de65abc5d94ba15b9b6240081bcd94)), closes [#8](https://github.com/rshade/finfocus-plugin-opencost/issues/8)
* report plugin info with spec capabilities (OC-2.2) ([0321fc0](https://github.com/rshade/finfocus-plugin-opencost/commit/0321fc0be15454c7afc53d31e67360cb8f4e4c83)), closes [#5](https://github.com/rshade/finfocus-plugin-opencost/issues/5)
* set cost currency from allocation or config (OC-3.8) ([563d31e](https://github.com/rshade/finfocus-plugin-opencost/commit/563d31e5bd65649b82c9b76bfbfaa57ed2f3ff1e))
* support the Kubernetes types core sends (OC-2.3) ([f7e1cde](https://github.com/rshade/finfocus-plugin-opencost/commit/f7e1cde772fc76cca3155097cbc3aa5aa88c84ff)), closes [#3](https://github.com/rshade/finfocus-plugin-opencost/issues/3)


### Bug Fixes

* correct actual-cost timestamps, labels, and estimates ([c097f54](https://github.com/rshade/finfocus-plugin-opencost/commit/c097f544e657103d2f080739a426ca213d75f78b))
* count a shared namespace budget once ([7bf8d54](https://github.com/rshade/finfocus-plugin-opencost/commit/7bf8d54d3efb8510809f2e3942b35837ce029b21))
* **deps:** update module github.com/rshade/finfocus-spec to v0.7.4 ([#80](https://github.com/rshade/finfocus-plugin-opencost/issues/80)) ([60ff368](https://github.com/rshade/finfocus-plugin-opencost/commit/60ff368fcd0ca92cd94026ac1e3a72af487895ff))
* install the server logger before serve ([e039b30](https://github.com/rshade/finfocus-plugin-opencost/commit/e039b307d8bb5bf5afbc535292fc7e9d227dc6ee))
* keep a failed kind cluster until logs are exported ([9044084](https://github.com/rshade/finfocus-plugin-opencost/commit/9044084058dd9979a54ac0965cbfb99b146fdb98))
* mark the kind e2e script executable (OC-5.3) ([3d8cf10](https://github.com/rshade/finfocus-plugin-opencost/commit/3d8cf10c38d14d01f3133cd00eef729a1995b63b))
* match kubecost estimates to configured namespace ([95f4f83](https://github.com/rshade/finfocus-plugin-opencost/commit/95f4f83c8a14e3ef1f51b2c3da7fdb3c05d250a3))
* reject an empty id and a disagreeing resource type tag ([f16fbad](https://github.com/rshade/finfocus-plugin-opencost/commit/f16fbad77e9c68251e22a0c02b456512d1e19456))
* **release:** set first release to 0.1.0 (OC-9.1) ([#79](https://github.com/rshade/finfocus-plugin-opencost/issues/79)) ([88784d1](https://github.com/rshade/finfocus-plugin-opencost/commit/88784d13c3ed719f5b3686e9fb580adab51785ce)), closes [#77](https://github.com/rshade/finfocus-plugin-opencost/issues/77)
* resolve kubernetes names without parsing a pulumi urn ([38334f6](https://github.com/rshade/finfocus-plugin-opencost/commit/38334f67eed61f3e010d05e3f74f4ed270db9c85))
* return every namespace on a budget rule ([c72a006](https://github.com/rshade/finfocus-plugin-opencost/commit/c72a00616d134fc8c1a522c1383d6b1e0808b240))
* satisfy section 7 lint, drift, and govulncheck ([5060965](https://github.com/rshade/finfocus-plugin-opencost/commit/5060965d2793de4d81f530343436e3fdb2b65cb9))
* verify tls and reject hostile allocation filters (OC-4.1) ([e843231](https://github.com/rshade/finfocus-plugin-opencost/commit/e843231b46da32e51eae7f167a7425b92348d292))


### Documentation

* align the architecture notes with the review fixes ([7038d84](https://github.com/rshade/finfocus-plugin-opencost/commit/7038d8487af02bb6c635bfb7ed0814a3eadc52b3))
* archive the remaining behaviour specs (OC-8.2) ([dceb13e](https://github.com/rshade/finfocus-plugin-opencost/commit/dceb13e9d9b8b921eeae73e41394cebc42929b43))
* decline AllocatorService for conservation (OC-3.6) ([5ad0821](https://github.com/rshade/finfocus-plugin-opencost/commit/5ad08211dd016ecb74162cd72b44aa0cc2109b7a))
* describe both allocation profiles (OC-6.3) ([2238147](https://github.com/rshade/finfocus-plugin-opencost/commit/2238147ffd1d83461ada5406a7fa71d7f5d33494))
* limit kubecost to profile history (OC-6.4) ([81a585e](https://github.com/rshade/finfocus-plugin-opencost/commit/81a585e91f4f1d985561a29ebb5950bf17991608))
* list kubecost contract fixture sources (OC-7.6) ([e6d1e19](https://github.com/rshade/finfocus-plugin-opencost/commit/e6d1e1932f387b71fe0350d63a3f487ea4611c05))
* name the opencost plugin in package descriptions ([56b31d8](https://github.com/rshade/finfocus-plugin-opencost/commit/56b31d8c4c6da212626477a37d6ab864ddeb6ac7))
* record the dependency dashboard disposition (OC-8.1) ([1089671](https://github.com/rshade/finfocus-plugin-opencost/commit/1089671537442ae3865ee891bfae0f04c806b9cf))
* record the issues closed against the shipped plugin ([#62](https://github.com/rshade/finfocus-plugin-opencost/issues/62)) ([57c69e9](https://github.com/rshade/finfocus-plugin-opencost/commit/57c69e9e71974b39352173db9126da5006c33f32))
* record the kubecost kind chart spike (OC-7.5) ([3e9a73d](https://github.com/rshade/finfocus-plugin-opencost/commit/3e9a73db65754ff3d6992422b9b5f7d829ebf5cf))
* record the OC-8.3 run report status ([01e7663](https://github.com/rshade/finfocus-plugin-opencost/commit/01e7663ecd076046ca982c8be46f9f16705b2f55))
* record which Kubernetes types core sends (OC-2.4) ([a4160c4](https://github.com/rshade/finfocus-plugin-opencost/commit/a4160c4d1d6b14233231206999428e6c0f528172))
