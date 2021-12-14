package config

import "encoding/json"

type versionCustom struct {
	Version uint64 `json:"version"`
}

func targetVersion(custom json.RawMessage) (uint64, error) {
	var version versionCustom
	err := json.Unmarshal(custom, &version)
	if err != nil {
		return 0, err
	}
	return version.Version, nil
}
