# Prometheus monitoring for Hertz

## Abstract
prometheus大概的工作流程：
1. Prometheus server 定期从配置好的 jobs 或者 exporters 中拉（pull模式） metrics，或者接收来自 Pushgateway 发过来（push模式）的 metrics，或者从其他的 Prometheus server 中拉 metrics；
2. Prometheus server 在本地存储收集到的 metrics，并运行已定义好的 alert.rules，记录新的时间序列或者向 Alertmanager 推送警报；
3. Alertmanager 根据配置文件，对接收到的警报进行处理，发出告警；
4. 在图形界面中，可视化采集数据，例如对接Grafana。

### 数据模型
Prometheus 中存储的数据为时间序列，是由 metric 的名字和一系列的标签（键值对）唯一标识的，不同的标签则代表不同的时间序列。
- name：一般用于表示 metric 的功能；注意，metric 名字由 ASCII 字符，数字，下划线，以及冒号组成，必须满足正则表达式 [a-zA-Z_:][a-zA-Z0-9_:]*；
- tag：标识了特征维度，便于过滤和聚合。例如 PSM 和 method 等信息。tag 中的 key 由 ASCII 字符，数字，以及下划线组成，必须满足正则表达式 [a-zA-Z_:][a-zA-Z0-9_:]*；
- sample：实际的时间序列，每个序列包括一个 float64 的值和一个毫秒级的时间戳；
- metric：通过如下格式表示：<metric name>{<label name>=<label value>, ...}

### Metric类型

#### Counter
- 可以理解为只增不减的计数器，典型的应用如：请求的个数，结束的任务数， 出现的错误数等等；
- 对应 gopkg/metrics 的 EmitStore。

#### Gauge
- 一种常规的 metric，典型的应用如：goroutines 的数量；
- 可以任意加减；
- 对应 gopkg/metrics 的 EmitCounter。

#### Histogram
- 生成直方图数据，用于统计和分析样本的分布情况，典型的应用如：pct99，CPU 的平均使用率等；
- 可以对观察结果采样，分组及统计。
- 对应 gopkg/metrics 的 EmitTimer。

#### Summary
- 类似于 Histogram，提供观测值的 count 和 sum 功能；
- 提供百分位的功能，即可以按百分比划分跟踪结果；
- Sumamry 的分位数是直接在客户端计算完成，因此对于分位数的计算而言，Summary 在通过 PromQL 进行查询时有更好的性能表现，而 Histogram 则会消耗更多的资源，对于客户端而言 Histogram 消耗的资源更少。

## Labels
- method - HTTP 的方法
- statusCode - HTTP 状态码

## Metrics
- Server 端处理的请求总数：
    - Name: hertz_server_throughput
    - Tags: method, statusCode
- Server 端请求处理耗时（处理完请求时间 - 收到请求时间，单位 us）：
    - Name: hertz_server_latency_us
    - Tags: method, statusCode

## Useful Examples
Prometheus 的查询语法可以参考 [Querying basics | Prometheus](https://prometheus.io/docs/prometheus/latest/querying/basics/), 这里给出一些常用示例：

**server throughput of succeed requests**
```
sum(rate(hertz_server_throughput{statusCode="200"}[1m])) by (method)
```

**server latency pct99 of succeed requests**
```
histogram_quantile(0.9,sum(rate(hertz_server_latency_us_bucket{statusCode="200"}[1m]))by(le))
```

## Usage Example
### Server

```
import (
   "context"
	"time"

	"github.com/hertz-contrib/monitor-prometheus"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
)

func main() {
...
	h := server.Default(server.WithHostPorts("127.0.0.1:8080"), server.WithTracer(prometheus.NewServerTracer(":9091", "/hertz")))

	h.GET("/metricGet", func(c context.Context, ctx *app.RequestContext) {
		ctx.String(200, "hello get")
	})

	h.POST("/metricPost", func(c context.Context, ctx *app.RequestContext) {
		time.Sleep(100 * time.Millisecond)
		ctx.String(200, "hello post")
	})

	h.Spin()
...
}
```
## 可视化界面
### 安装 Prometheus 和 Grafana

Hertz 已经写了一个 docker-compose.yml 和 Prometheus 的配置文件 prometheus.yml 的 demo，只需简单配置即可完成

1. 进入当前目录，将 prometheus.yml 中第30行的 $inetIP 改为内网 IP 即可。注意，不需要加 ``http://``
2. 启动 docker
```
docker-compose up
```
3. 浏览器访问 `http://localhost:3000`, 账号密码默认都是 `admin`
4. 配置数据源 `Configuration` ->`Data Source` -> `Add data source`，配置后点击 `Save & Test` 测试验证是否生效
5. 添加监控界面 `Create` -> `dashboard`，根据自己的需求添加 throughput 和 pct99 等监控指标，可以参考上面 `Useful Examples` 给出的样例。
