# Spec Delta

## ADDED Requirements

### Requirement: Profile kubecost is labelled experimental

On profile `kubecost`, `GetPluginInfo` metadata SHALL carry `kubecost_profile` = `experimental` and `kubecost_verified` = `false`. On profile `opencost` neither key SHALL be present.

#### Scenario: Profile kubecost

- **WHEN** `GetPluginInfo` is called with profile `kubecost`
- **THEN** metadata `kubecost_profile` is `experimental` and `kubecost_verified` is `false`

#### Scenario: Profile opencost

- **WHEN** `GetPluginInfo` is called with profile `opencost`
- **THEN** metadata has neither key

Tested by: TestGetPluginInfoLabelsKubecostExperimental
