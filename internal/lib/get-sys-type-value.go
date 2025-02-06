package lib

import (
	"fmt"
	"math/rand/v2"
)

func GetSysTypeValue(sysType string) (int8, error) {
	switch sysType {
	case "Vendor":
		{
			return 0, nil
		}
	case "Box":
		{
			return int8(rand.Int32N(8) + 1), nil
		}
	case "Clients":
		{
			return 9, nil
		}
	}

	return -1, fmt.Errorf("unknown sys_type: %s", sysType)
}
