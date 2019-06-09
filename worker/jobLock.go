package worker

import (
	"context"
	"go.etcd.io/etcd/clientv3"
	"imooc.com/go-crontab/crontab/common"
)

// 分布式锁(TXN事物)
type JobLock struct {
	// etcd 客户端
	kv    clientv3.KV
	lease clientv3.Lease

	jobName    string             // 任务名
	cancelFUnc context.CancelFunc // 用于取消自动续租
	leaseId    clientv3.LeaseID   // 用于Revoke
	isLocked   bool               // 是否上锁成功
}

func InitJobLock(jobName string, kv clientv3.KV, lease clientv3.Lease) (jobLock *JobLock) {
	jobLock = &JobLock{
		kv:      kv,
		lease:   lease,
		jobName: jobName,
	}
	return
}

// 尝试上锁
func (jl *JobLock) TryLock() (err error) {
	var (
		leaseGrantResp *clientv3.LeaseGrantResponse
		ctx            context.Context
		cancelFunc     context.CancelFunc
		leaseId        clientv3.LeaseID
		keepAliveResp  <-chan *clientv3.LeaseKeepAliveResponse
		keepResp       *clientv3.LeaseKeepAliveResponse
		txn            clientv3.Txn
		lockKey        string
		txnResp        *clientv3.TxnResponse
	)
	// 创建一个租约, 5秒
	if leaseGrantResp, err = jl.lease.Grant(context.TODO(), 5); err != nil {
		return
	}
	leaseId = leaseGrantResp.ID

	// context用于取消自动续租
	ctx, cancelFunc = context.WithCancel(context.TODO())

	// 自动续约
	if keepAliveResp, err = jl.lease.KeepAlive(ctx, leaseId); err != nil {
		goto FAIL
	}

	// 处理续租应答的协程
	go func() {
		for {
			select {
			case keepResp = <-keepAliveResp:
				if keepResp == nil {
					goto END
				}
			}
		}
	END:
	}()
	// 创建事物txn
	txn = jl.kv.Txn(context.TODO())

	// 锁路径
	lockKey = common.JOB_LOCK_DIR + jl.jobName

	// 事物抢锁
	txn.If(clientv3.Compare(clientv3.CreateRevision(lockKey), "=", 0)).
		Then(clientv3.OpPut(lockKey, "", clientv3.WithLease(leaseId))).
		Else(clientv3.OpGet(lockKey))

	// 提交事务
	if txnResp, err = txn.Commit(); err != nil {
		goto FAIL
	}
	// 上锁失败释放租约
	if !txnResp.Succeeded { // not succeeded 锁被占用
		err = common.ERR_LOCK_ALREADY_REQUIRED
		goto FAIL
	}

	// 上锁成功
	jl.leaseId = leaseId
	jl.cancelFUnc = cancelFunc
	jl.isLocked = true
	return
FAIL:
	cancelFunc()                             // 取消自动续租
	jl.lease.Revoke(context.TODO(), leaseId) // 施放租约
	return
}

// 施放锁
func (jl *JobLock) UnLock() {
	if jl.isLocked {
		jl.cancelFUnc() // 取消自动租约协程
		jl.lease.Revoke(context.TODO(), jl.leaseId)
	}
}
