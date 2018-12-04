package worker

import (
	"context"
	"crontab/common"
	"go.etcd.io/etcd/clientv3"
	"go.etcd.io/etcd/mvcc/mvccpb"
	"time"
)

type JobMgr struct {
	client    *clientv3.Client
	kv        clientv3.KV
	lease     clientv3.Lease
	watcher   clientv3.Watcher
	watchResp clientv3.WatchResponse
}

var (
	//单例
	G_JobMgr *JobMgr
)

//监听任务变化
func (JobMgr *JobMgr) watchJobs() (err error) {
	var (
		getResp            *clientv3.GetResponse
		kvpair             *mvccpb.KeyValue
		job                *common.Job
		watchStartRevision int64
		watchChan          clientv3.WatchChan
		watchResp          clientv3.WatchResponse
		watchEvent         *clientv3.Event
		jobName            string
		jobEvent           *common.JobEvent
	)
	//1.get一下/cron/jobs/目录下的所有任务，并且获知当前集群的revision
	if getResp, err = JobMgr.kv.Get(context.TODO(), common.JOB_SAVA_DIR, clientv3.WithPrefix()); err != nil {
		return
	}
	//1.查找当前有哪些任务
	for _, kvpair = range getResp.Kvs {
		//反序列化
		if job, err = common.UnpackJob(kvpair.Value); err == nil {
			jobEvent = common.BuildJobEvent(common.JOB_EVENT_SAVE, job)
			//把这个job同步给scheduler(调度协程)
			G_scheduler.PushJobEvent(jobEvent)
		}
	}
	//从该revision向后监听变化事件
	go func() { //监听协程
		//从GET时刻的后续版本开始监听版本
		watchStartRevision = getResp.Header.Revision + 1
		watchChan = JobMgr.watcher.Watch(context.TODO(), common.JOB_SAVA_DIR, clientv3.WithRev(watchStartRevision), clientv3.WithPrefix())
		//处理监听事件
		for watchResp = range watchChan {
			for _, watchEvent = range watchResp.Events {
				switch watchEvent.Type {
				case mvccpb.PUT: //任务保存事件
					//反序列化Job
					if job, err = common.UnpackJob(watchEvent.Kv.Value); err != nil {
						//忽略无效json
						continue
					}
					//构建一个更新Event
					jobEvent = common.BuildJobEvent(common.JOB_EVENT_SAVE, job)

				case mvccpb.DELETE: //任务被删除
					// Delete /cron/jobs/job10
					jobName = common.ExtractJobName(string(watchEvent.Kv.Value))

					job = &common.Job{Name: jobName}

					//构建一个删除Event
					jobEvent = common.BuildJobEvent(common.JOB_EVENT_DELETE, job)

				}
				// 推送给Scheduler
				G_scheduler.PushJobEvent(jobEvent)
			}
		}
	}()
	return
}

//监听强啥任务通知
func (JobMgr *JobMgr) watchKiller() {
	var (
		watchChan  clientv3.WatchChan
		watchResp  clientv3.WatchResponse
		watchEvent *clientv3.Event
		jobEvent   *common.JobEvent
		jobName    string
		job        *common.Job
	)
	go func() { //监听协程
		//监听/cron/killer/目录变化

		watchChan = JobMgr.watcher.Watch(context.TODO(), common.JOB_KILLER_DIR, clientv3.WithPrefix())
		//处理监听事件
		for watchResp = range watchChan {
			for _, watchEvent = range watchResp.Events {
				switch watchEvent.Type {
				case mvccpb.PUT: //任务保存事件
					jobName = common.ExtractKillerName(string(watchEvent.Kv.Key))
					job = &common.Job{Name: jobName}
					jobEvent = common.BuildJobEvent(common.JOB_EVENT_KILL, job)
					// 推送给Scheduler
					G_scheduler.PushJobEvent(jobEvent)
				case mvccpb.DELETE: //任务被删除
				}
			}
		}
	}()
	return
}

//初始化管理器
func InitJobMgr() (err error) {
	var (
		config  clientv3.Config
		client  *clientv3.Client
		kv      clientv3.KV
		lease   clientv3.Lease
		watcher clientv3.Watcher
	)

	config = clientv3.Config{
		Endpoints:   G_config.EtcdEndpoints,                                     //集群地址
		DialTimeout: time.Duration(G_config.EtcdDialTimeout) * time.Millisecond, //超时时间
	}

	if client, err = clientv3.New(config); err != nil {
		return
	}

	//得到KV和Lease的API子集
	kv = clientv3.NewKV(client)
	lease = clientv3.NewLease(client)
	watcher = clientv3.NewWatcher(client)
	//赋值单例
	G_JobMgr = &JobMgr{
		client:  client,
		kv:      kv,
		lease:   lease,
		watcher: watcher,
	}

	//启动任务监听
	G_JobMgr.watchJobs()
	//监听任务强杀
	G_JobMgr.watchKiller()

	return
}

//创建任务执行锁
func (jobMgr *JobMgr) CreateJobLock(jobName string) (jobLock *JobLock) {
	jobLock = InitJobLock(jobName, jobMgr.kv, jobMgr.lease)

	return
}
