package collection

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Kasbe14/Dattaniddhi/internal/types"
)

func TestNewCollectionConfig(t *testing.T) {
	tests := []struct {
		name           string
		collectionName string
		dim            int
		metric         types.SimilarityMetric
		idxType        types.IndexType
		dataType       types.DataType
		modelName      string
		expectError    bool
		errorType      string // Description of which error to check for
	}{
		{
			name:           "Success: Valid Configuration",
			collectionName: "my-vectors",
			dim:            128,
			metric:         types.Cosine,
			idxType:        types.LinearIndex,
			dataType:       types.Text,
			modelName:      "bert-base",
			expectError:    false,
		},
		{
			name:           "Failure: Empty Collection Name",
			collectionName: "",
			dim:            128,
			metric:         types.Cosine,
			idxType:        types.LinearIndex,
			dataType:       types.Text,
			modelName:      "bert-base",
			expectError:    true,
			errorType:      "ErrInvalidCollectionName",
		},
		{
			name:           "Failure: Zero Dimension",
			collectionName: "my-vectors",
			dim:            0,
			metric:         types.Cosine,
			idxType:        types.LinearIndex,
			dataType:       types.Text,
			modelName:      "bert-base",
			expectError:    true,
			errorType:      "ErrInvalidDimension",
		},
		{
			name:           "Failure: Negative Dimension",
			collectionName: "my-vectors",
			dim:            -5,
			metric:         types.Cosine,
			idxType:        types.LinearIndex,
			dataType:       types.Text,
			modelName:      "bert-base",
			expectError:    true,
			errorType:      "ErrInvalidDimension",
		},
		{
			name:           "Failure: Invalid Metric",
			collectionName: "my-vectors",
			dim:            128,
			metric:         types.SimilarityMetric(99), // Cast invalid int to enum
			idxType:        types.LinearIndex,
			dataType:       types.Text,
			modelName:      "bert-base",
			expectError:    true,
			errorType:      "ErrInvalidMetric",
		},
		{
			name:           "Failure: Invalid Index Type",
			collectionName: "my-vectors",
			dim:            128,
			metric:         types.Cosine,
			idxType:        types.IndexType(99),
			dataType:       types.Text,
			modelName:      "bert-base",
			expectError:    true,
			errorType:      "ErrInvalidIndexType",
		},
		{
			name:           "Failure: Invalid Data Type",
			collectionName: "my-vectors",
			dim:            128,
			metric:         types.Cosine,
			idxType:        types.LinearIndex,
			dataType:       types.DataType(99),
			modelName:      "bert-base",
			expectError:    true,
			errorType:      "ErrInvalidDataType",
		},
		{
			name:           "Failure: Empty Model Name",
			collectionName: "my-vectors",
			dim:            128,
			metric:         types.Cosine,
			idxType:        types.LinearIndex,
			dataType:       types.Text,
			modelName:      "",
			expectError:    true,
			errorType:      "ErrInvalidModelName",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: I assumed the typo 'NewCollectionConfiga' is fixed to 'NewCollectionConfig'
			cfg, err := NewCollectionConfig(
				tt.collectionName,
				tt.dim,
				tt.metric,
				tt.idxType,
				tt.dataType,
				tt.modelName,
			)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for %s, got nil", tt.name)
				}

				// --- ERROR CHECKING SECTION ---
				// LINE 130: Insert your specific error variable checks here.
				// Since I cannot see your error variables, I have provided the logic below.
				// Unwrap the comment blocks and replace 'YourErrVariable' with your actual variable.

				switch tt.errorType {
				case "ErrInvalidCollectionName":
					if err != ErrInvalidCollectionName {
						t.Errorf("Got %v,invalid collection name", err)
					}
				case "ErrInvalidDimension":
					if err != ErrInvalidDimension {
						t.Errorf("Got %v, invalid vector dimension", err)
					}
				case "ErrInvalidMetric":
					if err != ErrInvalidMetric {
						t.Errorf("Got %v, invalid similarity metric", err)
					}
				case "ErrInvalidIndexType":
					if err != ErrInvalidIndexType {
						t.Errorf("Got %v, invalid index type", err)
					}
				case "ErrInvalidDataType":
					if err != ErrInvalidDataType {
						t.Errorf("Got %v, invalid data type", err)
					}
				case "ErrInvalidModelName":
					if err != ErrInvalidModelName {
						t.Errorf("Got %v, invalid model name type", err)
					}
				}

			} else {
				if err != nil {
					t.Errorf("Unexpected error for %s: %v", tt.name, err)
				}
				// Verify the object was actually created correctly
				if cfg.Name != tt.collectionName {
					t.Errorf("Config Name mismatch: got %s, want %s", cfg.Name, tt.collectionName)
				}
				if cfg.Dimension != tt.dim {
					t.Errorf("Config Dimension mismatch: got %d, want %d", cfg.Dimension, tt.dim)
				}
			}
		})
	}
}

func TestConfig_Lifecycle_SaveAndLoad(t *testing.T) {
	rootDir := t.TempDir()

	originalCfg, _ := NewCollectionConfig(
		"test_config", 128, types.Cosine, types.LinearIndex, types.Text, "test-model",
	)

	// Save
	if err := saveConfig(originalCfg, rootDir); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Load
	loadedCfg, err := loadConfig(rootDir, "test_config")
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify equality
	if loadedCfg.Name != originalCfg.Name || loadedCfg.Dimension != originalCfg.Dimension || loadedCfg.Version != originalCfg.Version {
		t.Errorf("Config mismatch. Expected %+v, got %+v", originalCfg, loadedCfg)
	}
}

func TestConfig_Load_MissingFails(t *testing.T) {
	rootDir := t.TempDir()

	_, err := loadConfig(rootDir, "ghost_collection")
	if err == nil {
		t.Fatal("Expected error when loading non-existent config, got nil")
	}
}

func TestConfig_Load_Corruption(t *testing.T) {
	rootDir := t.TempDir()
	collName := "corrupted_coll"

	// Create directory manually
	collDir := filepath.Join(rootDir, collName)
	os.MkdirAll(collDir, 0755)

	// Write malformed JSON
	configPath := filepath.Join(collDir, "config.json")
	os.WriteFile(configPath, []byte(`{"Name": "corrupted_coll", "Dimension": "NOT_A_NUMBER"}`), 0644)

	_, err := loadConfig(rootDir, collName)
	if err == nil {
		t.Fatal("Expected error when loading malformed JSON config, got nil")
	}
}
