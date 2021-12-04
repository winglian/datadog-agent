package uptane

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/theupdateframework/go-tuf/data"
)

type metaPath struct {
	role       role
	version    uint64
	versionSet bool
}

func parseMetaPath(rawMetaPath string) (metaPath, error) {
	splitRawMetaPath := strings.SplitN(rawMetaPath, ".", 3)
	if len(splitRawMetaPath) != 2 && len(splitRawMetaPath) != 3 {
		return metaPath{}, fmt.Errorf("invalid metadata path '%s'", rawMetaPath)
	}
	suffix := splitRawMetaPath[len(splitRawMetaPath)-1]
	if suffix != "json" {
		return metaPath{}, fmt.Errorf("invalid metadata path (suffix) '%s'", rawMetaPath)
	}
	rawRole := splitRawMetaPath[len(splitRawMetaPath)-2]
	if rawRole == "" {
		return metaPath{}, fmt.Errorf("invalid metadata path (role) '%s'", rawMetaPath)
	}
	if len(splitRawMetaPath) == 2 {
		return metaPath{
			role: role(rawRole),
		}, nil
	}
	rawVersion, err := strconv.ParseUint(splitRawMetaPath[0], 10, 64)
	if err != nil {
		return metaPath{}, fmt.Errorf("invalid metadata path (version) '%s': %w", rawMetaPath, err)
	}
	return metaPath{
		role:       role(rawRole),
		version:    rawVersion,
		versionSet: true,
	}, nil
}

func rootVersion(rawRoot json.RawMessage) (uint64, error) {
	var signed data.Signed
	err := json.Unmarshal(rawRoot, &signed)
	if err != nil {
		return 0, err
	}
	var root data.Root
	err = json.Unmarshal(signed.Signed, &root)
	if err != nil {
		return 0, err
	}
	return uint64(root.Version), nil
}
