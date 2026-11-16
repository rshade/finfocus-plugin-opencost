# FOCUS split-cost columns for OpenCost allocation

Spike for issue #137, 2026-11-15. The question: should this plugin fill the FOCUS 1.3 and 1.4 split cost allocation columns that `FocusCostRecord` carries (`allocated_method_id`, `allocated_method_details`, `allocated_resource_id`, `allocated_resource_name`, `allocated_tags`)?

## Verdict: kill

Do not fill the split-cost columns. Those columns describe how an origin charge on a shared resource is divided among recipients, with ratios that sum to one. This plugin does not emit that origin charge, and OpenCost's totals are not a split of one. Filling the columns would assert a split the backend never computed. No consumer reads them today either. "What would reopen this" lists the changes that would make the question worth asking again.

FOCUS sources were read from `github.com/FinOps-Open-Cost-and-Usage-Spec/FOCUS_Spec` at tags `v1.3`, `v1.4`, and `v1.5-cr`, under `specification/`. FinFocus core was read from a local clone at commit `79b11388`.

## 1. The FOCUS 1.4 shape of AllocatedMethodDetails

`datasets/cost_and_usage/columns/allocatedmethoddetails.md@v1.4`:

- It is Recommended, and it "MUST be of type JSON Object (serialized as a String where necessary)" (line 13). FOCUS 1.3 typed it as a String.
- It "SHOULD NOT be null when a charge is related to" a data-generator split (line 18).
- The object (lines 44 to 56) has an `Elements` array, which is required. Each element has:
  - `AllocatedRatio`: Numeric, required.
  - `UsageUnit`: String, required when `UsageQuantity` is present.
  - `UsageQuantity`: Numeric, optional.
- "The sum of … AllocatedRatio across all allocated charges related to a single origin charge MUST be equal to 1" (line 33).
- Custom keys use the `x_` prefix in PascalCase (line 62).
- The example (lines 71 to 82) is `{"Elements":[{"AllocatedRatio":0.05,"UsageUnit":"CPU","UsageQuantity":0.5},{"AllocatedRatio":0.1,"UsageUnit":"Memory","UsageQuantity":4}]}`.

A ratio needs a denominator: the origin charge. This plugin has none (see question 2).

## 2. What goes in resource_id for a row with no single host

FOCUS uses `ResourceId` for the shared origin resource and `AllocatedResourceId` for the recipient:

- An origin charge is split "into multiple charge records (allocated charges) … while ensuring the origin charge can be derived". Dimension columns, including `ResourceId`, "MUST match the corresponding value in the origin charge", and summable metrics sum back to the origin. Source: `attributes/data_generator_calculated_split_cost_allocation_handling.md@v1.4`, lines 3, 9, and 11.
- The supported-feature page names "compute nodes in a shared cluster (Kubernetes…)". Source: `supported_features/data_generator_calculated_split_cost_allocation.md@v1.4`, line 5.
- The 1.5 working draft's worked example spells it out: every row's `ResourceId` is the host, `AllocatedResourceId` is the pod, and the unallocated origin row has a method id with a null `AllocatedResourceId`. Source: `appendix/split_cost_allocation_examples.md@v1.5-cr`, lines 34 to 39, and `split_cost_allocation_field_mapping.md@v1.5-cr`, lines 35 to 43, 59, 88, and 103.
- `AllocatedResourceId` "MUST be null when a charge represents the unallocated portion of the origin charge" and "MUST NOT be null when a charge represents the allocated portion". Source: `allocatedresourceid.md@v1.4`, lines 13 and 14.

**Pod and controller rows.** The node is known (`properties.node`), but the row's cost is OpenCost's own price, not a ratio of a priced node charge. `docs/allocator-decision.md` records the evidence: the recorded namespace totals sum to 0.36187 against any host node price, and OpenCost does not take a host price as input. Setting `ResourceId` to the node would claim the node charge was split, with no origin row and no ratios.

**Namespace rows.** Several nodes can host a namespace, so there is no single origin resource.

The honest values are therefore the ones the plugin sends today: an empty `resource_id` on the record, and no `allocated_*` columns.

## 3. Moving labels from tags to allocated_tags

No.

- `AllocatedTags` "MUST include all applicable … tags for the AllocatedResourceId" and "MUST NOT include resource tags already present in Tags". Source: `allocatedtags.md@v1.4`, lines 16 and 17. They only mean something when `AllocatedResourceId` is set, which question 2 rules out.
- Today nothing breaks either way. FinFocus core does not read `tags` from an actual-cost record (question 4).
- A third-party reader of `tags` would lose labels for no gain. The move would also contradict the `actual-cost` spec requirement that copies labels to `focus_record.tags`.

## 4. Does core read the allocated columns

No. In the local clone of `rshade/finfocus`, `rg -i "allocated_?(method|resource|tags)|GetAllocated|AllocatedResource"` finds no Go or doc use. The only hits are `AllocatedScopes` (budgets) and `allocatedStorage` fixtures.

From a `FocusRecord`, core reads only four fields, all in `internal/proto/adapter.go`:

- `ChargePeriodStart` and `ChargePeriodEnd`, for daily bucketing (lines 1760, 1764, and 1787 to 1809);
- `BillingCurrency` and `PricingCurrency` (lines 1814 to 1820 and 1829 to 1837).

Core does not read `resource_id`, `tags`, or `extended_columns` from an actual-cost record. Every `GetTags()` call is on a `ResourceDescriptor`, a budget filter, or recommendation info. Populating the columns today has no consumer.

## 5. A documented method id

FOCUS defines `AllocatedMethodId` as an identifier "defined by the service provider … can be used to find how the allocated charge was calculated in the provider's documentation". The method documentation "MUST reference a single AllocatedMethodId value" (`allocatedmethodid.md@v1.4`, lines 3 and 14). The value format is "not specified" (line 37).

No standard values exist. The only example is the fictitious `aura-vcpu-proportional-v1` (`split_cost_allocation_field_mapping.md@v1.5-cr`, lines 72 and 101). OpenCost does document its own model: `spec/opencost-specv01.md` on the develop branch says workload cost is `max(request, usage)` at container level (line 153), and asset cost equals workload plus idle cost (line 74). But OpenCost publishes no method id and no FOCUS export with these columns, and its FOCUS use is ingesting custom costs (`protos/customcost/messages.proto`, lines 44 to 84).

The data generator here is OpenCost, not this plugin. A method id published by the plugin would be an invented method. #137's boundary check rules that out: "The plugin must not invent a host or a method".

## A conflict worth reporting upstream

`finfocus-spec` v0.7.12 `proto/finfocus/v1/focus.proto` line 444 says: "If populated, allocated_resource_id MUST also be populated". The SDK enforces this in `sdk/go/testing/allocation.go:360` and `allocator_conformance.go:393`. That is stricter than FOCUS, whose origin and unallocated rows carry a method id with a null `AllocatedResourceId`. Separately, `allocation.proto` lines 177 to 192 use `AllocationRow.allocated_resource_id` for the priced host (a node), which is the opposite of FOCUS's recipient meaning. Neither affects this plugin, which fills none of these columns. A plugin that does emit FOCUS splits would hit both. This is a `finfocus-spec` question and is not filed from here.

## What would reopen this

Reopen this question if all of the following change:

- the plugin emits the origin charge of a priced shared resource (for example a node from an assets query), so that allocated rows can reference that resource and carry ratios that sum to one;
- OpenCost, or a FinFocus allocator, documents a method that the plugin can name rather than invent;
- core, or another consumer the owner names, reads the `allocated_*` columns.

Any one of these alone does not justify the columns.
