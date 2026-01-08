package version

import (
	"errors"
	"strings"
)

func (svc *versionSvc) InExp(ve VersionExp) (bool, error) {
	exp := svc.GetFormatExp(ve.Expression)
	if exp == "*" {
		return true, nil
	}
	if ve.Version == "" || exp == "" {
		return false, nil
	}

	specificVersion, err := svc.GetCompareVersion(ve.Version, ve.Type)
	if err != nil {
		return false, err
	}
	if specificVersion.Main == "" {
		return false, errors.New("nil version")
	}
	for _, _exp := range strings.Split(exp, "||") {
		_exp, err = svc.FormatException(_exp)
		if err != nil {
			return false, err
		}
		result, err := svc.CompareSingleExpression(_exp, specificVersion, ve)
		if err != nil || result {
			return result, err
		}
	}
	return false, nil
}

func (svc *versionSvc) CompareSingleExpression(exp string, specificVersion compareVersion, ve VersionExp) (bool, error) {
	var result bool
	if exp == "*" {
		return true, nil
	}
	exp = svc.formatExp(exp)
	parts := strings.Split(exp, ",")
	if len(parts) != 2 {
		return result, errors.New("wrong expression")
	}

	minEv, err := svc.GetExpressionVersion(parts[0], ve.Type, 0)
	if err != nil {
		return result, err
	}
	//log.Printf("version: %s, minEv: %s %s", specifyVersion.Original, minEv.Express, minEv.Version.Original)

	// 如果漏洞版本表达式不含pre和Additional, 则比较版本的时候不比较pre
	_matchMain := ve.MatchMain
	if !_matchMain && minEv.CompareVersion.Pre == "" && minEv.CompareVersion.Additional == "" {
		_matchMain = true
	}
	compareResult, err := svc.CompareExp(minEv, specificVersion, ve)
	if err != nil {
		return result, err
	}
	if compareResult == false {
		return false, nil
	}

	maxEv, err := svc.GetExpressionVersion(parts[1], ve.Type, 1)
	//log.Printf("version: %s, maxEv: %s %s", specifyVersion.Original, maxEv.Version.Original, maxEv.Express)
	if err != nil {
		return result, err
	}

	_matchMain = ve.MatchMain
	if !_matchMain && maxEv.CompareVersion.Pre == "" && maxEv.CompareVersion.Additional == "" {
		_matchMain = true
	}
	return svc.CompareExp(maxEv, specificVersion, ve)
}

func (svc *versionSvc) formatExp(exp string) string {
	if !strings.Contains(exp, ",") && strings.HasPrefix(exp, "[") && strings.HasSuffix(exp, "]") { //处理[3.1.2]
		_version := strings.TrimPrefix(exp, "[")
		_version = strings.TrimSuffix(_version, "]")
		exp = "[" + _version + "," + _version + "]"
	} else if !strings.ContainsAny(exp, "[()]>=<") { // 处理RASA-2013-0015 0.5.1
		exp = "[" + exp + "," + exp + "]"
	}
	return exp
}

// 版本表达式字符串转ExpressionVersion
func (svc *versionSvc) GetExpressionVersion(expression, tp string, index int) (expressionVersion, error) {
	expression = strings.TrimSpace(expression)
	ev := expressionVersion{}
	if index == 0 {
		if strings.HasPrefix(expression, "(") {
			ev.Express = "("
			expression = strings.TrimPrefix(expression, "(")
		} else if strings.HasPrefix(expression, "[") {
			ev.Express = "["
			expression = strings.TrimPrefix(expression, "[")
		} else {
			return expressionVersion{}, errors.New("wrong expression")
		}
	} else {
		if strings.HasSuffix(expression, ")") {
			ev.Express = ")"
			expression = strings.TrimSuffix(expression, ")")
		} else if strings.HasSuffix(expression, "]") {
			ev.Express = "]"
			expression = strings.TrimSuffix(expression, "]")
		} else {
			return expressionVersion{}, errors.New("wrong expression")
		}
	}
	expression = strings.TrimSpace(expression)
	//// 检查版本是否在fixmap大版本内，是就将版本表达式后一个版本替换为fixmap对应的版本
	//if fixMap != nil && specifyVersion != nil {
	//	for k, v := range fixMap {
	//		if strings.HasPrefix(specifyVersion.Main, k+".") || specifyVersion.Main == k {
	//			e = v
	//			ev.Express = ")"
	//			break
	//		}
	//	}
	//}
	v, err := svc.GetCompareVersion(expression, tp)
	if err != nil {
		return expressionVersion{}, err
	}
	ev.CompareVersion = v
	return ev, nil
}

// expression中的开始或结束版本与指定版本比较
func (svc *versionSvc) CompareExp(ev expressionVersion, sv compareVersion, ve VersionExp) (bool, error) {
	var inExpression bool
	var result int
	var err error
	if (ev.Express == ")" || ev.Express == "]") && len(ve.Fixed) != 0 {
		ev, err = svc.FixedReplaceExpEnd(ev, sv, ve.Fixed)
		if err != nil {
			return inExpression, err
		}
	}
	if ev.CompareVersion.Main == "" {
		return true, nil
	}
	result, err = svc.Compare(sv, ev.CompareVersion, ve.MatchMain)

	if err != nil {
		return inExpression, err
	}
	if result == 5 {
		return false, nil
	}
	switch ev.Express {
	case "(":
		if result == 2 {
			inExpression = true
		}
	case "[":
		if result != 0 {
			inExpression = true
		}
	case ")":
		if result == 0 {
			inExpression = true
		}
	case "]":
		if result != 2 {
			inExpression = true
		}
	}

	// 如果版本范围是1.0 ~ 3.0， update 是 rc1 ，满足版本范围后还必须匹配update
	return inExpression, nil
}

// FixedReplaceExpEnd 用fixed里面的版本替换expression的结束版本
func (svc *versionSvc) FixedReplaceExpEnd(ev expressionVersion, sv compareVersion, fixed []string) (expressionVersion, error) {
	for _, fixVersion := range fixed {
		fixMainVersion := svc.getFixedMainVersion(fixVersion)
		if strings.HasPrefix(sv.Main, fixMainVersion+".") || sv.Main == fixMainVersion {
			version, err := svc.GetCompareVersion(fixVersion, "")
			if err != nil {
				return ev, err
			}
			ev.CompareVersion = version
			ev.Express = ")"
		}
	}
	return ev, nil
}

func (svc *versionSvc) getFixedMainVersion(fixVersion string) string {
	if strings.Contains(fixVersion, ":") {
		return strings.TrimSpace(strings.Split(fixVersion, ":")[0])
	}
	if !strings.Contains(fixVersion, ".") {
		return fixVersion
	}
	parts := strings.Split(fixVersion, ".")
	return strings.Split(strings.Join(parts[:len(parts)-1], "."), "-")[0] //6.12-rc1 -> 6.12
}
