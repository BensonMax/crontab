package worker

import (
	"context"
	"crontab/common"
	"go.etcd.io/etcd/clientv3"
)

//分布式锁(TXN事务)
type JobLock struct {
	//etcd 客户端
	kv    clientv3.KV
	lease clientv3.Lease

	jobName    string             //任务名
	cancelFunc context.CancelFunc //用于终止自动续租
	leaseId    clientv3.LeaseID   //租约ID
	isLocked   bool               //是否上锁成功
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

//尝试上锁
func (jobLock *JobLock) TryLock() (err error) {
	var (
		leaseGrantResp *clientv3.LeaseGrantResponse
		cancleCtx      context.Context
		cancelFunc     context.CancelFunc
		leaseId        clientv3.LeaseID
		keepRespChan   <-chan *clientv3.LeaseKeepAliveResponse
		txn            clientv3.Txn
		lockkey        string
		txnResp        *clientv3.TxnResponse
	)

	// 1、创建租约(5秒)
	if leaseGrantResp, err = jobLock.lease.Grant(context.TODO(), 5); err != nil {
		return
	}
	//context 用于取消自动续租
	cancleCtx, cancelFunc = context.WithCancel(context.TODO())

	//租约ID
	leaseId = leaseGrantResp.ID

	// 2、自动续租
	if keepRespChan, err = jobLock.lease.KeepAlive(cancleCtx, leaseId); err != nil {
		goto FAIL
	}
	// 3、处理续租应答的协程
	go func() {
		var (
			keepResp *clientv3.LeaseKeepAliveResponse
		)
		for {
			select {
			case keepResp = <-keepRespChan: //自动续租的应答
				if keepResp == nil {
					goto ERR
				}
			}
		}
	ERR:
	}()

	// 3、创建事务txn
	txn = jobLock.kv.Txn(context.TODO())

	//锁路径
	lockkey = common.JOB_LOCK_DIR + jobLock.jobName

	// 4、事务抢锁
	txn.If(clientv3.Compare(clientv3.CreateRevision(lockkey), "=", 0)).
		Then(clientv3.OpPut(lockkey, "", clientv3.WithLease(leaseId))).
		Else(clientv3.OpGet(lockkey))

	//提交事务
	if txnResp, err = txn.Commit(); err != nil {
		goto FAIL
	}
	// 5、成功返回，失败释放租约
	if !txnResp.Succeeded { //锁被占用
		err = common.ERR_LOCK_ALREADY_REQUIRED
		goto FAIL
	}

	//抢锁成功
	jobLock.leaseId = leaseId
	jobLock.cancelFunc = cancelFunc
	jobLock.isLocked = true
	return

FAIL:
	cancelFunc()                                  //取消自动续租
	jobLock.lease.Revoke(context.TODO(), leaseId) //释放租约
	return
}

//释放锁
func (jobLock *JobLock) UnLock() {
	if jobLock.isLocked {
		jobLock.cancelFunc()                                  //取消我们程序的自动续租协程
		jobLock.lease.Revoke(context.TODO(), jobLock.leaseId) //释放租约
	}

}
