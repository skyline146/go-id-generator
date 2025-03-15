package generator_storage

import (
	"context"
	"fmt"

	"id-generator/internal/lib"
)

func GetUniqueIdWithType(ctx context.Context, sysType string) (newId string, err error) {
	sysTypeId, err := lib.GetSysTypeValue(sysType)
	if err != nil {
		return "", err
	}

	rawId := Storage.GetRawId()

	return fmt.Sprintf("%010d%01d%07d", rawId.Timestamp, sysTypeId, rawId.Tail), nil
}
