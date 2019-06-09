# crontab

#### 项目结构
* /master
* /worker
* /common

### master
* 配置文件，命令行参数，线程配置
* 给web后台提供http API，用于管理job
* web前端页面，bootstrap+jquery，前后端分离开发

### worker
* 从etcd中同步job到内存中
* 实现调度模块，基于cron表达式调度N个job
* 实现执行模块，并发的执行多个job
* 对job的分布式锁，防止集群并发
* 把执行日志保存到mongodb