# Spec Delta

## ADDED Requirements

### Requirement: Log lines name the plugin with the SDK field

Every log line the plugin writes SHALL carry `plugin_name` `opencost` and
`plugin_version`, the fields `pluginsdk.NewPluginLogger` sets. A line SHALL
NOT carry a `plugin` field.

#### Scenario: Startup error line

- **WHEN** the plugin starts with no base URL and exits 1
- **THEN** each stderr line is JSON with `plugin_name` `opencost` and `plugin_version` equal to the build version
- **AND** no line has a `plugin` field
