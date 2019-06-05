package common

import "encoding/json"

// 定时任务
type Job struct {
	Name     string `json:"name"`     // 任务名
	Command  string `json:"command"`  // Shell命令
	CronExpr string `json:"cronExpr"` // cron表达式
}

// HTTP接口应答
type Response struct {
	Errno int         `json:"errno"`
	Msg   string      `json:"msg"`
	Data  interface{} `json:"data"`
}

// 应答方法
func BuildResponse(errno int, msg string, data interface{}) (resp []byte, err error) {
	// 定义一个response
	var (
		res Response
	)
	res.Errno = errno
	res.Msg = msg
	res.Data = data

	// 序列化json
	resp, err = json.Marshal(res)
	return
}