package version

import (
	"fmt"
	"sort"
)

// GetClosestVersion 从versions中找出和version最接近的版本
func (svc *versionSvc) GetClosestVersion(version string, versions []string) (string, error) {
	if len(versions) == 0 {
		return "", fmt.Errorf("versions list is empty")
	}

	// 先检查是否有完全匹配的版本
	for _, v := range versions {
		result, err := svc.Compare(version, v)
		if err != nil {
			continue
		}
		if result == 1 { // 相等
			return v, nil
		}
	}

	// 对versions排序（复制一份避免修改原切片）
	sorted := make([]string, len(versions))
	copy(sorted, versions)
	var sortErr error
	sort.SliceStable(sorted, func(i, j int) bool {
		result, err := svc.Compare(sorted[i], sorted[j])
		if err != nil {
			sortErr = err
		}
		return result == 0 // sorted[i] < sorted[j]
	})
	if sortErr != nil {
		return "", sortErr
	}

	// 找到version在排序后列表中的插入位置
	insertIdx := sort.Search(len(sorted), func(i int) bool {
		result, _ := svc.Compare(sorted[i], version)
		return result >= 1 // sorted[i] >= version
	})

	// 边界情况：version小于所有版本，返回最小版本
	if insertIdx == 0 {
		return sorted[0], nil
	}
	// 边界情况：version大于所有版本，返回最大版本
	if insertIdx == len(sorted) {
		return sorted[len(sorted)-1], nil
	}

	// version在两个版本之间，比较左右两侧哪个更接近
	// 通过比较version与左右邻居的版本段差异来决定
	left := sorted[insertIdx-1]
	right := sorted[insertIdx]

	leftDist := svc.versionDistance(version, left)
	rightDist := svc.versionDistance(version, right)

	if rightDist <= leftDist {
		return right, nil
	}
	return left, nil
}

// versionDistance 计算两个版本之间的近似距离
// 返回值越小表示越接近
func (svc *versionSvc) versionDistance(v1, v2 string) int {
	cv1, err := svc.GetCompareVersion(v1, "")
	if err != nil {
		return int(^uint(0) >> 1) // MaxInt
	}
	cv2, err := svc.GetCompareVersion(v2, "")
	if err != nil {
		return int(^uint(0) >> 1)
	}

	return segmentDistance(cv1.Main, cv2.Main)
}

// segmentDistance 计算两个主版本号之间的段距离
func segmentDistance(main1, main2 string) int {
	parts1 := splitVersionParts(main1)
	parts2 := splitVersionParts(main2)

	// 对齐长度
	maxLen := len(parts1)
	if len(parts2) > maxLen {
		maxLen = len(parts2)
	}
	for len(parts1) < maxLen {
		parts1 = append(parts1, 0)
	}
	for len(parts2) < maxLen {
		parts2 = append(parts2, 0)
	}

	// 计算加权距离，高位权重更大
	distance := 0
	weight := 1
	for i := maxLen - 1; i >= 0; i-- {
		diff := parts1[i] - parts2[i]
		if diff < 0 {
			diff = -diff
		}
		distance += diff * weight
		weight *= 1000
	}
	return distance
}

// splitVersionParts 将版本号字符串按.分割为整数数组
func splitVersionParts(v string) []int {
	if v == "" {
		return []int{0}
	}
	parts := []int{}
	current := 0
	hasDigit := false
	for _, ch := range v {
		if ch == '.' {
			parts = append(parts, current)
			current = 0
			hasDigit = false
		} else if ch >= '0' && ch <= '9' {
			current = current*10 + int(ch-'0')
			hasDigit = true
		}
	}
	if hasDigit || len(parts) == 0 {
		parts = append(parts, current)
	}
	return parts
}
