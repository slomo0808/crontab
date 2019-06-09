package common

import "github.com/kataras/iris/core/errors"

var (
	ERR_LOCK_ALREADY_REQUIRED error = errors.New("锁已经被占用")
	ERR_NO_LOCAL_IP_FOUND     error = errors.New("没有找到网卡ip")
)
