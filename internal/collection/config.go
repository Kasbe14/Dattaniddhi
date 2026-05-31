package collection

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Kasbe14/Dattaniddhi/internal/types"
)

const collectionConfigVersion uint32 = 1

// configuration for the collection
// existence of config.json
// becomes authoritative collection existence signal
type CollectionConfig struct {
	Name      string
	Dimension int
	Metric    types.SimilarityMetric
	IndexType types.IndexType
	DataType  types.DataType
	ModelName string
	Version   uint32
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
		Version:   collectionConfigVersion,
	}, nil
}

func saveConfig(cfg CollectionConfig, path string) error {
	jsonData, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	fullpath := filepath.Join(path /*,dbName*/, cfg.Name)
	err = os.MkdirAll(fullpath, 0755)
	if err != nil {
		return fmt.Errorf("failed to make directory for config file: %w", err)
	}
	filePath := filepath.Join(fullpath, "config.json")
	err = os.WriteFile(filePath, jsonData, 0644)
	if err != nil {
		return fmt.Errorf("failed to write json data: %w", err)
	}
	return nil
}

func loadConfig(path, colName string) (*CollectionConfig, error) {
	filePath := filepath.Join(path, colName, "config.json")

	_, err := os.Stat(filePath)
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrCollectionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to laod config file: %w", err)
	}
	jsonfile, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read json file: %w", err)
	}
	var config CollectionConfig
	err = json.Unmarshal(jsonfile, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal json: %w", err)
	}
	if config.Name != colName {
		return nil, fmt.Errorf("invalid collection name")
	}
	if config.Dimension <= 0 {
		return nil, fmt.Errorf("invalid collection dimension")
	}
	switch config.Metric {
	case types.Cosine, types.Dot, types.Euclidean:
		//do nothing correct type
	default:
		return nil, fmt.Errorf("invalid collection metric type")
	}
	switch config.IndexType {
	case types.HNSWIndex, types.IVFIndex, types.PQIndex, types.LinearIndex:
		//do nthign valid data
	default:
		return nil, fmt.Errorf("invalid collection index type")
	}
	switch config.DataType {
	case types.Audio, types.Image, types.Text, types.Video:
		//do nothing valid data
	default:
		return nil, fmt.Errorf("invalid collection data type")
	}
	if collectionConfigVersion != config.Version {
		return nil, fmt.Errorf("invalid collection config verison")
	}
	return &config, nil
}
