---
categories: [Kubernetes, KEDA]
tags: keda
---

## 前言
KEDA 提供了一套 GRPC 接口来供用户编写获取指标的 GRPC 服务。 KEDA 的 External Scaler 与 External Push Scaler 在 KEDA 的 pod 中运行 ，它作为 GRPC 客户端，从用户编写的 GRPC 服务获取指标。

GRPC 接口定义:

~~~grpc
service ExternalScaler {
    // 给 Scaler (External Scaler) 的 GetMetricsAndActivity 方法调用，
    // 返回结果决定了 Scaler Object 的 Active 状态。
    rpc IsActive(ScaledObjectRef) returns (IsActiveResponse) {}
    // 给 PushScaler (External Push Scaler) 的 Run 方法调用，
    // 返回结果决定了 Scaler Object 的 Active 状态。
    rpc StreamIsActive(ScaledObjectRef) returns (stream IsActiveResponse) {}
    // 给 Scaler 的 GetMetricSpecForScaling 方法调用，返回指标的名称与目标值，
    // 对应生成 HPA 的 metric.name 与 target.value
    rpc GetMetricSpec(ScaledObjectRef) returns (GetMetricSpecResponse) {}
    // 给 Scaler 的 GetMetricsAndActivity 方法调用，返回请求时的指标值。
    rpc GetMetrics(GetMetricsRequest) returns (GetMetricsResponse) {}
}
~~~

External  与 External Push Scaler 的区别

~~~go
type externalScaler struct {
	metricType      v2.MetricTargetType
	metadata        externalScalerMetadata
	scaledObjectRef pb.ScaledObjectRef
	logger          logr.Logger
}

type externalPushScaler struct {
	externalScaler
}

func (s *externalPushScaler) Run(ctx context.Context, active chan<- bool) {}
~~~

externalScaler 实现了 PushScaler 的 Run 方法。 当 scaleHandler HandleScalableObject 的时候，如果 Scaler 的类型是 PushScaler ，会执行 Run 方法。
这区别于其他类型的 Scaler 只能通过 startScaleLoop 处理缩放的逻辑，它还可以通过 StreamIsActive 返回值来触发缩放的逻辑。

## 场景

假设应用的 session 是由 Redis 存储，期望每个应用的实例最多只处理 100 个 session 的请求。 当 session 的数量大于 100 的时候，会启动一个新的实例。
当 Redis 实例不可用时，缩放回 1 个实例。

eg：

| session count | instance |
|---------------|----------|
| 10            | 1        |
| 100           | 1        |
| 150           | 2        |
| ...           | ...      |
| 200           | 2        |
| 250           | 3        |

基于上述场景，我们使用 External Scaler 来实现，对应的 GRPC 服务需实现 IsActive、GetMetricSpec、GetMetrics 这三个接口。
 - IsActive 检验 Redis 实例是否可用，不可用时返回 False，且 ScaledObject.spec.minReplicaCount = 1；idleReplicaCount = null。
 - GetMetricSpec 定义 Metric Name 为 session_size，target size 从声明 ScaledObject 的 Metadata[sessionSize] 中获取；并且ScaledObject 声明的 metricType 要为 AverageValue。
 - GetMetrics 返回请求时 Redis 实例的 DBSize。

## GRPC 服务端-代码清单
### RedisSessionExternalScaler.go
~~~go
const (
	metricName               = "session_size"
	defaultSessionSize int64 = 100
)

type RedisSessionExternalScaler struct {
	UnimplementedExternalScalerServer
	rdb redis.Client
}

func NewScaler(rdb redis.Client) *RedisSessionExternalScaler {
	return &RedisSessionExternalScaler{UnimplementedExternalScalerServer{}, rdb}
}

func (e *RedisSessionExternalScaler) IsActive(ctx context.Context, scaledObject *ScaledObjectRef) (*IsActiveResponse, error) {
  // redis 实例可用返回true
	_, err := e.rdb.Ping().Result()
	return &IsActiveResponse{
		Result: err == nil,
	}, nil
}

func (e *RedisSessionExternalScaler) GetMetricSpec(_ context.Context, ref *ScaledObjectRef) (*GetMetricSpecResponse, error) {
 // 返回 Metric 名称与指标值， 指标值从 ScaledObject Metadata 中定义的 sessionSize 获取
	sessionSize := defaultSessionSize
	if val, ok := ref.ScalerMetadata["sessionSize"]; ok {
		size, err := strconv.ParseInt(val, 10, 64)
		if err == nil {
			sessionSize = size
		}
	}
	return &GetMetricSpecResponse{
		MetricSpecs: []*MetricSpec{{
			MetricName: metricName,
			TargetSize: sessionSize,
		}},
	}, nil
}

func (e *RedisSessionExternalScaler) GetMetrics(_ context.Context, metricRequest *GetMetricsRequest) (*GetMetricsResponse, error) {
// 获取具体的指标值，在这个场景中返回 Redis 的 DBSize
	val, err := e.rdb.DBSize().Result()
	var mv int64 = 0
	if err == nil {
		mv = val
	}

	return &GetMetricsResponse{
		MetricValues: []*MetricValue{{
			MetricName:  metricName,
			MetricValue: mv,
		}},
	}, nil
}
~~~

## 完整代码清单

[github](https://github.com/ebinsu/ebinsu.github.io/tree/main/example/keda/redis-session-scaler-sample)
