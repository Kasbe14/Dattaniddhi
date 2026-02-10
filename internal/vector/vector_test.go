package vector

import (
	"math"
	"testing"
)

// Invariant: The constructor must reject empty slices.
// Contract: Input slice length must be greater than 0.
// Contract: Input slice length must equal the provided 'dim' argument.
// Contract: Input values must be valid (not NaN, not Inf).
// Contract: Input vector must have non-zero magnitude (enforced by Normalize).
// Post-condition: The resulting vector values must be normalized (Unit Vector).
func TestNewVector_ContractsAndLogic(t *testing.T) {
	tests := []struct {
		name          string
		vecValues     []float32
		dim           int
		expectError   bool
		errorContains string // Optional: substring check for specific error messages
	}{
		{
			name:        "Success: Valid Input",
			vecValues:   []float32{3, 4},
			dim:         2,
			expectError: false,
		},
		{
			name:          "Invariant Violation: Empty Vector",
			vecValues:     []float32{},
			dim:           0,
			expectError:   true,
			errorContains: "a vector must have atleast one dimension",
		},
		{
			name:          "Contract Violation: Dimension Mismatch",
			vecValues:     []float32{1, 2, 3},
			dim:           2,
			expectError:   true,
			errorContains: "number of vector values not equal to given dimension",
		},
		{
			name:        "Contract Violation: NaN Values",
			vecValues:   []float32{1.0, float32(math.NaN())},
			dim:         2,
			expectError: true,
			// Assuming validateValues returns a specific error
		},
		{
			name:        "Contract Violation: Infinite Values",
			vecValues:   []float32{float32(math.Inf(1)), 2.0},
			dim:         2,
			expectError: true,
		},
		{
			name:        "Contract Violation: Zero Magnitude",
			vecValues:   []float32{0, 0, 0},
			dim:         3,
			expectError: true,
			// Assuming Normalize returns an error for zero magnitude
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vec, err := NewVector(tt.vecValues, tt.dim)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected an error for %s, but got nil", tt.name)
				}
				if tt.errorContains != "" && err != nil {
					if !contains(err.Error(), tt.errorContains) { // helper function assumed
						t.Errorf("Expected error containing '%s', got '%v'", tt.errorContains, err)
					}
				}
			} else {
				if err != nil {
					t.Errorf("Did not expect an error for %s, but got: %v", tt.name, err)
				}
				if vec == nil {
					t.Errorf("Expected valid vector, got nil")
					return
				}

				// Post-condition Check: Verify Normalization
				// For input {3, 4}, magnitude is 5. Normalized should be {0.6, 0.8}.
				// 0.6*0.6 + 0.8*0.8 = 0.36 + 0.64 = 1.0
				expectedVec := []float32{0.6, 0.8}
				if !slicesApproxEqual(vec.values, expectedVec) {
					t.Errorf("Vector not normalized correctly. Got %v, want %v", vec.values, expectedVec)
				}
			}
		})
	}
}

// Invariant: The Vector must be Immutable.
// Contract: The internal state of the Vector must not be affected by modifications
// to the input slice after the constructor returns.
func TestNewVector_ImmutabilityInvariant(t *testing.T) {
	// Arrange
	inputValues := []float32{1, 0}
	dim := 2

	// Act
	vec, err := NewVector(inputValues, dim)
	if err != nil {
		t.Fatalf("Failed to create vector: %v", err)
	}

	// Capture the state of the vector immediately after creation
	initialVecValue := vec.values[0]

	// Maliciously modify the original input slice
	inputValues[0] = 999.99

	// Assert
	// The vector's internal value should remain what it was upon creation (normalized 1.0)
	// and NOT take the new value (999.99)
	if vec.values[0] != initialVecValue {
		t.Errorf("Immutability violation! Modifying the input slice changed the vector internal state. Got: %f, Expected: %f", vec.values[0], initialVecValue)
	}

	if vec.values[0] == 999.99 {
		t.Error("Critical: Vector holds a reference to the input slice rather than a copy/new slice.")
	}
}

// --- Helper functions for tests ---

func slicesApproxEqual(a, b []float32) bool {
	if len(a) != len(b) {
		return false
	}
	const epsilon = 1e-6
	for i := range a {
		if math.Abs(float64(a[i]-b[i])) > epsilon {
			return false
		}
	}
	return true
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[0:len(substr)] == substr // simplistic check, use strings.Contains in real code
}
