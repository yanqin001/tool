package version

import (
	"errors"
	"fmt"
	"github.com/dlclark/regexp2"
	"strings"
)

// 转换(,9.4.51],[10.0.0-alpha0,10.0.15],[11.0.0-alpha0,11.0.15],[12.0.0.alpha0,12.0.0.beta4] 为(,9.4.51]||[10.0.0-alpha0,10.0.15]||(11.0.0-alpha0,11.0.15)||[12.0.0.alpha0,12.0.0.beta4]
func (svc *versionSvc) GetFormatExp(exp string) string {
	exp = strings.ReplaceAll(exp, " ", "")
	exp = strings.ReplaceAll(exp, "),(", ")||(")
	exp = strings.ReplaceAll(exp, "),[", ")||[")
	exp = strings.ReplaceAll(exp, "],(", "]||(")
	exp = strings.ReplaceAll(exp, "],[", "]||[")
	exp = strings.ReplaceAll(exp, "==", "=")
	exp = strings.ReplaceAll(exp, ")(", ")||(")
	exp = strings.ReplaceAll(exp, ")[", ")||[")
	exp = strings.ReplaceAll(exp, "](", "]||(")
	exp = strings.ReplaceAll(exp, "][", "]||[")
	return exp
}

// 将>=等类型的版本表达式转换为[类型(单个表达式)
func (svc *versionSvc) FormatException(s string) (string, error) {
	s = strings.TrimSpace(s)
	if !strings.ContainsAny(s, ">=<") {
		return s, nil
	}

	parts := strings.Split(s, ",")
	length := len(parts)
	if length > 2 {
		return "", errors.New("wrong length")
	}

	if length == 1 && strings.Contains(s, ">") && strings.Contains(s, "<") {
		// 处理">=4.0 <4.4.1"这种类型
		if strings.HasPrefix(s, ">") {
			parts = strings.Split(s, "<")
			length = len(parts)
			if length > 2 {
				return "", errors.New("wrong length")
			}
			parts[1] = "<" + parts[1]
		} else if strings.HasPrefix(s, "<") { // 处理<2.1.5-M1 >=2.1.4-M1
			parts = strings.Split(s, ">")
			length = len(parts)
			if length > 2 {
				return "", errors.New("wrong length")
			}
			_part0 := ">" + parts[1]
			_part1 := parts[0]
			parts[0] = _part0
			parts[1] = _part1
		}

	}

	exp, version, err := svc.GetExpVersion(parts[0])
	if err != nil {
		return "", err
	}

	var result string
	if length == 1 {
		switch exp {
		case "<":
			result = fmt.Sprintf("(,%s)", version)
		case "<=":
			result = fmt.Sprintf("(,%s]", version)
		case "=":
			result = fmt.Sprintf("[%s,%s]", version, version)
		case ">":
			result = fmt.Sprintf("(%s,)", version)
		case ">=":
			result = fmt.Sprintf("[%s,)", version)
		}
		return result, nil
	}

	switch exp {
	case ">":
		result = "(" + version
	case ">=":
		result = "[" + version
	default:
		return "", errors.New("error exp prefix")
	}

	exp2, version2, err := svc.GetExpVersion(parts[1])
	if err != nil {
		return "", err
	}
	switch exp2 {
	case "<":
		result = result + "," + version2 + ")"
	case "<=":
		result = result + "," + version2 + "]"
	default:
		return "", errors.New("error exp sufix")
	}
	return result, nil
}

// 解析>=等类型的版本表达式
func (svc *versionSvc) GetExpVersion(s string) (string, string, error) {
	regexpString := `([<=>]+)(.*)`
	reg := regexp2.MustCompile(regexpString, 0)
	match, err := reg.FindStringMatch(s)
	if err != nil {
		return "", "", err
	}

	if match == nil {
		return "", "", errors.New("No match")
	}
	groups := match.Groups()
	if len(groups) != 3 {
		return "", "", errors.New("wrong groups")
	}
	exp := groups[1].Captures[0].String()
	version := groups[2].Captures[0].String()
	return strings.TrimSpace(exp), strings.TrimSpace(version), nil
}
