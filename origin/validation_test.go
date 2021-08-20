package origin

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidation(t *testing.T) {

	// empty origin
	{
		o := Origin{}
		err := o.Validate()
		assert.NoError(t, err)
	}

	// valid non-zero origin
	{
		o := Origin{MetricType: Origin_metric_type_dist_aggr}
		err := o.Validate()
		assert.NoError(t, err)
	}

	// invalid origin
	{
		o := Origin{MetricType: Origin_MetricType(math.MaxInt32 - 1)}
		err := o.Validate()
		assert.Error(t, err)
	}

}
