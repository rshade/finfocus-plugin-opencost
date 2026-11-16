# Spec Delta

## ADDED Requirements

### Requirement: Recommendations need a headroom and profile opencost

The configuration SHALL read `recommendations` (default `false`) and `recommendationHeadroom` (no default) from YAML. `OPENCOST_RECOMMENDATIONS` and `OPENCOST_RECOMMENDATION_HEADROOM` SHALL override the file when set. Start-up validation SHALL reject `recommendations` with a headroom at or below zero, and with profile `kubecost`. An unparsable env value SHALL be a load error naming the variable.

#### Scenario: Enabled without a headroom

- **WHEN** `recommendations` is true and `recommendationHeadroom` is unset
- **THEN** validation fails and names `recommendationHeadroom`

#### Scenario: Profile kubecost

- **WHEN** `recommendations` is true with profile `kubecost`
- **THEN** validation fails and names the profile

Tested by: TestRecommendationsConfig
