package master

import (
	"context"
	"encoding/json"
	"github.com/BensonMax/crontab/common"
	"go.etcd.io/etcd/clientv3"
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
	jobKey = "/cron/jobs/" + job.Name
	//任务信息job
	if jobValue, err = json.Marshal(job); err != nil {
		return
	}

	if putResp, err = JobMgr.kv.Put(context.TODO(), jobKey, string(jobValue), clientv3.WithPrevKV()); err != nil {
		return
	}

	if putResp.PrevKv != nil {
		//	对旧址进行反序列化
		if err = json.Unmarshal(putResp.PrevKv.Value, &oldjobObj); err != nil {
			err = nil
			return
		}
		oldjob = &oldjobObj
	}
	return
}
