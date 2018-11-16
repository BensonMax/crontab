package main

import (
	"net"
	"net/http"
)

//任务的http接口
type ApiServer struct {
	httpServer *http.Server
}

func handleJobSave(w http.ResponseWriter, r *http.Request) {

}

//初始化服务
func lnitApiServer(err error) {
	var (
		mux      *http.ServeMux
		listener net.Listener
	)

	//配置路由
	mux = http.NewServeMux()
	mux.HandleFunc("/job/save", handleJobSave)

	//启动TCP监听
	if listener, err = net.Listen("tcp", "8070"); err != nil {
		return
	}
	listener = listener
}
