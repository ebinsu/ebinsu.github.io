---
categories: [ Kubernetes, KEDA ]
tags: keda
---

## 引言

KEDA 提供的与 Redis Scaler 主要有以下两种：

- Redis Lists（支持 Cluster 和 Sentinel）
- Redis Streams（支持 Cluster 和 Sentinel）

对于需要更多 Redis 指标操作的场景，我们可以通过编写 [External Scaler](../keda-external-scaler-demo) 来实现相关逻辑。

## 场景

假设业务场景涉及多个异步计算任务，这些任务使用 Redis Lists 存储。生产服务每 2 秒生成 4 个任务，而消费服务每 2
秒处理一个任务。如果生产者和消费者服务各有一个实例，那么每 2 秒将累积 3 个任务。当消费者数量达到 4
个时，生产和消费能力达到平衡，任务堆积数量将不再增加。

## Redis List Scaler 与 HPA

Redis List Scaler 生成的 HPA（Horizontal Pod Autoscaler）指标类型为 AverageValue。根据上述场景，当消费者数量为 4
时，任务堆积数量将保持在 Scaler triggers 的 redis.metadata.listLength 所设置的预期值以内。随着时间的推移，任务堆积数量将被控制在
`4 * triggers.redis.metadata.listLength` 以内。

* 这个逻辑仅适用于当前场景。

例如：

设置 listLength 为 2，最终实例数量将扩展至 4 个，且任务堆积数量稳定在 8 个以下。
设置 listLength 为 4，最终实例数量将扩展至 4 个，且任务堆积数量稳定在 16 个以下。
设置 listLength 为 50，最终实例数量将扩展至 4 个，且任务堆积数量稳定在 200 个以下。
如果生产和消费的频率相同，并且已累积了大量任务，消费者实例将被扩展至 5 个或更多。随着时间的推移，任务将被清空，Redis list key
将被移除。Redis Lists Scaler 的 Active 状态受 Redis list key 是否存在的影响。当 list key 被移除时，Active 状态为 False。如果
minReplicas 设置为 1，实例数量将被设置为 1，随后又开始产生任务堆积，再次触发伸缩操作，最终稳定在 4 个实例。

## 时间与消费者实例产生的任务堆积示例

| 时间（秒） | 消费者实例 | 任务堆积 |
|-------|-------|------|
| 1     | 1     | 0    |
| 2     | 2     | 4    |
| 3     | 3     | 8    |
| 4     | 4     | 12   |
| ……    | ……    | ……   |
| 22    | 4     | 32   |
| 24    | 4     | 36   |
| 26    | 4     | 40   |
| 28    | 4     | 44   |

（注：表中数据为示例，实际情况可能有所不同）

## 代码清单

> ScaledObject.yaml

~~~yaml
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: async-task-redis-list
spec:
  scaleTargetRef:
    name: consumer
  minReplicaCount: 1
  maxReplicaCount: 20
  cooldownPeriod: 10
  triggers:
    - type: redis
      metadata:
        address: redis-nodeport.default:6379
        listName: async_task
        listLength: "50"
        activationListLength: "0"
  advanced:
    restoreToOriginalReplicaCount: true
    horizontalPodAutoscalerConfig:
      behavior:
        scaleUp:
          policies:
            - type: Pods
              value: 1
              periodSeconds: 60
        scaleDown:
          policies:
            - type: Pods
              value: 1
              periodSeconds: 30
~~~

> producer.yaml

~~~yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: producer
spec:
  replicas: 1
  selector:
    matchLabels:
      app: producer
  template:
    metadata:
      labels:
        app: producer
    spec:
      containers:
        - name: producer
          image: ebin/business-service:1.2
          env:
            - name: IS_PROCUDER
              value: "true"
~~~

> consumer.yaml

~~~yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: consumer
spec:
  replicas: 1
  selector:
    matchLabels:
      app: consumer
  template:
    metadata:
      labels:
        app: consumer
    spec:
      containers:
        - name: consumer
          image: ebin/business-service:1.2
~~~

> business-service main.go

~~~go
package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-redis/redis"
	"github.com/google/uuid"
)

var ASYNC_TASK_NAME = "async_task"

func main() {
	c := make(chan os.Signal)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	rdb := redis.NewClient(&redis.Options{
		Addr: "redis-nodeport:6379",
		// Addr:     "192.168.136.128:30091",
		Password: "",
		DB:       0,
	})

	isProcuder := os.Getenv("IS_PROCUDER")

	if isProcuder != "" {
		go func() {
			for {
				produce(*rdb)
			}
		}()
	}

	if isProcuder == "" {
		go func() {
			for {
				consume(*rdb)
			}
		}()
	}

	<-c
}

func produce(rdb redis.Client) {
	<-time.After(2 * time.Second)
	log.Println("produce async task...")
	rdb.RPush(
		ASYNC_TASK_NAME,
		uuid.New().String(),
		uuid.New().String(),
		uuid.New().String(),
		uuid.New().String(),
	)
}

func consume(rdb redis.Client) {
	val, err := rdb.LPop(ASYNC_TASK_NAME).Result()
	if err == nil && val != "" {
		log.Println("consume task :" + val)
		<-time.After(2 * time.Second)
	}
}
~~~
