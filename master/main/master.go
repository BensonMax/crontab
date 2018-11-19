package main

import (
	"crontab/master"
	"flag"
	"fmt"
	"runtime"
)

func initEnv() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}

var (
	confFile string //配置文件路径
)

func initArgs() {
	//master -config ./master.jon
	//master -h
	flag.StringVar(&confFile, "config", "./master.json", "指定master.json")
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

	if err = master.InitConfig(confFile); err != nil {
		goto ERR
	}

	//启动服务
	if err = master.InitApiServer(); err != nil {
		goto ERR

	}

	//正常退出
	return

ERR:
	fmt.Println(err)

}
