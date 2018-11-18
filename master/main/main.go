package main

import (
	"fmt"
	"github.com/BensonMax/crontab/master"
	"runtime"
)

func initEnv() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}

func main() {
	var (
		err error
	)

	//初始化线程
	initEnv()

	//启动服务
	if err = master.InitApiServer(); err != nil {
		goto ERR

	}

ERR:
	fmt.Println(err)

}
