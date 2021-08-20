package origin

import "errors"

var (
	// ErrUnknownMetricType is the error returned when the origin metric type is not one of the values allowed
	ErrUnknownMetricType = errors.New("unknown origin metric type")
)

// Validate validates the origin of a point
func (o Origin) Validate() error {
	originMetricType := o.GetMetricType()
	if _, foundMetricType := Origin_MetricType_name[int32(originMetricType)]; !foundMetricType {
		return ErrUnknownMetricType
	}

	return nil
}
