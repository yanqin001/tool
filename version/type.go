package version

type VersionSvc interface {
	InExp(VersionExp) (bool, error)
	GetMaxVersion(versions []string) (string, error)
	GetMinVersion(versions []string) (string, error)
	Compare(v1, v2 string) (int, error)
	//SortVersions(cs []string) ([]string, error)
}

type versionSvc struct{}

func NewVersionSvc() VersionSvc {
	return &versionSvc{}
}

// VersionExp用于判断某个版本是否在Expression里面
// 外部服务调用version服务需传入VersionExp
type VersionExp struct {
	Version    string
	Expression string
	Type       string
	Fixed      []string // 可传入["4.14: 4.14.272","4.19: 4.19.235"] 或者 ["4.14.272","4.19.235"]
	MatchMain  bool     //如果MatchMain为true, 只要main版本符合就行，不比较pre和Additional
}

// 服务内部用于比较的版本
type compareVersion struct {
	Main           string
	MainSupplement string // openssl 1.1.1f f就是MainSupplement
	Pre            string
	Type           int // 0：标准版本，如1.0.1； 1: 纯数字版本, 如 39；2：字符串版本，如xx2x
	Additional     string
	Original       string
	OriginalType   string
}

// Expression转换的版本，Express字段为(,[,),]之一
type expressionVersion struct {
	CompareVersion compareVersion
	Express        string
}
