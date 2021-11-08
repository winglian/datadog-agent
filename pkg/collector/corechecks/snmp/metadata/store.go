package metadata

import (
	"github.com/DataDog/datadog-agent/pkg/collector/corechecks/snmp/valuestore"
	"github.com/DataDog/datadog-agent/pkg/util/log"
)

// ResourceIndex is a map of resource to field used for index
// TODO: make private
var ResourceIndex = map[string]string{
	"interface": IfNameOID,
}

// Store MetadataStore stores metadata scalarValues
type Store struct {
	scalarValues map[string]valuestore.ResultValue

	// map[<FIELD>][<index>]valuestore.ResultValue
	columnValues map[string]map[string]valuestore.ResultValue

	// map[<RESOURCE>][<index>][]<TAG>
	resourceIdTags map[string]map[string][]string
}

func NewMetadataStore() *Store {
	return &Store{
		scalarValues:   make(map[string]valuestore.ResultValue),
		columnValues:   make(map[string]map[string]valuestore.ResultValue),
		resourceIdTags: make(map[string]map[string][]string),
	}
}

func (s Store) AddScalarValue(field string, value valuestore.ResultValue) {
	s.scalarValues[field] = value
}

func (s Store) AddColumnValue(field string, index string, value valuestore.ResultValue) {
	column, ok := s.columnValues[field]
	if !ok {
		column = make(map[string]valuestore.ResultValue)
		s.columnValues[field] = column
	}
	column[index] = value
}

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

func (s Store) GetIdTags(resource string, index string) []string {
	resTags, ok := s.resourceIdTags[resource]
	if !ok {
		return nil
	}
	tags, ok := resTags[index]
	if !ok {
		return nil
	}
	return tags
}

func (s Store) AddIdTags(resource string, index string, tags []string) {
	indexToTags, ok := s.resourceIdTags[resource]
	if !ok {
		indexToTags = make(map[string][]string)
		s.resourceIdTags[resource] = indexToTags
	}
	s.resourceIdTags[resource][index] = append(s.resourceIdTags[resource][index], tags...)
}
