package master

import (
	"encoding/json"
	"imooc.com/go-corntab/crontab/common"
	"net"
	"net/http"
	"strconv"
	"time"
)

// 任务的http借口
type ApiService struct {
	httpServer *http.Server
}

var (
	// 单例对象
	G_apiServer *ApiService
)

// 保存任务接口
// POST job={"name":"job1","command":"echo hello","cronExpr":"* * * * * "}
func handleJobSave(w http.ResponseWriter, r *http.Request) {
	var (
		err     error
		postJob string
		job     common.Job
		oldJob  *common.Job
		bytes   []byte
	)
	// 1.解析POST表单
	if err = r.ParseForm(); err != nil {
		goto ERR
	}
	// 2. 取表单中的job字段
	postJob = r.PostForm.Get("job")
	// 3. 反序列化job
	if err = json.Unmarshal([]byte(postJob), &job); err != nil {
		goto ERR
	}

	// 保存到etcd
	if oldJob, err = G_JobMgr.SaveJob(&job); err != nil {
		goto ERR
	}
	// 返回正常应答
	if bytes, err = common.BuildResponse(0, "succeed", oldJob); err == nil {
		w.Write(bytes)
	}
	return
ERR:
	// 返回异常应答
	if bytes, err = common.BuildResponse(-1, err.Error(), oldJob); err == nil {
		w.Write(bytes)
	}
}

// 删除任务接口
// POST /job/delete name=job1
func handleJobDelete(w http.ResponseWriter, r *http.Request) {
	var (
		err     error
		jobName string
		oldJob  *common.Job
		bytes   []byte
	)
	if err = r.ParseForm(); err != nil {
		goto ERR
	}

	// 删除的任务名
	jobName = r.PostForm.Get("name")

	// 删除任务
	if oldJob, err = G_JobMgr.DeleteJob(jobName); err != nil {
		goto ERR
	}

	// 正常应答
	if bytes, err = common.BuildResponse(0, "succeed", oldJob); err == nil {
		w.Write(bytes)
	}
	return
ERR:
	// 返回异常应答
	if bytes, err = common.BuildResponse(-1, err.Error(), oldJob); err == nil {
		w.Write(bytes)
	}
}

// 列举所有crontab任务接口
// GET /job/list
func handleJobList(w http.ResponseWriter, r *http.Request) {
	var (
		jobList []*common.Job
		err     error
		bytes   []byte
	)
	if jobList, err = G_JobMgr.ListJob(); err != nil {
		goto ERR
	}
	// 返回正常应答
	if bytes, err = common.BuildResponse(0, "succeed", jobList); err == nil {
		w.Write(bytes)
	}
	return
ERR:
	// 返回异常应答
	if bytes, err = common.BuildResponse(-1, err.Error(), jobList); err == nil {
		w.Write(bytes)
	}

}

// 强制杀死任务
func handleJobKill(w http.ResponseWriter, r *http.Request) {
	var (
		err     error
		jobName string
		bytes   []byte
	)
	if err = r.ParseForm(); err != nil {
		goto ERR
	}
	jobName = r.PostForm.Get("name")

	if err = G_JobMgr.KillJob(jobName); err != nil {
		goto ERR
	}

	// 返回正常应答
	if bytes, err = common.BuildResponse(0, "succeed", nil); err == nil {
		w.Write(bytes)
	}
	return
ERR:
	// 返回异常应答
	if bytes, err = common.BuildResponse(-1, err.Error(), nil); err == nil {
		w.Write(bytes)
	}
}

// 初始化服务
func InitApiServer() (err error) {
	var (
		mux           *http.ServeMux
		listener      net.Listener
		httpServer    *http.Server
		staticDir     http.Dir
		staticHandler http.Handler
	)

	// 配置路由
	mux = http.NewServeMux()
	mux.HandleFunc("/job/save", handleJobSave)
	mux.HandleFunc("/job/delete", handleJobDelete)
	mux.HandleFunc("/job/list", handleJobList)
	mux.HandleFunc("/job/kill", handleJobKill)

	// 静态文件目录
	staticDir = http.Dir(G_config.WebRoot)
	staticHandler = http.FileServer(staticDir)
	mux.Handle("/", http.StripPrefix("/", staticHandler))

	// 启动监听
	if listener, err = net.Listen("tcp", ":"+strconv.Itoa(G_config.ApiPort)); err != nil {
		return
	}

	// 创建一个http服务
	httpServer = &http.Server{
		ReadTimeout:  time.Duration(G_config.ApiReadTimeout) * time.Millisecond,
		WriteTimeout: time.Duration(G_config.ApiWriteTimeout) * time.Millisecond,
		Handler:      mux,
	}

	// 赋值单例
	G_apiServer = &ApiService{
		httpServer: httpServer,
	}

	// 启动了服务端
	go httpServer.Serve(listener)
	return
}
