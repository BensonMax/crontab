package common

//定时任务
type Job struct {
	Name     string `json:"name"`     //任务名
	Command  string `json:"command"`  //shel 命令
	CronExpr string `json:"cornExpr"` //cron 表达式
}
