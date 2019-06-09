package master

import (
	"context"
	"go.etcd.io/etcd/clientv3"
	"go.etcd.io/etcd/mvcc/mvccpb"
	"imooc.com/go-crontab/crontab/common"
	"time"
)

type WorkerMgr struct {
	client *clientv3.Client
	kv     clientv3.KV
}

var (
	G_WorkerMgr *WorkerMgr
)

func InitWorkerMgr() (err error) {
	var (
		config clientv3.Config
		client *clientv3.Client
		kv     clientv3.KV
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

	// 得到kv的api子集
	kv = clientv3.NewKV(client)

	G_WorkerMgr = &WorkerMgr{
		client: client,
		kv:     kv,
	}

	return
}

func (wm *WorkerMgr) ListWorkers() (ips []string, err error) {
	var (
		getResp *clientv3.GetResponse
		kvPair  *mvccpb.KeyValue
		ip      string
	)
	ips = make([]string, 0)
	if getResp, err = wm.kv.Get(context.TODO(), common.JOB_WORKER_DIR, clientv3.WithPrefix()); err != nil {
		return
	}

	if len(getResp.Kvs) != 0 {
		for _, kvPair = range getResp.Kvs {
			ip = common.ExtractIp(string(kvPair.Key))
			ips = append(ips, ip)
		}
	}
	return
}
