package version

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_versionSvc_InExp(t *testing.T) {
	testCasas := []struct {
		Name     string
		Ve       VersionExp
		expected bool
	}{
		{
			Name: "t1",
			Ve: VersionExp{
				Version:    "0.9.1",
				Expression: "(,0.9.5]",
			},
			expected: true,
		},
		{
			Name: "t2",
			Ve: VersionExp{
				Version:    "6.7",
				Expression: "[6.7-rc1,6.7)",
			},
			expected: false,
		},
		{
			Name: "t3",
			Ve: VersionExp{
				Version:    "6.7-rc2",
				Expression: "[6.7-rc1,6.7)",
			},
			expected: true,
		},
		{
			Name: "t4",
			Ve: VersionExp{
				Version:    "6.7-rc1",
				Expression: "[6.7-rc1,6.7)",
			},
			expected: true,
		},
		{
			Name: "t5",
			Ve: VersionExp{
				Version:    "6.7.1-rc1",
				Expression: "[6.7-rc1,6.7)",
			},
			expected: false,
		},
		{
			Name: "t1",
			Ve: VersionExp{
				Version:    "1.1.1a",
				Expression: "[1.1.1, 1.1.1t)",
			},
			expected: true,
		},
	}
	for _, ts := range testCasas {
		svc := NewVersionSvc()
		result, err := svc.InExp(ts.Ve)
		assert.NoError(t, err)
		assert.Equal(t, ts.expected, result)
	}
}
