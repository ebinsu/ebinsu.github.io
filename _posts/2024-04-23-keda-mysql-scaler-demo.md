---
categories: [ Kubernetes, KEDA ]
tags: keda
---

## 场景

考虑以下业务场景：当用户下单时，订单服务会创建一个新的订单，并通知下游服务计算订单的预计到达时间（ETA）。ETA
通过消息传递，并会持续变化，每次变化都生成一条新消息。我们希望在订单数量达到一定阈值时，能够预先对订单服务进行扩容，以避免消息积压。

具体来说，我们设定当5分钟内的订单数量超过50个时触发扩容，扩容操作将按当前实例数量的倍数增加。如果最近5分钟内没有新的订单，服务将缩容至1个实例。在缩容过程中，15秒内最多只减少1个实例。

## 实现

我们主要关注 Mysql Scaler 的这两个配置：
- queryValue: "50" # 在 HPA 用作 targetValue 或 targetAverageValue（取决于触发器指标类型）的阈值。
- activationQueryValue: "1" # 激活 Scaler 的目标值。在这里了解更多关于激活的信息。（默认值：0，可选，这个值可以是一个浮点数）

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
