package version

func (svc *versionSvc) GetMaxVersion(versions []string) (string, error) {
	var maxVersion compareVersion
	var err error
	for _, strVersion := range versions {
		if maxVersion.Main == "" {
			maxVersion, err = svc.GetCompareVersion(strVersion, "")
			if err != nil {
				return "", err
			}
			continue
		}

		version, err := svc.GetCompareVersion(strVersion, "")
		if err != nil {
			return "", err
		}

		result, err := svc.CompareVersion(maxVersion, version, false)
		if err != nil {
			return "", err
		}
		if result == 0 {
			maxVersion = version
		}
	}
	return maxVersion.Original, nil
}

func (svc *versionSvc) GetMinVersion(versions []string) (string, error) {
	var minVersion compareVersion
	var err error
	for _, strVersion := range versions {
		if minVersion.Main == "" {
			minVersion, err = svc.GetCompareVersion(strVersion, "")
			if err != nil {
				return "", err
			}
			continue
		}

		version, err := svc.GetCompareVersion(strVersion, "")
		if err != nil {
			return "", err
		}

		result, err := svc.CompareVersion(minVersion, version, false)
		if err != nil {
			return "", err
		}
		if result == 2 {
			minVersion = version
		}
	}
	return minVersion.Original, nil
}
