package worker

import (
	"imooc.com/go-crontab/crontab/common"
	"os/exec"
	"time"
)

// 任务执行器
type Executor struct {
	Cmd exec.Cmd
}

var (
	G_Executor *Executor
)

// 执行一个任务
func (e *Executor) ExecuteJob(jobInfo *common.JobExecuteInfo) {
	go func() {
		var (
			cmd     *exec.Cmd
			err     error
			output  []byte
			result  *common.JobExcuteResult
			jobLock *JobLock
		)
		// 任务结果
		result = &common.JobExcuteResult{
			ExecuteInfo: jobInfo,
			Output:      make([]byte, 0),
		}

		// 首先获取分布式锁
		jobLock = G_JobMgr.CreateJobLock(jobInfo.Job.Name)
		// 记录任务开始时间
		result.StartTIme = time.Now()

		// 上锁
		err = jobLock.TryLock()
		// 释放锁
		defer jobLock.UnLock()

		if err != nil { // 上锁失败
			result.Err = err
			result.EndTime = time.Now()
		} else {
			// 抢锁成功则重新获取任务开始时间
			result.StartTIme = time.Now()

			// 执行shell命令
			cmd = exec.CommandContext(jobInfo.CancelCtx, "/bin/bash", "-c", jobInfo.Job.Command)

			// 执行并捕获输出
			output, err = cmd.CombinedOutput()

			// 记录任务结束时间
			result.EndTime = time.Now()
			result.Output = output
			result.Err = err
		}
		// 任务执行完成后，把执行的结果返回给Scheduler, Scheduler会从executingTable中删除执行记录
		G_Scheduler.PushJobResult(result)
	}()
}

// 初始化执行器
func InitExcutor() (err error) {
	G_Executor = &Executor{}
	return
}
