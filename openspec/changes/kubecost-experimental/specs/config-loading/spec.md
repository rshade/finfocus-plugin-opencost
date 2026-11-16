# Spec Delta

## ADDED Requirements

### Requirement: Profile kubecost warns at start-up

On profile `kubecost`, start-up SHALL log exactly one WARN line saying the profile is experimental and not verified against a live backend. Profile `opencost` SHALL log no such line.

#### Scenario: Profile kubecost

- **WHEN** the binary starts with `OPENCOST_PROFILE=kubecost`
- **THEN** stderr contains the experimental warning once

#### Scenario: Profile opencost

- **WHEN** the binary starts with profile `opencost`
- **THEN** stderr does not contain the warning

Tested by: TestKubecostProfileWarnsAtStartup
