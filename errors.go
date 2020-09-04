package oss

import (
	"errors"

	error_utils "github.com/unapu-go/error-utils"
)

var ErrAssetFsUnavailable = errors.New("asset fs unavailable")

func IsErrAssetFsUnavailable(err error) bool {
	return error_utils.IsError(ErrAssetFsUnavailable, err)
}
