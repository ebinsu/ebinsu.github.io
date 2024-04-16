package externalscaler

import (
	context "context"
	"strconv"

	"github.com/go-redis/redis"
)

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
	_, err := e.rdb.Ping().Result()
	return &IsActiveResponse{
		Result: err == nil,
	}, nil
}

func (e *RedisSessionExternalScaler) GetMetricSpec(_ context.Context, ref *ScaledObjectRef) (*GetMetricSpecResponse, error) {
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
