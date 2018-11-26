package common

import (
	"encoding/json"
	"github.com/gorhill/cronexpr"
	"strings"
	"time"
)

/*
[postman_collections]https://documenter.getpostman.com/view/2684216/RzfZQtBn
*/
//定时任务
type Job struct {
	Name     string `json:"name"`     //任务名
	Command  string `json:"command"`  //shel 命令
	CronExpr string `json:"cornExpr"` //cron 表达式
}

//任务调度计划
type JobSchedulePlan struct {
	Job      *Job                 //要调度的任务信息
	Expr     *cronexpr.Expression //解析好的cronexpr
	NextTime time.Time            //下次调度时间
}

//HTTP接口应答
type Response struct {
	Error int         `json:"error"`
	Msg   string      `json:"msg"`
	Data  interface{} `json:"data"`
}

//变化事件
type JobEvent struct {
	EventType int // save、delete
	Job       *Job
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

//反序列化Job
func UnpackJob(value []byte) (ret *Job, err error) {
	var (
		job *Job
	)
	job = &Job{}
	if err = json.Unmarshal(value, job); err != nil {
		return
	}
	ret = job
	return
}

//从etcd的key中提取任务名
func ExtractJobName(jobkey string) string {
	return strings.TrimPrefix(jobkey, JOB_SAVA_DIR)
}

//任务变化事件有两种: 1.更新任务 2.删除任务
func BuildJobEvent(eventType int, job *Job) (jobEvent *JobEvent) {
	return &JobEvent{
		EventType: eventType,
		Job:       job,
	}
}

//构造任务执行计划
func BuildJobSchedulePlan(job *Job) (jobSchedulePlan *JobSchedulePlan, err error) {
	var (
		expr *cronexpr.Expression
	)
	//解析Job的cron 表达式
	if expr, err = cronexpr.Parse(job.CronExpr); err != nil {
		return
	}

	//生成任务调度计划对象
	jobSchedulePlan = &JobSchedulePlan{
		Job:      job,
		Expr:     expr,
		NextTime: expr.Next(time.Now()),
	}
	return
}
