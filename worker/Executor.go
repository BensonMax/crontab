package worker

import (
	"crontab/common"
	"math/rand"
	"os/exec"
	"time"
)

type Executor struct {
}

var (
	G_executor *Executor
)

//执行一个任务
func (executor *Executor) ExecuteJob(info *common.JobExecuteInfo) {
	go func() {
		var (
			cmd     *exec.Cmd
			err     error
			output  []byte
			result  *common.JobExecteResult
			jobLock *JobLock
		)
		//任务结果
		result = &common.JobExecteResult{
			ExecuteInfo: info,
			Output:      make([]byte, 0),
		}
		//初始化锁
		jobLock = G_JobMgr.CreateJobLock(info.Job.Name)

		//记录任务开始时间
		result.StartTime = time.Now()

		//上锁
		//随机睡眠（0~1s）
		time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)
		err = jobLock.TryLock()
		//释放锁
		defer jobLock.UnLock()

		if err != nil { //上锁失败
			result.Err = err
			result.EndTime = time.Now()
		} else {
			//重置任务开始时间
			result.StartTime = time.Now()
			//执行shell命令
			cmd = exec.CommandContext(info.CancelCtx, "C:\\cygwin64\\bin\\bash.exe", "-c", info.Job.Command)
			//执行并捕获输出
			output, err = cmd.CombinedOutput()
			//记录任务结束时间
			result.EndTime = time.Now()
			result.Output = output
			result.Err = err
		}
		//任务执行完成后，把执行的结果返回给Scheduler，Scheduler会从executingTable中删除掉执行记录
		G_scheduler.PushJobResult(result)
	}()
}

//初始化执行器
func InitExecutor() (err error) {
	G_executor = &Executor{}
	return
}
