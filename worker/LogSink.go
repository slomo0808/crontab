package worker

import (
	"context"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"imooc.com/go-crontab/crontab/common"
	"time"
)

// mongodb存储日志
type LogSink struct {
	client         *mongo.Client
	logCollection  *mongo.Collection
	logChan        chan *common.JobLog
	autoCommitChan chan *common.LogBatch
}

var (
	G_LogSink *LogSink
)

func (ls *LogSink) saveLogs(logBatch *common.LogBatch) {
	ls.logCollection.InsertMany(context.TODO(), logBatch.Logs)
}

func (ls *LogSink) writeLoop() {
	var (
		log          *common.JobLog
		logBatch     *common.LogBatch
		commitTimer  *time.Timer
		timeoutBatch *common.LogBatch
	)
	for {
		select {
		case log = <-ls.logChan:
			// 每次插入需要等待mongodb的一次请求往返，耗时比较长的时间
			if logBatch == nil {
				logBatch = &common.LogBatch{}
				// 让这个批次超时自动提交,给1秒
				commitTimer = time.AfterFunc(
					time.Duration(G_config.JobLogCommitTimeout)*time.Millisecond,
					func(batch *common.LogBatch) func() {
						return func() {
							ls.autoCommitChan <- batch
						}
					}(logBatch))
			}
			// 把新的日志追加到logBatch中
			logBatch.Logs = append(logBatch.Logs, log)
			// 如果批次满了，立即发送
			if len(logBatch.Logs) >= G_config.JobLogBatchSize {
				// 发送日志
				ls.saveLogs(logBatch)
				// 清空logBatch
				logBatch = nil
				// 取消定时器
				commitTimer.Stop()
			}
		case timeoutBatch = <-ls.autoCommitChan: // 批次过期
			// 判断过期批次是否仍旧是当前批次
			if timeoutBatch != logBatch {
				// 证明logBatch已经被提交过了
				continue
			}
			// 提交到mongodb
			ls.saveLogs(timeoutBatch)
			// 清空 logBatch
			logBatch = nil
		}
	}
}

func InitLogSink() (err error) {
	var (
		client *mongo.Client
	)

	if client, err = mongo.Connect(context.TODO(),
		options.Client().ApplyURI(G_config.MongodbUrl),
		options.Client().SetConnectTimeout(time.Duration(G_config.MongodbTimeout)*time.Millisecond)); err != nil {
		return
	}
	G_LogSink = &LogSink{
		client:         client,
		logCollection:  client.Database("cron").Collection("log"),
		logChan:        make(chan *common.JobLog, 1000),
		autoCommitChan: make(chan *common.LogBatch, 1000),
	}

	// 启动一个协程
	go G_LogSink.writeLoop()
	return
}

// 发送日志
func (ls *LogSink) Append(log *common.JobLog) {
	select {
	case ls.logChan <- log:
	default:
		// 队列满了就丢弃
	}

}
