package service

import (
	"sync"
	"testing"
)

func TestGetNextServerIndex(t *testing.T) {
	tests := []struct {
		name           string
		outbound       string
		serverCount    int
		callCount      int
		expectIndices  []int
	}{
		{
			name:          "Single server",
			outbound:      "proxy",
			serverCount:   1,
			callCount:     5,
			expectIndices: []int{0, 0, 0, 0, 0},
		},
		{
			name:          "Two servers round-robin",
			outbound:      "proxy",
			serverCount:   2,
			callCount:     6,
			expectIndices: []int{0, 1, 0, 1, 0, 1},
		},
		{
			name:          "Three servers round-robin",
			outbound:      "proxy",
			serverCount:   3,
			callCount:     9,
			expectIndices: []int{0, 1, 2, 0, 1, 2, 0, 1, 2},
		},
		{
			name:          "Zero servers",
			outbound:      "proxy",
			serverCount:   0,
			callCount:     3,
			expectIndices: []int{0, 0, 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ResetIndex(tt.outbound)

			indices := make([]int, 0, tt.callCount)
			for i := 0; i < tt.callCount; i++ {
				idx := GetNextServerIndex(tt.outbound, tt.serverCount)
				indices = append(indices, idx)
			}

			if len(indices) != len(tt.expectIndices) {
				t.Errorf("Expected %d indices, got %d", len(tt.expectIndices), len(indices))
			}

			for i, expected := range tt.expectIndices {
				if i >= len(indices) {
					break
				}
				if indices[i] != expected {
					t.Errorf("Call %d: expected index %d, got %d", i, expected, indices[i])
				}
			}
		})
	}
}

func TestResetIndex(t *testing.T) {
	outbound := "test-outbound"

	GetNextServerIndex(outbound, 3)
	GetNextServerIndex(outbound, 3)
	GetNextServerIndex(outbound, 3)

	ResetIndex(outbound)

	idx := GetNextServerIndex(outbound, 3)
	if idx != 0 {
		t.Errorf("After reset, expected index 0, got %d", idx)
	}
}

func TestLoadBalancerConcurrently(t *testing.T) {
	outbound := "concurrent-test"
	serverCount := 5
	numGoroutines := 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	results := make([]int, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			results[idx] = GetNextServerIndex(outbound, serverCount)
		}(i)
	}

	wg.Wait()

	for i, idx := range results {
		if idx < 0 || idx >= serverCount {
			t.Errorf("Result[%d] = %d is out of range [0, %d)", i, idx, serverCount)
		}
	}
}

func TestGetServerIndices(t *testing.T) {
	GetNextServerIndex("outbound1", 3)
	GetNextServerIndex("outbound1", 3)
	GetNextServerIndex("outbound2", 2)

	indices := GetServerIndices()

	if len(indices) < 2 {
		t.Errorf("Expected at least 2 outbound indices, got %d", len(indices))
	}

	if _, ok := indices["outbound1"]; !ok {
		t.Error("Expected outbound1 in indices")
	}

	if _, ok := indices["outbound2"]; !ok {
		t.Error("Expected outbound2 in indices")
	}

	ResetIndex("outbound1")
	ResetIndex("outbound2")
}