package vector

import (
	"errors"
	"math"
	"strconv" //for converions frm string to num and vice versa
)

func validateValues(values []float32) error {
	for i, v := range values {
		if math.IsNaN(float64(v)) {
			return errors.New("ERROR NaN VALUE : vector contains NaN value at index" + strconv.Itoa(i))
		}
		if math.IsInf(float64(v), 0) {
			return errors.New("ERROR Inf : vector contains Inf value at index " + strconv.Itoa(i))
		}
	}
	return nil
}
