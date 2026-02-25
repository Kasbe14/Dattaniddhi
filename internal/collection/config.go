package collection

import (
	"github.com/Kasbe14/Dattaniddhi/internal/types"
)

// configuration for the collection
type CollectionConfig struct {
	Name      string
	Dimension int
	Metric    types.SimilarityMetric
	IndexType types.IndexType
	DataType  types.DataType
	ModelName string
}

// constructor
func NewCollectionConfig(
	name string,
	dim int,
	metric types.SimilarityMetric,
	idxType types.IndexType,
	daType types.DataType,
	modName string,
) (CollectionConfig, error) {
	if name == "" {
		return CollectionConfig{}, ErrInvalidCollectionName
	}
	if dim <= 0 {
		return CollectionConfig{}, ErrInvalidDimension
	}
	switch metric {
	case types.Cosine, types.Dot, types.Euclidean:
	//ok valid input
	default:
		return CollectionConfig{}, ErrInvalidMetric
	}
	switch idxType {
	case types.LinearIndex, types.HNSWIndex, types.IVFIndex, types.PQIndex:
	//ok valid input
	default:
		return CollectionConfig{}, ErrInvalidIndexType
	}
	switch daType {
	case types.Audio, types.Image, types.Text, types.Video:
	//ok valid input
	default:
		return CollectionConfig{}, ErrInvalidDataType
	}
	if modName == "" {
		return CollectionConfig{}, ErrInvalidModelName
	}

	return CollectionConfig{
		Name:      name,
		Dimension: dim,
		Metric:    metric,
		IndexType: idxType,
		DataType:  daType,
		ModelName: modName,
	}, nil
}
