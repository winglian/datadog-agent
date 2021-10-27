package metadata

import (
	"github.com/DataDog/datadog-agent/pkg/collector/corechecks/snmp/valuestore"
	"github.com/DataDog/datadog-agent/pkg/util/log"
)

// Store MetadataStore stores metadata values
type Store struct {
	values map[string]valuestore.ResultValue
}

func NewMetadataStore() *Store {
	return &Store{make(map[string]valuestore.ResultValue)}
}

func (s Store) Add(field string, value valuestore.ResultValue) {
	s.values[field] = value
}


func (s Store) Get(field string) valuestore.ResultValue {
	return s.values[field]
}


func (s Store) GetString(field string) string {
	value, ok := s.values[field]
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

