package master

import (
	"context"
	"encoding/json"
	"github.com/BensonMax/crontab/common"
	"go.etcd.io/etcd/clientv3"
	"go.etcd.io/etcd/mvcc/mvccpb"
	"time"
)

type JobMgr struct {
	client *clientv3.Client
	kv     clientv3.KV
	lease  clientv3.Lease
}

var (
	//单例
	G_JobMgr *JobMgr
)

func InitJobMgr() (err error) {
	var (
		config clientv3.Config
		client *clientv3.Client
		kv     clientv3.KV
		lease  clientv3.Lease
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

	//赋值单例
	G_JobMgr = &JobMgr{
		client: client,
		kv:     kv,
		lease:  lease,
	}

	return
}

//面向对象，为JobMgr新建一个SaveJob方法，传入参数job 返回 olbjob，error
func (JobMgr *JobMgr) SaveJob(job *common.Job) (oldjob *common.Job, err error) {
	//把任务保存到/cron/job/任务名 ->json
	var (
		jobKey    string
		jobValue  []byte
		putResp   *clientv3.PutResponse
		oldjobObj common.Job
	)
	//etcd key
	jobKey = common.JOB_SAVA_DIR + job.Name
	//任务信息job
	if jobValue, err = json.Marshal(job); err != nil {
		return
	}

	if putResp, err = JobMgr.kv.Put(context.TODO(), jobKey, string(jobValue), clientv3.WithPrevKV()); err != nil {
		return
	}

	if putResp.PrevKv != nil {
		//	对旧值进行反序列化
		if err = json.Unmarshal(putResp.PrevKv.Value, &oldjobObj); err != nil {
			err = nil
			return
		}
		oldjob = &oldjobObj
	}
	return
}

//删除任务
func (JobMgr *JobMgr) DeleteJob(name string) (oldJob *common.Job, err error) {
	var (
		jobKey    string
		delResp   *clientv3.DeleteResponse
		oldJobObj common.Job
	)

	jobKey = common.JOB_SAVA_DIR + name

	//从etcd 中删除
	if delResp, err = JobMgr.kv.Delete(context.TODO(), jobKey, clientv3.WithPrevKV()); err != nil {
		return
	}
	//返回被删除的任务信息
	if len(delResp.PrevKvs) != 0 {
		//解析一下旧值，返回
		if err = json.Unmarshal(delResp.PrevKvs[0].Value, &oldJobObj); err != nil {
			err = nil
			return
		}
		oldJob = &oldJobObj
	}

	return
}

//获取List
func (JobMgr *JobMgr) ListJobs() (jobList []*common.Job, err error) {
	var (
		dirKey  string
		getResp *clientv3.GetResponse
		kvPair  *mvccpb.KeyValue
		job     *common.Job
	)
	//保存任务信息
	dirKey = common.JOB_SAVA_DIR
	//获取任务下所有的信息
	if getResp, err = JobMgr.kv.Get(context.TODO(), dirKey, clientv3.WithPrefix()); err != nil {
		return
	}

	//初始化数组空间
	jobList = make([]*common.Job, 0)
	//len(jobList) == 0

	//遍历所有任务，进行反序列化
	for _, kvPair = range getResp.Kvs {
		job = &common.Job{}
		if err = json.Unmarshal(kvPair.Value, job); err != nil {
			err = nil
			continue
		}

		jobList = append(jobList, job)
	}
	return
}

//杀死任务
func (JobMgr *JobMgr) KillJob(name string) (err error) {
	//更新一下key = /cron/killer/任务名
	var (
		killerKey      string
		leaseGrantResp *clientv3.LeaseGrantResponse
		leaseID        clientv3.LeaseID
	)

	//通知walker 杀死对应的任务
	killerKey = common.JOB_KILLER_DIR + name

	//让worker 监听到一次put操作即可，创建一个租约让其稍后自动获取
	if leaseGrantResp, err = JobMgr.lease.Grant(context.TODO(), 1); err != nil {
		return
	}
	//租约ID
	leaseID = leaseGrantResp.ID

	//设置killer标记
	if _, err = JobMgr.kv.Put(context.TODO(), killerKey, "", clientv3.WithLease(leaseID)); err != nil {
		return
	}
	return
}
