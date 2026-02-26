package version

import (
	"fmt"
	"testing"
)

func Test_versionSvc_Sort(t *testing.T) {
	versions := []string{
		"1.0.6",
		"1.0.1",
		"1.0.2",
		"1.0.3",
		"1.0.4",
		"1.0.5",
		"1.0.7",
		"1.0.8",
		"1.0.9",
		"1.0.10",
		"1.0.11",
		"1.3",
		"1.0.0"}
	svc := NewVersionSvc()
	svc.Sort(versions)
	fmt.Println(versions)
}
