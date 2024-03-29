package worker

import (
	"encoding/json"
	"io/ioutil"
)

// 程序配置
type Config struct {
	EtcdEndPoints       []string `json:"etcdEndPoints"`
	EtcdDialTimeout     int      `json:"etcdDialTimeout"`
	MongodbUrl          string   `json:"mongodbUrl"`
	MongodbTimeout      int      `json:"mongodbConnectTimeout"`
	JobLogBatchSize     int      `json:"jobLogBatchSize"`
	JobLogCommitTimeout int      `json:"jobLogCommitTimeout"`
}

var (
	G_config *Config
)

func InitConfig(filename string) (err error) {
	var (
		content []byte
		conf    Config
	)
	// 读配置文件
	if content, err = ioutil.ReadFile(filename); err != nil {
		return
	}

	// 反序列化
	if err = json.Unmarshal(content, &conf); err != nil {
		return
	}

	// 赋值单例
	G_config = &conf

	return
}
