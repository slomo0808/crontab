package master

import (
	"context"
	"encoding/json"
	"go.etcd.io/etcd/clientv3"
	"go.etcd.io/etcd/mvcc/mvccpb"
	"imooc.com/go-crontab/crontab/common"
	"time"
)

// 任务管理器
type JobMgr struct {
	client *clientv3.Client
	kv     clientv3.KV
	lease  clientv3.Lease
}

// 单例
var (
	G_JobMgr *JobMgr
)

// 初始化管理器
func InitJobMgr() (err error) {
	var (
		config clientv3.Config
		client *clientv3.Client
		kv     clientv3.KV
		lease  clientv3.Lease
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

	G_JobMgr = &JobMgr{
		client: client,
		kv:     kv,
		lease:  lease,
	}

	return
}

func (j *JobMgr) SaveJob(job *common.Job) (oldJob *common.Job, err error) {
	// 把任务保存到/cron/jobs/任务名 -> json
	var (
		jobKey    string
		jobValue  []byte
		putResp   *clientv3.PutResponse
		oldJobObj common.Job
	)
	// etcd保存的key
	jobKey = common.JOB_SAVE_DIR + job.Name
	// 任务信息的json
	if jobValue, err = json.Marshal(*job); err != nil {
		return
	}

	// 保存到etcd
	if putResp, err = j.kv.Put(context.TODO(), jobKey, string(jobValue), clientv3.WithPrevKV()); err != nil {
		return
	}

	// 如果是更新，那么返回旧值
	if putResp.PrevKv != nil {
		if err = json.Unmarshal(putResp.PrevKv.Value, &oldJobObj); err != nil {
			err = nil
			return
		}
		oldJob = &oldJobObj
	}
	return
}

func (j *JobMgr) DeleteJob(name string) (oldJob *common.Job, err error) {
	var (
		jobKey    string
		delResp   *clientv3.DeleteResponse
		oldJobObj common.Job
	)
	jobKey = common.JOB_SAVE_DIR + name

	// 删除任务
	if delResp, err = G_JobMgr.kv.Delete(context.TODO(), jobKey, clientv3.WithPrevKV()); err != nil {
		return
	}

	if delResp.PrevKvs != nil {
		if err = json.Unmarshal(delResp.PrevKvs[0].Value, &oldJobObj); err != nil {
			err = nil
			return
		}
		oldJob = &oldJobObj
	}

	return
}

func (j *JobMgr) ListJob() (jobs []*common.Job, err error) {
	var (
		getResp  *clientv3.GetResponse
		keyValue *mvccpb.KeyValue
		job      *common.Job
	)
	if getResp, err = G_JobMgr.kv.Get(context.TODO(), common.JOB_SAVE_DIR, clientv3.WithPrefix()); err != nil {
		return
	}

	// 初始化数组空间, 这样调用者只用判断返回的数组长度
	jobs = make([]*common.Job, 0)

	if getResp.Kvs != nil {
		for _, keyValue = range getResp.Kvs {
			job = &common.Job{}
			if err = json.Unmarshal(keyValue.Value, job); err != nil {
				err = nil
				continue
			}
			jobs = append(jobs, job)
		}
	}
	return
}

// 杀死任务
func (j *JobMgr) KillJob(jobName string) (err error) {
	// 更新一下key=/cron/killer/任务名
	var (
		killerKey      string
		leaseGrantResp *clientv3.LeaseGrantResponse
		leaseId        clientv3.LeaseID
	)

	killerKey = common.JOB_KILLER_DIR + jobName

	// 让worker监听到一次put操作, 稍后让它自动过期即可
	if leaseGrantResp, err = G_JobMgr.lease.Grant(context.TODO(), common.KillerLeaseTimeout); err != nil {
		return
	}
	// 租约ID
	leaseId = leaseGrantResp.ID

	// 设置killer标记
	if _, err = G_JobMgr.kv.Put(context.TODO(), killerKey, "", clientv3.WithLease(leaseId)); err != nil {
		return
	}

	return
}
