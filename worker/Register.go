package worker

import (
	"context"
	"fmt"
	"go.etcd.io/etcd/clientv3"
	"imooc.com/go-crontab/crontab/common"
	"net"
	"time"
)

// 注册节点到 /cron/workers/ip
type Register struct {
	client *clientv3.Client
	kv     clientv3.KV
	lease  clientv3.Lease

	localIP string
}

var (
	G_Register *Register
)

// 获取本机网卡ip
func getLocalIp() (ipv4 string, err error) {
	var (
		addrs   []net.Addr
		addr    net.Addr
		ipNet   *net.IPNet
		isIpNet bool
	)
	if addrs, err = net.InterfaceAddrs(); err != nil {
		return
	}
	// 取第一个非lo的网卡IP
	for _, addr = range addrs {
		// 这个addr是ip地址
		// 是ipv4或者ipv6
		// 如果它是ip地址并且不是环回地址, 进一步处理
		if ipNet, isIpNet = addr.(*net.IPNet); isIpNet && !ipNet.IP.IsLoopback() {
			// 跳过ipv6
			if ipNet.IP.To4() != nil {
				ipv4 = ipNet.IP.String()
				return
			}
		}
	}
	err = common.ERR_NO_LOCAL_IP_FOUND
	return
}

// 注册到/cron/workers/IP, 并且自动续租
func (r *Register) keepOnline() {
	var (
		regKey            string
		leaseGrantResp    *clientv3.LeaseGrantResponse
		err               error
		leaseId           clientv3.LeaseID
		keepAliveRespChan <-chan *clientv3.LeaseKeepAliveResponse
		keepAliveResp     *clientv3.LeaseKeepAliveResponse
		cancelCtx         context.Context
		cancelFunc        context.CancelFunc
	)
	regKey = common.JOB_WORKER_DIR + r.localIP
	for {
		// 创建租约
		if leaseGrantResp, err = r.lease.Grant(context.TODO(), 10); err != nil {
			fmt.Println("grant err:", err)
			goto RETRY
		}
		leaseId = leaseGrantResp.ID

		cancelCtx, cancelFunc = context.WithCancel(context.TODO())
		// 自动续租
		if keepAliveRespChan, err = r.lease.KeepAlive(cancelCtx, leaseId); err != nil {
			fmt.Println("keepAlive err:", err)
			goto RETRY
		}

		// 注册到etcd
		if _, err = r.kv.Put(context.TODO(), regKey, "", clientv3.WithLease(leaseId)); err != nil {
			fmt.Println("put err:", err)
			goto RETRY
		}

		// 处理续租应答
		for {
			select {
			case keepAliveResp = <-keepAliveRespChan:
				if keepAliveResp == nil { // 续租失败
					goto RETRY
				}
			}
		}

	RETRY:
		time.Sleep(1 * time.Second)
		if cancelFunc != nil {
			cancelFunc()
		}
	}

}

func InitRegister() (err error) {
	var (
		config  clientv3.Config
		client  *clientv3.Client
		lease   clientv3.Lease
		kv      clientv3.KV
		localIp string
	)
	if localIp, err = getLocalIp(); err != nil {
		return
	}
	config = clientv3.Config{
		Endpoints:   G_config.EtcdEndPoints,
		DialTimeout: time.Duration(G_config.EtcdDialTimeout) * time.Millisecond,
	}
	if client, err = clientv3.New(config); err != nil {
		return
	}

	kv = clientv3.NewKV(client)
	lease = clientv3.NewLease(client)
	G_Register = &Register{
		client:  client,
		kv:      kv,
		lease:   lease,
		localIP: localIp,
	}

	go G_Register.keepOnline()

	return
}
