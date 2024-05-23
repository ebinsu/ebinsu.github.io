---
categories: [ KEDA ]
tags: [ k8s, keda ]
---

## 引言

KEDA 提供了一套 gRPC 接口，供用户编写用于获取指标的 gRPC 服务。
KEDA 的 External Scaler 和 External Push Scaler 在 KEDA 的 Pod 中运行，作为 gRPC 客户端，从用户编写的 gRPC 服务中获取指标。

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

External 与 External Push Scaler 的区别

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

当 scaleHandler `HandleScalableObject` 时，如果 ScalerObject 类型是 PushScaler，则会通过 startPushScalers 方法启动一个线程，线程里调用 externalPushScaler 的Run方法。
该方法通过 GRPC 请求我们实现的 StreamIsActive 方法，返回的 resp.Result （true/false）决定了 ScalerObject 的 Active 状态，也就是触发 0 -> 1 / 1 -> 0 的逻辑。
其他类型的ScalerObject，则只能通过 startScaleLoop 方法处理0 ->1 / 1 ->0 的逻辑。

externalPushScaler handleIsActiveStream方法：
~~~go
func handleIsActiveStream(ctx context.Context, scaledObjectRef *pb.ScaledObjectRef, grpcClient pb.ExternalScalerClient, active chan<- bool) error {
	stream, err := grpcClient.StreamIsActive(ctx, scaledObjectRef)
	if err != nil {
		return err
	}

	for {
		resp, err := stream.Recv()
		if err != nil {
			return err
		}

		active <- resp.Result
	}
}

~~~

StreamIsActive 例子：
~~~go
func (e *RedisSessionExternalScaler) StreamIsActive(ref *ScaledObjectRef, epsServer ExternalScaler_StreamIsActiveServer) error {
  // 每 10 秒发送Result状态。Result 的值由 redis active key 决定。
	for {
		select {
		case <-epsServer.Context().Done():
			// call cancelled
			return nil
		case <-time.NewTicker(10 * time.Second).C:
			val, err := e.rdb.Exists("active").Result()
			if err != nil {
				// log error
			} else if val > 0 {
				active, err := e.rdb.Get("active").Result()
				if err != nil {
					// log error
				} else {
					epsServer.Send(&IsActiveResponse{
						Result: active == "1",
					})
				}
			}
		}
	}
}
~~~

## 场景

假设应用的会话（session）由 Redis 存储，我们希望每个应用实例最多只处理 100 个会话请求。当会话数量超过 100 时，启动一个新的实例。若
Redis 实例不可用，则缩放回 1 个实例。示例场景如下：

eg：

| 会话数量   | 实例数 |
|--------|-----|
| 10     | 1   |
| 1-100	 | 1   |
| 101    | 2   |
| ...    | ... | 

## 实现

基于上述场景，我们使用外部伸缩器来实现，对应的 gRPC 服务需要实现 IsActive、GetMetricSpec、GetMetrics 这三个接口：

- IsActive 用于检查 Redis 实例是否可用。若不可用，返回 False，并设置 ScaledObject.spec.minReplicaCount 为 1。
- GetMetricSpec 定义指标名称为 session_size，目标大小从 ScaledObject 的 Metadata[sessionSize] 中获取。同时，ScaledObject
  声明的 metricType 应为 AverageValue。
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
