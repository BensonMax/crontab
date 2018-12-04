package worker

import (
	"crontab/common"
	"fmt"
	"time"
)

//任务调度
type Scheduler struct {
	jobEventChan      chan *common.JobEvent              //etcd任务事件队列
	jobPlanTable      map[string]*common.JobSchedulePlan //任务调度计划表
	jobExecutingTable map[string]*common.JobExecuteInfo  //任务执行表
	jobResultChan     chan *common.JobExecteResult       //	任务结果队列

}

var (
	G_scheduler *Scheduler
)

//处理任务事件
func (scheduler *Scheduler) handleJobEvent(jobEvent *common.JobEvent) {
	var (
		jobSchedulePlan *common.JobSchedulePlan
		jobExecuteInfo  *common.JobExecuteInfo
		jobExecuting    bool
		err             error
		jobExisted      bool
	)
	switch jobEvent.EventType {
	case common.JOB_EVENT_SAVE: //保存事件
		if jobSchedulePlan, err = common.BuildJobSchedulePlan(jobEvent.Job); err != nil {
			return
		}
		scheduler.jobPlanTable[jobEvent.Job.Name] = jobSchedulePlan
	case common.JOB_EVENT_DELETE: //删除事件
		if jobSchedulePlan, jobExisted = scheduler.jobPlanTable[jobEvent.Job.Name]; jobExisted {
			delete(scheduler.jobPlanTable, jobEvent.Job.Name)
		}
	case common.JOB_EVENT_KILL: //强杀任务
		//取消Command执行，判断任务是否在执行中
		if jobExecuteInfo, jobExecuting = scheduler.jobExecutingTable[jobEvent.Job.Name]; jobExecuting {
			jobExecuteInfo.CancelFunc() //触发command杀死shell 子进程,任务得到退出
		}
	}
}

// 尝试执行任务
func (scheduler *Scheduler) TryStartJob(jobPlan *common.JobSchedulePlan) {
	// 调度和执行是两件事情
	var (
		jobExecuteInfo *common.JobExecuteInfo
		jobExecuting   bool
	)
	//执行的任务可能运行很久，一分钟会调度60次

	//如果任务正在执行，跳过本次任务
	if jobExecuteInfo, jobExecuting = scheduler.jobExecutingTable[jobPlan.Job.Name]; jobExecuting {
		//fmt.Println("尚未执行,跳出执行:", jobPlan.Job.Name)
		return
	}
	//构建执行状态信息
	jobExecuteInfo = common.BuildJobExecuteInfo(jobPlan)

	//保存执行状态
	scheduler.jobExecutingTable[jobPlan.Job.Name] = jobExecuteInfo

	//执行任务 TODO：
	G_executor.ExecuteJob(jobExecuteInfo)
	fmt.Println("执行任务:", jobExecuteInfo.Job.Name, jobExecuteInfo.PlanTime, jobExecuteInfo.RealTime)
}

//重新计算任务调度状态
func (scheduler *Scheduler) TrySchedule() (scheduleAfter time.Duration) {
	var (
		jobPlan  *common.JobSchedulePlan
		now      time.Time
		nearTime *time.Time
	)

	//如果任务表为空，随便随眠多久
	if len(scheduler.jobPlanTable) == 0 {
		scheduleAfter = 1 * time.Second
		return
	}

	//当前时间
	now = time.Now()

	// 1.遍历所有任务
	for _, jobPlan = range scheduler.jobPlanTable {
		if jobPlan.NextTime.Before(now) || jobPlan.NextTime.Equal(now) {
			scheduler.TryStartJob(jobPlan)
			jobPlan.NextTime = jobPlan.Expr.Next(now) //更新下次执行时间
		}

		//统计最近一个要过期的任务时间
		if nearTime == nil || jobPlan.NextTime.Before(*nearTime) {
			nearTime = &jobPlan.NextTime
		}
	}

	// 下次调度间隔(最近要执行的任务调度时间 - 当前时间)
	scheduleAfter = (*nearTime).Sub(now)
	return
}

//处理任务结果
func (scheduler *Scheduler) handleJobResult(result *common.JobExecteResult) {
	var (
		jobLog *common.JobLog
	)
	//删除执行状态
	delete(scheduler.jobExecutingTable, result.ExecuteInfo.Job.Name)

	//生成执行日志
	if result.Err != common.ERR_LOCK_ALREADY_REQUIRED {
		jobLog = &common.JobLog{
			JobName:      result.ExecuteInfo.Job.Name,
			Command:      result.ExecuteInfo.Job.Command,
			Output:       string(result.Output),
			PlanTime:     result.ExecuteInfo.PlanTime.UnixNano() / 1000 / 1000,
			ScheduleTime: result.ExecuteInfo.RealTime.UnixNano() / 1000 / 1000,
			StartTime:    result.StartTime.UnixNano() / 1000 / 1000,
			EndTime:      result.EndTime.UnixNano() / 1000 / 1000,
		}
		if result.Err != nil {
			jobLog.Err = result.Err.Error()
		} else {
			jobLog.Err = ""
		}
		//TODO:存到MongoDB
	}

	fmt.Println("任务执行完成:", result.ExecuteInfo.Job.Name, string(result.Output), result.Err)
}

//调度协程
func (scheduler *Scheduler) sheduleLoop() {
	var (
		jobEvent      *common.JobEvent
		scheduleAfter time.Duration
		scheduleTimer *time.Timer
		jobResult     *common.JobExecteResult
	)

	//初始化一次(1秒)
	scheduleAfter = scheduler.TrySchedule()

	//调度的延迟定时器
	scheduleTimer = time.NewTimer(scheduleAfter)

	//定时任务
	for {
		select {
		case jobEvent = <-scheduler.jobEventChan: //监听任务变化事件
			//对内存中维护的任务列表做增删改查
			scheduler.handleJobEvent(jobEvent)
		case <-scheduleTimer.C: //最近的任务到期了
		case jobResult = <-scheduler.jobResultChan: //监听任务执行结果
			scheduler.handleJobResult(jobResult)
		}
		//调度一次任务
		scheduleAfter = scheduler.TrySchedule()
		//重置调度时间间隔
		scheduleTimer.Reset(scheduleAfter)
	}
}

//推送任务变化事件
func (scheduler *Scheduler) PushJobEvent(jobEvent *common.JobEvent) {
	scheduler.jobEventChan <- jobEvent
}

//初始化调度器
func InitScheduler() (err error) {
	G_scheduler = &Scheduler{
		jobEventChan:      make(chan *common.JobEvent, 1000),
		jobPlanTable:      make(map[string]*common.JobSchedulePlan),
		jobExecutingTable: make(map[string]*common.JobExecuteInfo),
		jobResultChan:     make(chan *common.JobExecteResult, 1000),
	}
	//启动调度协程
	go G_scheduler.sheduleLoop()
	return
}

//回传任务执行结果
func (scheduler *Scheduler) PushJobResult(jobResult *common.JobExecteResult) {
	scheduler.jobResultChan <- jobResult
}
