package checkconfig

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
			"serial_number": {
				OID:  "1.3.6.1.4.1.3375.2.1.3.3.3.0",
				Name: "sysGeneralChassisSerialNum",
			},
			"sys_object_id": {
				OID:  "1.3.6.1.2.1.1.2.0",
				Name: "sysObjectID",
			},
		},
	},
	"interface": {
		Fields: map[string]SymbolConfig{
			"admin_status": {
				OID:  "1.3.6.1.2.1.2.2.1.7",
				Name: "ifAdminStatus",
			},
			"alias": {
				OID:  "1.3.6.1.2.1.31.1.1.1.18",
				Name: "ifAlias",
			},
			"description": {
				OID:  "1.3.6.1.2.1.2.2.1.2",
				Name: "ifDescr",
			},
			"mac_address": {
				OID:  "1.3.6.1.2.1.2.2.1.6",
				Name: "ifPhysAddress",
			},
			"name": {
				OID:  "1.3.6.1.2.1.31.1.1.1.1",
				Name: "ifName",
			},
			"oper_status": {
				OID:  "1.3.6.1.2.1.2.2.1.8",
				Name: "ifOperStatus",
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
