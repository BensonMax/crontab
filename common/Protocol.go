package common

import "encoding/json"

//定时任务
type Job struct {
	Name     string `json:"name"`     //任务名
	Command  string `json:"command"`  //shel 命令
	CronExpr string `json:"cornExpr"` //cron 表达式
}

//HTTP接口应答
type Response struct {
	Error int         `json:"error"`
	Msg   string      `json:"msg"`
	Data  interface{} `json:"data"`
}

//应答方法
func BuildResponse(errno int, msg string, data interface{}) (resp []byte, err error) {
	//定义一个response
	var (
		response Response
	)

	response.Error = errno
	response.Msg = msg
	response.Data = data

	//2.序列化
	resp, err = json.Marshal(response)
	return
}
