package sippyserver

const (
	ACSCapability = "acs_releases"

	// LocalDB is whether we have a local DB client (currently postgres)
	LocalDBCapability = "local_db"

	// BuildclusterCapability is whether we have build cluster health data.
	BuildClusterCapability = "build_clusters"

	// ComponentReadiness capability is whether this sippy instance is configured for Component Readiness
	ComponentReadinessCapability = "component_readiness"

	// WriteEndpointsCapability is whether we have enabled write APIs on this server.
	WriteEndpointsCapability = "write_endpoints"

	// ChatCapability is whether this sippy instance is configured to proxy chat requests to sippy-chat service.
	ChatCapability = "chat"
)
