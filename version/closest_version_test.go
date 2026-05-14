package version

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_versionSvc_GetClosestVersion(t *testing.T) {
	testCases := []struct {
		Name     string
		Version  string
		Versions []string
		Expected string
		HasError bool
	}{
		{
			Name:     "完全匹配",
			Version:  "1.2.3",
			Versions: []string{"1.2.1", "1.2.3", "1.2.5"},
			Expected: "1.2.3",
		},
		{
			Name:     "最接近的较高版本",
			Version:  "1.2.5",
			Versions: []string{"1.2.3", "1.2.6", "1.3.0"},
			Expected: "1.2.6",
		},
		{
			Name:     "最接近的较低版本",
			Version:  "1.2.5",
			Versions: []string{"1.2.4", "1.2.0", "1.3.0"},
			Expected: "1.2.4",
		},
		{
			Name:     "小于所有版本",
			Version:  "1.0.0",
			Versions: []string{"1.2.3", "2.0.0", "3.0.0"},
			Expected: "1.2.3",
		},
		{
			Name:     "大于所有版本",
			Version:  "5.0.0",
			Versions: []string{"1.2.3", "2.0.0", "3.0.0"},
			Expected: "3.0.0",
		},
		{
			Name:     "只有一个版本",
			Version:  "1.5.0",
			Versions: []string{"2.0.0"},
			Expected: "2.0.0",
		},
		{
			Name:     "两个等距版本取较高版本",
			Version:  "1.2.5",
			Versions: []string{"1.2.4", "1.2.6"},
			Expected: "1.2.6",
		},
		{
			Name:     "多段版本号比较",
			Version:  "2.3.4",
			Versions: []string{"1.0.0", "2.3.3", "2.3.5", "3.0.0"},
			Expected: "2.3.5",
		},
		{
			Name:     "主版本号差异优先",
			Version:  "2.0.0",
			Versions: []string{"1.9.9", "3.0.0"},
			Expected: "3.0.0", // 3.0.0与2.0.0的加权距离更小(仅主版本差1)，1.9.9还有minor/patch偏移
		},
		{
			Name:     "空版本列表报错",
			Version:  "1.0.0",
			Versions: []string{},
			HasError: true,
		},
	}

	svc := NewVersionSvc()
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			result, err := svc.GetClosestVersion(tc.Version, tc.Versions)
			if tc.HasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.Expected, result)
			}
		})
	}
}
