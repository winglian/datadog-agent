package checkconfig

// LegacyMetadataConfig contain metadata config used for backward compatibility
// When users have their own copy of _base.yaml and _generic_if.yaml files
// they won't have the new profile based metadata definitions for device and interface resources
// The LegacyMetadataConfig is used as fallback to provide metadata definitions for those resources.
var LegacyMetadataConfig = MetadataConfig{
	"device": {
		Fields: map[string]SymbolConfig{
			"description": {
				OID:  "1.3.6.1.2.1.1.1.0",
				Name: "sysDescr",
			},
			"name": {
				OID:  "1.3.6.1.2.1.1.5.0",
				Name: "sysName",
			},
			"sys_object_id": {
				OID:  "1.3.6.1.2.1.1.2.0",
				Name: "sysObjectID",
			},
		},
	},
	"interface": {
		Fields: map[string]SymbolConfig{
			"name": {
				OID:  "1.3.6.1.2.1.31.1.1.1.1",
				Name: "ifName",
			},
			"description": {
				OID:  "1.3.6.1.2.1.2.2.1.2",
				Name: "ifDescr",
			},
			"admin_status": {
				OID:  "1.3.6.1.2.1.2.2.1.7",
				Name: "ifAdminStatus",
			},
			"oper_status": {
				OID:  "1.3.6.1.2.1.2.2.1.8",
				Name: "ifOperStatus",
			},
			"alias": {
				OID:  "1.3.6.1.2.1.31.1.1.1.18",
				Name: "ifAlias",
			},
			"mac_address": {
				OID:  "1.3.6.1.2.1.2.2.1.6",
				Name: "ifPhysAddress",
			},
		},
		IDTags: MetricTagConfigList{
			{
				Tag: "interface",
				Column: SymbolConfig{
					OID:  "1.3.6.1.2.1.31.1.1.1.1",
					Name: "ifName",
				},
			},
		},
	},
}

// MetadataConfig holds configs for a metadata
type MetadataConfig map[string]MetadataResourceConfig

// MetadataResourceConfig holds configs for a metadata resource
type MetadataResourceConfig struct {
	Fields map[string]SymbolConfig `yaml:"fields"`

	// TODO: Implement tags
	//       Should we use the same structure as for metrics ?
	IDTags MetricTagConfigList `yaml:"id_tags"`
}

// NewMetadataResourceConfig returns a new metadata resource config
func NewMetadataResourceConfig() MetadataResourceConfig {
	return MetadataResourceConfig{}
}

// IsMetadataResourceWithScalarOids returns true if the resource is based on scalar OIDs
// at the moment, we only expect "device" resource to be based on scalar OIDs
func IsMetadataResourceWithScalarOids(resource string) bool {
	return resource == "device"
}
