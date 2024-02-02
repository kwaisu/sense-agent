package system

import (
	"fmt"
	"testing"
)

func TestPath(t *testing.T) {
	fmt.Println(Path(9708, "net", "dev_snmp6"))
}

func TestGetPids(t *testing.T) {
	pids, err := GetAllPids()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(pids)
}
