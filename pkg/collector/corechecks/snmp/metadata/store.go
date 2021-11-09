package metadata

import (
	"github.com/DataDog/datadog-agent/pkg/collector/corechecks/snmp/valuestore"
	"github.com/DataDog/datadog-agent/pkg/util/log"
)

// Store MetadataStore stores metadata scalarValues
type Store struct {
	scalarValues map[string]valuestore.ResultValue

	// map[<FIELD>][<index>]valuestore.ResultValue
	columnValues map[string]map[string]valuestore.ResultValue

	// map[<RESOURCE>][<index>][]<TAG>
	resourceIDTags map[string]map[string][]string
}

// NewMetadataStore returns a new metadata Store
func NewMetadataStore() *Store {
	return &Store{
		scalarValues:   make(map[string]valuestore.ResultValue),
		columnValues:   make(map[string]map[string]valuestore.ResultValue),
		resourceIDTags: make(map[string]map[string][]string),
	}
}

// AddScalarValue add scalar value to metadata store
func (s Store) AddScalarValue(field string, value valuestore.ResultValue) {
	s.scalarValues[field] = value
}

// AddColumnValue add column value to metadata store
func (s Store) AddColumnValue(field string, index string, value valuestore.ResultValue) {
	column, ok := s.columnValues[field]
	if !ok {
		column = make(map[string]valuestore.ResultValue)
		s.columnValues[field] = column
	}
	column[index] = value
}

// GetColumnAsString get column value as string
func (s Store) GetColumnAsString(field string, index string) string {
	column, ok := s.columnValues[field]
	if !ok {
		// TODO: log error?
		return ""
	}
	value, ok := column[index]
	if !ok {
		// TODO: log error?
		return ""
	}
	strVal, err := value.ToString()
	if err != nil {
		log.Debugf("error converting value string `%v`: %s", value, err)
		return ""
	}
	return strVal
}

// GetColumnAsFloat get column value as float
func (s Store) GetColumnAsFloat(field string, index string) float64 {
	column, ok := s.columnValues[field]
	if !ok {
		// TODO: log error?
		return 0
	}
	value, ok := column[index]
	if !ok {
		// TODO: log error?
		return 0
	}
	strVal, err := value.ToFloat64()
	if err != nil {
		log.Debugf("error converting value to float `%v`: %s", value, err)
		return 0
	}
	return strVal
}

// GetScalarAsString get scalar value as string
func (s Store) GetScalarAsString(field string) string {
	value, ok := s.scalarValues[field]
	if !ok {
		// TODO: log error?
		return ""
	}
	strVal, err := value.ToString()
	if err != nil {
		log.Debugf("error parsing value `%v`: %s", value, err)
		return ""
	}
	return strVal
}

// GetColumnIndexes get column indexes for a field
func (s Store) GetColumnIndexes(field string) []string {
	column, ok := s.columnValues[field]
	if !ok {
		return nil
	}
	var indexes []string
	for key := range column {
		indexes = append(indexes, key)
	}
	return indexes
}

// GetIDTags get idTags for a specific resource and index
func (s Store) GetIDTags(resource string, index string) []string {
	resTags, ok := s.resourceIDTags[resource]
	if !ok {
		return nil
	}
	tags, ok := resTags[index]
	if !ok {
		return nil
	}
	return tags
}

// AddIDTags add idTags for a specific resource and index
func (s Store) AddIDTags(resource string, index string, tags []string) {
	indexToTags, ok := s.resourceIDTags[resource]
	if !ok {
		indexToTags = make(map[string][]string)
		s.resourceIDTags[resource] = indexToTags
	}
	s.resourceIDTags[resource][index] = append(s.resourceIDTags[resource][index], tags...)
}
