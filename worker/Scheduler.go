package worker

import (
	"fmt"
	"imooc.com/go-crontab/crontab/common"
	"time"
)

// 调度协程
type Scheduler struct {
	jobEventChan      chan *common.JobEvent
	jobPlanTable      map[string]*common.JobSchedulerPlan
	jobExecutingTable map[string]*common.JobExecuteInfo
	jobResultChan     chan *common.JobExcuteResult
}

var (
	G_Scheduler *Scheduler
)

// 处理任务事件
func (s *Scheduler) handlerJobEvent(jobEvt *common.JobEvent) {
	var (
		jobSchedulerPlan *common.JobSchedulerPlan
		jobExisted       bool
		err              error
		jobExcuteInfo    *common.JobExecuteInfo
		jobExcuting      bool
	)
	switch jobEvt.EventType {
	case common.JOB_EVENT_SAVE: // 保存任务事件
		if jobSchedulerPlan, err = common.BuildJobSchedulerPlan(jobEvt.Job); err != nil {
			return
		}
		s.jobPlanTable[jobEvt.Job.Name] = jobSchedulerPlan
	case common.JOB_EVENT_DELETE: // 删除任务事件
		if jobSchedulerPlan, jobExisted = s.jobPlanTable[jobEvt.Job.Name]; jobExisted {
			delete(s.jobPlanTable, jobEvt.Job.Name)
		}
	case common.JOB_EVENT_KILL: // 强杀任务事件
		// 取消任务执行
		// 判断任务是否在执行中
		if jobExcuteInfo, jobExcuting = s.jobExecutingTable[jobEvt.Job.Name]; jobExcuting {
			jobExcuteInfo.CancelFunc() // 出发command杀死shell子进程
		}
	}
}

// 处理任务结果
func (s *Scheduler) handleJobResult(jobResult *common.JobExcuteResult) {
	var (
		jobLog *common.JobLog
	)
	// 删除执行状态
	delete(s.jobExecutingTable, jobResult.ExecuteInfo.Job.Name)

	// 生成执行日志
	if jobResult.Err != common.ERR_LOCK_ALREADY_REQUIRED {
		jobLog = &common.JobLog{
			JobName:      jobResult.ExecuteInfo.Job.Name,
			Command:      jobResult.ExecuteInfo.Job.Command,
			Output:       string(jobResult.Output),
			PlanTime:     jobResult.ExecuteInfo.PlanTime.UnixNano() / 1000 / 1000,
			ScheduleTime: jobResult.ExecuteInfo.RealTime.UnixNano() / 1000 / 1000,
			StartTime:    jobResult.StartTIme.UnixNano() / 1000 / 1000,
			EndTime:      jobResult.EndTime.UnixNano() / 1000 / 1000,
		}
		if jobResult.Err != nil {
			jobLog.Err = jobResult.Err.Error()
		} else {
			jobLog.Err = ""
		}
		G_LogSink.Append(jobLog)
	}
	fmt.Println("任务执行完成:", jobResult.ExecuteInfo.Job.Name,
		", 任务输出:", string(jobResult.Output),
		", 任务错误", jobResult.Err)
}

// 尝试执行任务
func (s *Scheduler) TryStartJob(jobPlan *common.JobSchedulerPlan) {
	// 调度和执行是两件事情
	// 执行的任务可能执行很久
	var (
		jobExecuteInfo *common.JobExecuteInfo
		executing      bool
	)
	if _, executing = s.jobExecutingTable[jobPlan.Job.Name]; executing {
		fmt.Println("尚未退出,跳过执行:", jobPlan.Job.Name)
		return
	}
	// 构建执行状态信息
	jobExecuteInfo = common.BuildJobExecteInfo(jobPlan)

	// 保存执行状态
	s.jobExecutingTable[jobPlan.Job.Name] = jobExecuteInfo

	// 执行任务
	fmt.Println("执行任务:", jobExecuteInfo.Job.Name,
		", 计划时间:", jobExecuteInfo.PlanTime,
		", 实际时间:", jobExecuteInfo.RealTime)
	G_Executor.ExecuteJob(jobExecuteInfo)
}

// 重新计算任务调度状态
func (s *Scheduler) TrySchedule() (next time.Duration) {
	var (
		jobPlan  *common.JobSchedulerPlan
		now      time.Time
		nearTime *time.Time
	)

	// 如果任务表为空, 随便睡眠多久
	if len(s.jobPlanTable) == 0 {
		return 1 * time.Second
	}

	now = time.Now()
	// 1.遍历所有任务
	for _, jobPlan = range s.jobPlanTable {
		if jobPlan.NextTime.Before(now) || jobPlan.NextTime.Equal(now) {
			// 尝试执行任务
			s.TryStartJob(jobPlan)
			// 更新下次调度时间
			jobPlan.NextTime = jobPlan.Expr.Next(now)
		}

		// 统计最近一个要过期的任务事件
		if nearTime == nil || jobPlan.NextTime.Before(*nearTime) {
			nearTime = &jobPlan.NextTime
		}
	}

	// 下次调度间隔(最近要执行的任务调度事件-当前事件)
	next = (*nearTime).Sub(now)
	return
}

// 调度携程
func (s *Scheduler) scheduleLoop() {
	var (
		jobEvt        *common.JobEvent
		scheduleAfter time.Duration
		scheduleTimer *time.Timer
		jobResult     *common.JobExcuteResult
	)

	// 初始化一次(1秒)
	scheduleAfter = s.TrySchedule()

	// 调度的延迟定时器
	scheduleTimer = time.NewTimer(scheduleAfter)

	// 定时任务
	for {
		select {
		case jobEvt = <-s.jobEventChan: // 监听到任务变化事件
			// 对内存中维护的任务列表做增改该查
			s.handlerJobEvent(jobEvt)
		case <-scheduleTimer.C: // 最近的任务到期了
		case jobResult = <-s.jobResultChan:
			s.handleJobResult(jobResult)
		}
		// 调度一次任务
		scheduleAfter = s.TrySchedule()
		// 重置定时器
		scheduleTimer.Reset(scheduleAfter)
	}
}

// 推送任务事件
func (s *Scheduler) PushJobEvent(jobEvt *common.JobEvent) {
	s.jobEventChan <- jobEvt
}

func InitScheduler() (err error) {
	G_Scheduler = &Scheduler{
		jobEventChan:      make(chan *common.JobEvent, 1000),
		jobPlanTable:      make(map[string]*common.JobSchedulerPlan),
		jobExecutingTable: make(map[string]*common.JobExecuteInfo),
		jobResultChan:     make(chan *common.JobExcuteResult, 1000),
	}

	// 启动调度协程
	go G_Scheduler.scheduleLoop()

	return
}

// 回传任务执行结果
func (s *Scheduler) PushJobResult(jobResult *common.JobExcuteResult) {
	s.jobResultChan <- jobResult
}
