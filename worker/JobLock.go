package worker

import "go.etcd.io/etcd/clientv3"

//分布式锁(TXN事务)
type JobLock struct {
	//etcd 客户端
	kv    clientv3.KV
	lease clientv3.Lease

	jobName string //任务名
}

//初始化一把锁
func InitJobLock(jobName string, kv clientv3.KV, lease clientv3.Lease) (jobLock *JobLock) {
	jobLock = &JobLock{
		kv:      kv,
		lease:   lease,
		jobName: jobName,
	}
	return
}
