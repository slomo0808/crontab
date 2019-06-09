package common

import (
	"context"
	"encoding/json"
	"github.com/gorhill/cronexpr"
	"strings"
	"time"
)

// 定时任务
type Job struct {
	Name     string `json:"name"`     // 任务名
	Command  string `json:"command"`  // Shell命令
	CronExpr string `json:"cronExpr"` // cron表达式
}

// 任务调度计划
type JobSchedulerPlan struct {
	Job      *Job                 //要调度的任务信息
	Expr     *cronexpr.Expression // 解析好的cron表达式
	NextTime time.Time            // 下次调度时间
}

// 任务执行状态
type JobExecuteInfo struct {
	Job        *Job               // 任务信息
	PlanTime   time.Time          // 理论上的调度时间
	RealTime   time.Time          // 实际执行时间
	CancelCtx  context.Context    // 任务 command 所用的上下文
	CancelFunc context.CancelFunc // 用于取消command执行的cancel函数
}

// HTTP接口应答
type Response struct {
	Errno int         `json:"errno"`
	Msg   string      `json:"msg"`
	Data  interface{} `json:"data"`
}

// 变化事件
type JobEvent struct {
	EventType int // SAVE DELETE
	Job       *Job
}

// 任务执行结果
type JobExcuteResult struct {
	ExecuteInfo *JobExecuteInfo // 执行状态
	Output      []byte          // 脚本输出
	Err         error           // 脚本错误
	StartTIme   time.Time
	EndTime     time.Time
}

// 任务执行日志
type JobLog struct {
	JobName      string `bson:"jobName" json:"jobName"`
	Command      string `bson:"command" json:"command"`
	Err          string `bson:"err" json:"err"`
	Output       string `bson:"output" json:"output"`
	PlanTime     int64  `bson:"planTime" json:"planTime"`         // 计划开始事件
	ScheduleTime int64  `bson:"scheduleTime" json:"scheduleTime"` // 实际调度时间
	StartTime    int64  `bson:"startTime" json:"startTime"`       // 任务启动事件
	EndTime      int64  `bson:"endTime" json:"endTime"`           // 任务执行结束时间
}

// 日志批次
type LogBatch struct {
	Logs []interface{} // 多条日志
}

// 任务日志查询条件
type JobLogFilter struct {
	JobName string `bson:"jobName"`
}

// 任务日志排序规则
type SortLogByStartTime struct {
	SortOrder int `bson:"startTime"` // 按照 {"startTime":-1}
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

// 反序列话到job
func UnpackJob(value []byte) (job *Job, err error) {
	var (
		res *Job
	)
	res = &Job{}
	if err = json.Unmarshal(value, res); err != nil {
		return
	}
	job = res
	return
}

// 从etcd的key中提取任务名
// 从 /cron/jobs/job10 中抹除 /cron/jobs/ 提取出job10
func ExtractJobName(jobKey string) string {
	return strings.TrimPrefix(jobKey, JOB_SAVE_DIR)
}

// 从etcd的key中提取任务名
// 从 /cron/killer/job10 中抹除 /cron/killer/ 提取出job10
func ExtractJobNameFromKiller(jobKey string) string {
	return strings.TrimPrefix(jobKey, JOB_KILLER_DIR)
}

// 从etcd的key中提取ip
// 从 /cron/workers/ip 中抹除 /cron/workers/ 提取出ip
func ExtractIp(ip string) string {
	return strings.TrimPrefix(ip, JOB_WORKER_DIR)
}

// 创建一个jobEvent
func BuildJobEvent(eventType int, job *Job) (je *JobEvent) {
	return &JobEvent{
		EventType: eventType,
		Job:       job,
	}
}

// 构造执行计划
func BuildJobSchedulerPlan(job *Job) (*JobSchedulerPlan, error) {
	var (
		expr *cronexpr.Expression
		err  error
	)
	if expr, err = cronexpr.Parse(job.CronExpr); err != nil {
		return nil, err
	}

	return &JobSchedulerPlan{
		Job:      job,
		Expr:     expr,
		NextTime: expr.Next(time.Now()),
	}, nil
}

// 构造执行状态信息
func BuildJobExecteInfo(jobPlan *JobSchedulerPlan) (jobExecuteInfo *JobExecuteInfo) {
	jobExecuteInfo = &JobExecuteInfo{
		Job:      jobPlan.Job,
		PlanTime: jobPlan.NextTime,
		RealTime: time.Now(),
	}
	jobExecuteInfo.CancelCtx, jobExecuteInfo.CancelFunc = context.WithCancel(context.TODO())
	return
}
