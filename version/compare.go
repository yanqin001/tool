package version

import (
	"strconv"
	"strings"
)

// v1为指定版本, v2为版本表达式版本
// 0 < , 1 = , 2 >， 7.0.3.1 > 7.0.3;0.15.1 > 0.15; 5 不满足
func (svc *versionSvc) Compare(v1, v2 compareVersion, matchMain bool) (int, error) {
	if v1.Original == v2.Original {
		return 1, nil
	}

	// 字符串版本只匹配是否相等（上面一步已匹配）
	if v1.Type == 2 || v2.Type == 2 {
		return 5, nil
	}

	result, err := svc.CompareStringVersion(v1.Main, v2.Main)
	if err != nil {
		return result, err
	}

	if result != 1 {
		return result, nil
	}

	// MainVersion没有比较出来，开始比较MainSupplement
	// 7.0.1 > 7.0.1-RC1 但是openssl除外，openssl 1.1.1a > 1.1.1 > 1.1.1-pre9
	result = svc.StringCompare(v1.MainSupplement, v2.MainSupplement)
	if result != 1 {
		return result, nil
	}

	if matchMain {
		return result, nil
	}
	// 开始比较pre
	if v1.Pre == "" && v2.Pre != "" {
		return 2, nil
	}

	if v1.Pre != "" && v2.Pre == "" {
		return 0, nil
	}

	// 字符串比较，beta > alpha
	if v1.Pre > v2.Pre {
		return 2, nil
	} else if v1.Pre < v2.Pre {
		return 0, nil
	}

	// 比较Additional
	if v1.Additional == "" && v2.Additional == "" {
		return 1, nil
	}

	if v1.Additional == "" && v2.Additional != "" {
		return 2, nil
	}

	if v1.Additional != "" && v2.Additional == "" {
		return 0, nil
	}
	return svc.CompareStringVersion(v1.Additional, v2.Additional)
}

func (svc *versionSvc) CompareStringVersion(stringVersion1, stringVersion2 string) (int, error) {
	parts1 := strings.Split(stringVersion1, ".")
	parts2 := strings.Split(stringVersion2, ".")
	v2Length := len(parts2)
	for i, part1 := range parts1 {
		// 7.0.1 > 7.0; 7.1.0 = 7.1
		if i == v2Length {
			for _, p := range parts1[i:] {
				v1IntVersion, err := strconv.Atoi(p)
				if err != nil {
					return 0, err
				}
				if v1IntVersion != 0 {
					return 2, nil
				}
			}
			return 1, nil
		}

		v1IntVersion, err := strconv.Atoi(part1)
		if err != nil {
			return 0, err
		}
		v2IntVersion, err := strconv.Atoi(parts2[i])
		if err != nil {
			return 0, err
		}
		if v1IntVersion > v2IntVersion {
			return 2, nil
		} else if v1IntVersion < v2IntVersion {
			return 0, nil
		}
	}

	// 7.0 < 7.0.1; 7.1 == 7.1.0 == 7.1.0.0
	if len(parts1) < v2Length {
		for _, p := range parts2[len(parts1):] {
			v2IntVersion, err := strconv.Atoi(p)
			if err != nil {
				return 0, err
			}
			if v2IntVersion != 0 {
				return 0, nil
			}
		}
		return 1, nil
	}
	return 1, nil
}

// 0 < , 1 = , 2 >
func (svc *versionSvc) StringCompare(str1, str2 string) int {
	if str1 > str2 {
		return 2
	} else if str1 == str2 {
		return 1
	}
	return 0
}
