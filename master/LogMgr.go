package master

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"imooc.com/go-crontab/crontab/common"
	"time"
)

type LogMgr struct {
	client        *mongo.Client
	logCollection *mongo.Collection
}

var (
	G_LogMgr *LogMgr
)

func InitLogMgr() (err error) {
	var (
		client *mongo.Client
	)

	if client, err = mongo.Connect(context.TODO(),
		options.Client().ApplyURI(G_config.MongodbUrl),
		options.Client().SetConnectTimeout(time.Duration(G_config.MongodbTimeout)*time.Millisecond)); err != nil {
		return
	}
	G_LogMgr = &LogMgr{
		client:        client,
		logCollection: client.Database("cron").Collection("log"),
	}
	return
}

func (lm *LogMgr) ListLog(name string, skip, limit int) (logs []*common.JobLog, err error) {
	var (
		filter  *common.JobLogFilter
		logSort *common.SortLogByStartTime
		findOpt *options.FindOptions
		limit64 int64
		skip64  int64
		cursor  *mongo.Cursor
		log     *common.JobLog
	)

	// 防止出现空指针，初始化logs 外部判断len(logs)
	logs = make([]*common.JobLog, 0)

	limit64 = int64(limit)
	skip64 = int64(skip64)
	// 过滤条件
	filter = &common.JobLogFilter{JobName: name}

	// 按照任务开始时间倒叙排列
	logSort = &common.SortLogByStartTime{SortOrder: -1}

	// 查询选项
	findOpt = &options.FindOptions{
		Limit: &limit64,
		Skip:  &skip64,
		Sort:  logSort,
	}

	if cursor, err = lm.logCollection.Find(context.TODO(), filter, findOpt); err != nil {
		return
	}
	// 释放游标
	defer cursor.Close(context.TODO())

	for cursor.Next(context.TODO()) {
		// 定义一个日志对象
		log = &common.JobLog{}

		if err = cursor.Decode(log); err != nil {
			fmt.Println("有日志不合法:", err)
			continue
		}
		logs = append(logs, log)
	}
	return
}
