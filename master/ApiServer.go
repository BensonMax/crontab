package master

import (
	"encoding/json"
	"github.com/BensonMax/crontab/common"
	"net"
	"net/http"
	"strconv"
	"time"
)

//任务的http接口
type ApiServer struct {
	httpServer *http.Server
}

var (
	//单例对象
	G_apiServer *ApiServer
)

//保存服务
func handleJobSave(resp http.ResponseWriter, req *http.Request) {
	//任务保存到etcd中
	//post job ={"name","job1","command":"echo hello","cronExpr","*****"}
	var (
		err     error
		postJob string
		job     common.Job
		oldjob  *common.Job
		bytes   []byte
	)
	//1、解析表单
	if err = req.ParseForm(); err != nil {
		goto ERR
	}
	//2、取表单中的job字段
	postJob = req.PostForm.Get("job")
	//3、反序列化job
	if err = json.Unmarshal([]byte(postJob), &job); err != nil {
		goto ERR
	}
	//4、保存到etcd
	if oldjob, err = G_JobMgr.SaveJob(&job); err != nil {
		goto ERR
	}
	//5、返回正常应答 {"errno":0,"msg"}
	if bytes, err = common.BuildResponse(0, "success", oldjob); err != nil {
		resp.Write(bytes)
	}
	return

ERR:
	//6,返回异常应答
	if bytes, err = common.BuildResponse(-1, err.Error(), nil); err == nil {
		resp.Write(bytes)
	}
}

//初始化服务
func InitApiServer() (err error) {
	var (
		mux        *http.ServeMux
		listener   net.Listener
		httpServer *http.Server
	)

	//配置路由
	mux = http.NewServeMux()
	mux.HandleFunc("/job/save", handleJobSave)

	//启动TCP监听
	if listener, err = net.Listen("tcp", ":"+strconv.Itoa(G_config.ApiPort)); err != nil {
		return
	}

	//创建一个HTTP服务
	httpServer = &http.Server{
		ReadTimeout:  time.Duration(G_config.ApiReadTimeout) * time.Millisecond,
		WriteTimeout: time.Duration(G_config.ApiWriteTimeout) * time.Millisecond,
		Handler:      mux,
	}
	//赋值单例
	G_apiServer = &ApiServer{
		httpServer: httpServer,
	}
	//启动了服务端
	go httpServer.Serve(listener)

	return
}
