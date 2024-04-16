---
categories: [ Kubernetes, KEDA ]
tags: keda
---

## 前言

KEDA 提供的与 Redis 相关的 Scaler 只有两种： Redis Lists （ Cluster / Sentinel） ， Redis Streams （ Cluster / Sentinel）。

如果需要更多 Redis 的指标操作的场景，则只能使用 External Scaler [（Demo）](../keda-external-scaler-demo) 编写相关的 Redis
逻辑来实现。

## 场景

假设业务场景会产生多个异步计算任务，异步计算任务采用 Redis Lists 存储。 每个生产服务每 2 秒会产生 4 个任务，每个消费服务每2秒消费一个任务。

如果生产者、消费者服务各 1 个，那么每 2 秒就会堆积 3 个任务。 当消费者数量为 4 时，生产与消费能力持平，不会扩大堆积任务数量。

Redis List Scaler 生成的 HPA metrics type 为 AverageValue。 由上述条件可知，当为 4 时不会扩大堆积任务数量，
所以可以通过 triggers.redis.metadata.listLength 控制预期任务堆积数量。 随着时间的进行，堆积数量会控制在 <= 4 * triggers.redis.metadata.listLength 。

* 仅对本场景适用

例如设置为 2 ，最终会扩容至 4 个实例，并且堆积任务稳定在 8 个以下；
例如设置为 4 ，最终会扩容至 4 个实例，并且堆积任务稳定在 16 个以下；
例如设置为 50 ，最终会扩容至 4 个实例，并且堆积任务稳定在 200 个以下；

由于生产于消费的频率一致，如果已经堆积了大量的任务，消费者会被扩容至 5 个及以上，随着时间推移，任务将被清空，redis list key 会被 removed.
而 Redis Lists Scaler Active 的状态受到 redis list key 是否存在影响。 当 list key 被 removed， Active为 Flase 。
假设 minReplicas 为 1 ， replicas 会被设置为 1 ，这时又会开始产生堆积任务，再次触发扩缩容，最终稳定在 4 个实例。

* 时间与 consumer 实例产生的堆积任务表：

| time | 1 consumer | 2 consumer | 3 consumer | 4 consumer |
|------|------------|------------|------------|------------|
| 2s   | 3          | 2          | 1          | 0          |
| 4s   | 6          | 4          | 2          | 0          |
| 6s   | 9          | 6          | 3          | 0          |
| ...  | ...        | ...        | ...        | ...        |
| 22s  | 33         | 24         | 11         | 0          |
| 24s  | 36         | 26         | 12         | 0          |
| 26s  | 39         | 28         | 14         | 0          |

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
