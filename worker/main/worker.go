package main

import (
	"flag"
	"fmt"
	"github.com/BensonMax/crontab/worker"
	"runtime"
	"time"
)

//初始化线程数
func initEnv() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}

var (
	confFile string //配置文件路径
)

//解析命令行参数
func initArgs() {
	//worker -config ./worker.jon
	//master -h
	flag.StringVar(&confFile, "config", "./worker.json", "worker.json")
	flag.Parse()
}

func main() {
	var (
		err error
	)
	//初始化命令行参数
	initArgs()

	//初始化线程
	initEnv()

	//加载配置
	if err = worker.InitConfig(confFile); err != nil {
		goto ERR
	}

	//初始化任务管理器
	if err = worker.InitJobMgr(); err != nil {
		goto ERR
	}

	//正常退出
	for {
		time.Sleep(1 * time.Second)
	}
	return

ERR:
	fmt.Println(err)

}
