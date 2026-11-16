# Tasks

## 1. billing_detail

- [x] 1.1 Write `TestProjectedBillingDetailNamespaceIncludesWorkloads` (namespace row has the sentence, controller row does not) and watch it fail; the break check is dropping the kind test
- [x] 1.2 Pass the `resourceRef` into `projectedResponse` and choose the detail by kind, in both `GetProjectedCost` and the projected batch item; verify the new test and `TestGetProjectedCostUsesRequestedResource` pass

## 2. Example and docs

- [x] 2.1 Remove the Namespace resource from `examples/kubernetes/Pulumi.yaml`, use the literal namespace, regenerate `preview.json`, and update `TestKubernetesExamplePreviewFixture` to require no Namespace step
- [x] 2.2 State the overlap in the README, `docs/resource-mapping.md`, and `examples/kubernetes/README.md`; verify markdownlint and the doc tests pass

## 3. Verify

- [x] 3.1 Run `go test ./...`, `go vet ./...`, `make lint`, `make test-race`, markdownlint, and `mise exec -- openspec validate --all --strict`
- [x] 3.2 Run the built binary through `finfocus cost projected` on the regenerated preview against kind and confirm the total is not doubled, then update `CLAUDE.md`
