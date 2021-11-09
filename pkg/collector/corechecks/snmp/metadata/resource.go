package metadata

// resourceIndex is a map of fields to be used to get a list of indexes for a specific resource
var resourceIndex = map[string]string{
	"interface": IfNameOID,
}

// GetIndexOIDForResource returns the index OID for a specific resource
func GetIndexOIDForResource(resource string) string {
	return resourceIndex[resource]
}
