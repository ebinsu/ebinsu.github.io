---
categories: [ Kubernetes, KEDA ]
tags: keda
---

## 场景

假设有以下业务场景：
当用户进行下单操作，订单服务会创建一个订单，同时通知下游服务计算订单的ETA，ETA 是通过消息传递且 EAT 会一致变动，每次变动都是一条新的消息。
期望当订单数量到达一定数量时，提前为订单服务进行扩容，避免消息的堆积。

例如我们以每个订单服务处理 50 个订单的 ETA 消息为条件，当5分钟内的订单数超过 50 个时，就进行扩容，扩容按当前的数量倍数增加。 
如果最近 5 分钟没有下订单，则缩容为 1 个实例，缩容时15秒内一次最多缩容 1 个实例。

## 代码清单

### ScaledObject.yaml

~~~yaml
apiVersion: v1
kind: Secret
metadata:
  name: mysql-secrets
type: Opaque
data:
  mysql_conn_str: cm9vdDplYmluc3UxMjM0NSFAdGNwKDE5Mi4xNjguMTM2LjEyODozMDA5MCkvb3JkZXI=
---
apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: keda-trigger-auth-mysql-secret
spec:
  secretTargetRef:
    - parameter: connectionString
      name: mysql-secrets
      key: mysql_conn_str
---
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: mysql-scaledobject
spec:
  scaleTargetRef:
    name: demo-app
  minReplicaCount: 1
  maxReplicaCount: 10
  triggers:
    - type: mysql
      metadata:
        queryValue: "50"
        activationQueryValue: "1"
        query: "select count(*) t from orders o where o.created_time > DATE_SUB(NOW(), INTERVAL 5 MINUTE)"
      authenticationRef:
        name: keda-trigger-auth-mysql-secret
  advanced:
    restoreToOriginalReplicaCount: true
    horizontalPodAutoscalerConfig:
      behavior:
        scaleUp:
          stabilizationWindowSeconds: 0
          policies:
            - type: Percent
              value: 100
              periodSeconds: 15
        scaleDown:
          stabilizationWindowSeconds: 300
          policies:
            - type: Pods
              value: 1
              periodSeconds: 15
~~~

### order table
~~~sql
CREATE TABLE `order`.orders (
 `id`            VARCHAR(50)  NOT NULL,
 `order_number`  VARCHAR(50)  NOT NULL,
 `created_time`  TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
 PRIMARY KEY (`id`)
)
ENGINE=InnoDB;
~~~

### Mock Place Order

~~~go
package main

import (
	"database/sql"
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
)

func main() {

	db, err := sql.Open("mysql", "root:ebinsu12345!@tcp(192.168.136.128:30090)/order")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	stmtIns, err := db.Prepare("INSERT INTO orders (id, order_number, created_time) VALUES( ? , ?, now())")
	if err != nil {
		panic(err.Error())
	}
	defer stmtIns.Close()

	orderNumber := 1
	for {
		id := uuid.New().String()
		_, err = stmtIns.Exec(id, orderNumber)
		if err != nil {
			panic(err.Error())
		}
		log.Println("place an order.")
		orderNumber++
		<-time.After(2 * time.Second)
	}
}
~~~
