# Tasks

## 1. Spec and filter key

- [x] 1.1 Amend the controller scenario to filter on `controllerName` and keep aggregate `namespace,controller`.
- [x] 1.2 Map the controller filter key separately from the aggregate key.

## 2. Tests

- [x] 2.1 Update the actual-cost and identity assertions to `controllerName:"fixed"`.
- [x] 2.2 Reject a generated filter key named `controller`.
- [x] 2.3 Read the captured HTTP 500 and the captured controller row from `testdata/opencost-real-controller/`.
- [x] 2.4 Price Deployment `fixed` in the kind oracle.

## 3. Record

- [x] 3.1 Run the break check that restores the key `controller` and record the failure.
- [x] 3.2 Archive the change after `openspec validate --all --strict` passes.
