package version

import (
	"github.com/dlclark/regexp2"
	"regexp"
	"strings"
)

func (svc *versionSvc) GetCompareVersion(v, tp string) (compareVersion, error) {
	cv, err := svc.getCompareVersion(v, tp)
	cv.Original = v
	return cv, err
}

func (svc *versionSvc) getCompareVersion(v, tp string) (compareVersion, error) {
	v = svc.FormatCharacter(v)
	if v == "" {
		return compareVersion{}, nil
	}
	cv, err := svc.GetNatureNumber(v)
	if err != nil || cv.Main != "" {
		return cv, err
	}

	if svc.IsStringVersion(v) {
		return svc.GetStringVersion(v), nil
	}

	if tp == "os" {
		return svc.GetOSVersion(v)
	}

	return svc.getNormalCompareVersion(v)
}

func (svc *versionSvc) FormatCharacter(v string) string {
	v = strings.TrimSpace(v)
	v = strings.ToLower(v)
	v = strings.ReplaceAll(v, "x86_64", "")
	v = strings.ReplaceAll(v, "~~", "-")
	v = strings.ReplaceAll(v, "~", "-")
	v = strings.ReplaceAll(v, "_", ".")
	v = strings.ReplaceAll(v, "+", "-")
	return v
}

// 匹配单个数字，如39
func (svc *versionSvc) GetNatureNumber(v string) (compareVersion, error) {
	regexpString := `(^\d+$)`
	reg := regexp2.MustCompile(regexpString, 0)
	match, err := reg.FindStringMatch(v)
	if err != nil {
		return compareVersion{}, err
	}
	if match == nil {
		return compareVersion{}, nil
	}
	groups := match.Groups()
	cv := compareVersion{
		Main: groups[1].Captures[0].String(),
		Type: 1,
	}
	return cv, nil
}

func (svc *versionSvc) IsStringVersion(s string) bool {
	if svc.IsCommitVersion(s) {
		return true
	}
	// 正则表达式：^[a-zA-Z]+[a-zA-Z0-9]*$ (至少一个字母，后面可以跟字母或数字)
	re := regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9]*$`)
	return re.MatchString(s)
}

// 直接将版本作为字符串处理
func (svc *versionSvc) GetStringVersion(v string) compareVersion {
	return compareVersion{
		Main: v,
		Type: 2,
	}
}

// 判断是否是commit_id类型的版本
func (svc *versionSvc) IsCommitVersion(v string) bool {
	regexpString := `^[a-zA-Z0-9]{40}$`
	reg := regexp2.MustCompile(regexpString, 0)
	match, err := reg.FindStringMatch(v)
	if err != nil {
		return false
	}
	if match == nil {
		return false
	}
	return true
}

func (svc *versionSvc) getNormalCompareVersion(v string) (compareVersion, error) {
	regexpString := `((\d+(\.\d+)+)([a-zA-Z]*)[\.\-]*([a-zA-Z]*|[\.\d]*(?<!\.))[\.\-]*([\d\.]*)(?<!\.)\.*([\.\-a-zA-Z0-9]*))`
	reg := regexp2.MustCompile(regexpString, 0)
	match, err := reg.FindStringMatch(v)
	if err != nil {
		return compareVersion{}, err
	}

	if match == nil {
		return svc.GetStringVersion(v), nil
	}
	groups := match.Groups()
	cv := compareVersion{
		Main:           groups[2].Captures[0].String(),
		MainSupplement: groups[4].Captures[0].String(),
		Pre:            groups[5].Captures[0].String(),
		Additional:     groups[6].Captures[0].String(),
		Type:           0,
	}
	return cv, nil
}

func (svc *versionSvc) GetOSVersion(v string) (compareVersion, error) {
	regexpString := `(\d+:)*(\d+(\.\d+)*)[\.\-]*([a-zA-Z]*|[\.\d]*(?<!\.))[\.\-]*([\d\.]*)(?<!\.)\.*([\.\-a-zA-Z0-9]*)`
	reg := regexp2.MustCompile(regexpString, 0)
	match, err := reg.FindStringMatch(v)
	if err != nil {
		return compareVersion{}, err
	}

	if match == nil {
		return svc.GetStringVersion(v), nil
	}
	groups := match.Groups()
	cv := compareVersion{
		Main:         groups[2].Captures[0].String(),
		Pre:          groups[4].Captures[0].String(),
		Additional:   groups[5].Captures[0].String(),
		Type:         0,
		OriginalType: "os",
	}
	return cv, nil
}
