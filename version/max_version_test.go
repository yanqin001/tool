package version

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_versionSvc_GetMaxVersion(t *testing.T) {
	testCasas := []struct {
		Name     string
		Versions []string
		expected string
	}{
		{
			Name:     "t1",
			Versions: []string{"1.0.1", "3.0.2", "2.0.3", "3.0.1"},
			expected: "3.0.2",
		},
	}
	for _, ts := range testCasas {
		svc := NewVersionSvc()
		result, err := svc.GetMaxVersion(ts.Versions)
		assert.NoError(t, err)
		assert.Equal(t, ts.expected, result)
	}
}

func Test_versionSvc_GetMinVersion(t *testing.T) {
	testCasas := []struct {
		Name     string
		Versions []string
		expected string
	}{
		{
			Name:     "t1",
			Versions: []string{"1.0.1", "3.0.2", "2.0.3", "3.0.1"},
			expected: "1.0.1",
		},
	}
	for _, ts := range testCasas {
		svc := NewVersionSvc()
		result, err := svc.GetMinVersion(ts.Versions)
		assert.NoError(t, err)
		assert.Equal(t, ts.expected, result)
	}
}
