# Spec Delta

## ADDED Requirements

### Requirement: includeUsage loads from the file and the environment

The configuration SHALL read the boolean `includeUsage` from YAML, default `false`. `OPENCOST_INCLUDE_USAGE` SHALL override the file value when it is set and not empty. A value that is not a boolean SHALL be a load error that names `OPENCOST_INCLUDE_USAGE`.

#### Scenario: Environment overrides the file

- **WHEN** the file sets `includeUsage: true` and `OPENCOST_INCLUDE_USAGE=false`
- **THEN** the loaded value is `false`

#### Scenario: Invalid value

- **WHEN** `OPENCOST_INCLUDE_USAGE=maybe`
- **THEN** loading fails and the error names `OPENCOST_INCLUDE_USAGE`

Tested by: TestIncludeUsageConfig
