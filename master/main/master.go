package main

import (
	"flag"
	"fmt"
	"imooc.com/go-crontab/crontab/master"
	"runtime"
	"time"
)

var (
	confFile string //配置文件路径
)

// 解析命令行参数
func initArgs() {
	// master -config ./master.json
	// master -h
	flag.StringVar(&confFile,
		"config",
		"./master.json",
		"指定master.json")

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
	if err = master.InitConfig(confFile); err != nil {
		fmt.Println(err)
		goto ERR
	}

	// 加载workerMGr
	if err = master.InitWorkerMgr(); err != nil {
		goto ERR
	}

	// 日志管理器
	if err = master.InitLogMgr(); err != nil {
		goto ERR
	}

	// 任务管理器
	if err = master.InitJobMgr(); err != nil {
		goto ERR
	}

	// 启动API HTTP服务
	if err = master.InitApiServer(); err != nil {
		goto ERR
	}

	for {
		time.Sleep(5 * time.Second)
	}
	// 正常退出

ERR:
	fmt.Println(err)
}
