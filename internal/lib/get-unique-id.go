package lib

import (
	"context"
	"fmt"

	generator_storage "id-generator/internal/generator-storage"
)

func GetUniqueIdWithType(ctx context.Context, sysType string) (newId string, err error) {
	sysTypeId, err := GetSysTypeValue(sysType)
	if err != nil {
		return "", err
	}

	rawId := generator_storage.Storage.GetRawId()

	return fmt.Sprintf("%010d%01d%07d", rawId.Timestamp, sysTypeId, rawId.Tail), nil
}
