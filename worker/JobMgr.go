package worker

import (
	"context"
	"go.etcd.io/etcd/clientv3"
	"go.etcd.io/etcd/mvcc/mvccpb"
	"imooc.com/go-crontab/crontab/common"
	"time"
)

// 任务管理器
type JobMgr struct {
	client  *clientv3.Client
	kv      clientv3.KV
	lease   clientv3.Lease
	watcher clientv3.Watcher
}

// 单例
var (
	G_JobMgr *JobMgr
)

// 初始化管理器
func InitJobMgr() (err error) {
	var (
		config  clientv3.Config
		client  *clientv3.Client
		kv      clientv3.KV
		lease   clientv3.Lease
		watcher clientv3.Watcher
	)
	// 初始化配置
	config = clientv3.Config{
		Endpoints:   G_config.EtcdEndPoints,
		DialTimeout: time.Duration(G_config.EtcdDialTimeout) * time.Millisecond,
	}

	// 建立链接
	if client, err = clientv3.New(config); err != nil {
		return
	}

	// 得到kv和lease的api子集
	kv = clientv3.NewKV(client)
	lease = clientv3.NewLease(client)
	watcher = clientv3.NewWatcher(client)
	G_JobMgr = &JobMgr{
		client:  client,
		kv:      kv,
		lease:   lease,
		watcher: watcher,
	}

	// 启动任务监听
	G_JobMgr.watchJobs()

	// 监听killer
	G_JobMgr.watchKiller()
	return
}

// 监听任务变化
func (j *JobMgr) watchJobs() (err error) {
	var (
		getResp       *clientv3.GetResponse
		kvPair        *mvccpb.KeyValue
		watchChan     clientv3.WatchChan
		job           *common.Job
		watchStartRev int64
		watchResp     clientv3.WatchResponse
		event         *clientv3.Event
		jobName       string
		jobEvent      *common.JobEvent
	)
	// 1，get一下从/cron/jobs/目录下的所有目录，并且获知当前集群的revision
	if getResp, err = j.kv.Get(context.TODO(), common.JOB_SAVE_DIR, clientv3.WithPrefix()); err != nil {
		return
	}

	// 当前有那些任务
	for _, kvPair = range getResp.Kvs {
		// 反序列化得到job
		if job, err = common.UnpackJob(kvPair.Value); err == nil {
			jobEvent = common.BuildJobEvent(common.JOB_EVENT_SAVE, job)
			// 把这个jobEvent同步给scheduler(调度携程)
			G_Scheduler.PushJobEvent(jobEvent)
		}
	}

	// 2.从该revision向后监听变化事件
	go func() { // 监听携程
		// 从GET时刻的后续版本开始监听变化
		watchStartRev = getResp.Header.Revision + 1
		// 启动监听
		watchChan = j.watcher.Watch(context.TODO(), common.JOB_SAVE_DIR, clientv3.WithPrefix(), clientv3.WithRev(watchStartRev))
		// 处理监听事件
		for watchResp = range watchChan {
			for _, event = range watchResp.Events {
				switch event.Type {
				case mvccpb.PUT: // 任务保存事件
					// 反序列化Job
					if job, err = common.UnpackJob(event.Kv.Value); err != nil {
						continue
					}
					// 构建一个更新Event事件
					jobEvent = common.BuildJobEvent(common.JOB_EVENT_SAVE, job)
				case mvccpb.DELETE: // 任务删除
					// Delete /cron/jobs/job10
					jobName = common.ExtractJobName(string(event.Kv.Key))
					job = &common.Job{Name: jobName}
					// 构造一个删除Event
					jobEvent = common.BuildJobEvent(common.JOB_EVENT_DELETE, job)

				}
				// 推事件给scheduler
				G_Scheduler.PushJobEvent(jobEvent)
			}
		}
	}()
	return
}

// 监听killer变化
func (j *JobMgr) watchKiller() (err error) {
	var (
		watchChan clientv3.WatchChan
		watchResp clientv3.WatchResponse
		event     *clientv3.Event
		jobName   string
		jobEvent  *common.JobEvent
		job       *common.Job
	)
	// 监听携程
	go func() {
		// 启动监听
		watchChan = j.watcher.Watch(context.TODO(), common.JOB_KILLER_DIR, clientv3.WithPrefix())
		// 处理监听事件
		for watchResp = range watchChan {
			for _, event = range watchResp.Events {
				switch event.Type {
				case mvccpb.PUT: // 杀死某个任务事件
					jobName = common.ExtractJobNameFromKiller(string(event.Kv.Key))
					job = &common.Job{Name: jobName}
					jobEvent = &common.JobEvent{
						EventType: common.JOB_EVENT_KILL,
						Job:       job,
					}
					// 推事件给scheduler
					G_Scheduler.PushJobEvent(jobEvent)
				}
			}
		}
	}()
	return
}

// 创建任务执行锁
func (j *JobMgr) CreateJobLock(jobName string) (jobLock *JobLock) {
	// 返回一把锁
	return InitJobLock(jobName, j.kv, j.lease)
}
