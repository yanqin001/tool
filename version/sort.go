package version

import (
	"sort"
)

func (svc *versionSvc) Sort(versions []string) ([]string, error) {
	if err := svc.sortVersions(versions); err != nil {
		return nil, err
	}
	return versions, nil
}

func (svc *versionSvc) getVersions(vs []string) ([]compareVersion, error) {
	cvs := make([]compareVersion, 0, len(vs))
	for _, v := range vs {
		cv, err := svc.GetCompareVersion(v, "")
		if err != nil {
			return nil, err
		}
		cvs = append(cvs, cv)
	}
	return cvs, nil
}

func (svc *versionSvc) sortVersions(versions []string) error {
	var err1 error
	sort.SliceStable(versions, func(i, j int) bool {
		result, err := svc.Compare(versions[i], versions[j])
		if err != nil {
			err1 = err
		}
		return result == 0 // v[i] < v[j] 时返回 true
	})
	return err1
}
