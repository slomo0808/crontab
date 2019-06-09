package main

import (
	"flag"
	"fmt"
	"imooc.com/go-crontab/crontab/worker"
	"runtime"
	"time"
)

var (
	confFile string //配置文件路径
)

// 解析命令行参数
func initArgs() {
	// worker -config ./master.json
	// worker -h
	flag.StringVar(&confFile,
		"config",
		"./worker.json",
		"指定worker.json")

	flag.Parse()
}

func initEnv() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}

func main() {
	var (
		err error
	)

	// 初始化命令行参数
	initArgs()

	// 初始化线程
	initEnv()

	// 加载配置
	if err = worker.InitConfig(confFile); err != nil {
		goto ERR
	}

	// 注册服务
	if err = worker.InitRegister(); err != nil {
		goto ERR
	}

	// 启动日志协程
	if err = worker.InitLogSink(); err != nil {
		goto ERR
	}

	// 加载执行器
	if err = worker.InitExcutor(); err != nil {
		goto ERR
	}

	// 启动调度器
	if err = worker.InitScheduler(); err != nil {
		goto ERR
	}

	// 初始化任务管理器
	if err = worker.InitJobMgr(); err != nil {
		goto ERR
	}

	for {
		time.Sleep(5 * time.Second)
	}
	// 正常退出

ERR:
	fmt.Println(err)
}
