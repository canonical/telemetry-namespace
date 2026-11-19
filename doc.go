// Package namespace is the reference implementation of the Telemetry MQTT
// namespace: wire topics, identifiers, topic selectors, MQTT filter matching,
// the cc-* user properties and the _system payloads.
//
// The Connect Agent, the Telemetry Management Service, the connector service
// and the CLI all import the subpackages of this module, so that they build,
// parse and validate topics identically. The module has no runtime
// dependencies.
//
// Specifications: TM102 (Topic Namespace & User Properties) and TM103
// (Topic Selectors).
package namespace
