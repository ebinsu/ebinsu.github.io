---
categories: [K8S, KEDA]
tags: keda
---

# keda 与 hpa关系
## 1. hpa 是什么
HorizontalPodAutoscaler 的常见用途是将其配置为从 Aggregated APIs （metrics.k8s.io、custom.metrics.k8s.io 或 external.metrics.k8s.io）获取指标。 从scaleTargetRef 获取定义的目标资源（例如：Deployment 和 StatefulSet等），这些资源每个都有一个名为 scale 的子资源 ，该接口允许你动态设置副本的数量并检查它们的每个当前状态。接下来，kube-controller-manager 根据平均的资源使用率或原始值计算出扩缩的比例，进而计算出目标副本数。

## 2. metrics 如何获取？
- 对于资源指标，将使用 metrics.k8s.io API， 一般由 metrics-server 提供。 它可以作为集群插件启动。

- 对于自定义指标，将使用 custom.metrics.k8s.io API。 它由其他度量指标方案厂商的“适配器（Adapter）” API 服务器提供。 检查你的指标管道以查看是否有可用的 Kubernetes 指标适配器。

- 对于外部指标，将使用 external.metrics.k8s.io API。 可能由上面的自定义指标适配器提供。

## 3. keda 是什么
1. KEDA 提供了一套 CRD ，常见用途是声明 ScaledObject ，keda-operator 按声明 ScaledObject 创建对应的HPA ，并能激活 HPA 的部署（0 -> 1）。[源码分析](../keda-operator-%E6%BA%90%E7%A0%81%E5%88%86%E6%9E%90)
> * 0 -> 1 指的是，如果当前副本数为 0 但是 hpa 定义的最小 replicas 不等0 时，被认为关闭弹性伸缩的能力，HPA 不会再进行伸缩操作。而 KEDA 实现了个控制，但一旦缩放到 1 后，伸缩操作取决于 HPA。

2. KEDA 提供 keda-operator-metrics-apiserver，实现 external.metrics.k8s.io API ，为 HPA 获取 Kubernetes 集群外部的指标源的指标。 [源码分析](../keda-operator-metrics-apiserver-%E6%BA%90%E7%A0%81%E5%88%86%E6%9E%90)

> * 想知道当前注册了哪些外部指标服务器
~~~
kubectl get APIService/v1beta1.external.metrics.k8s.io
~~~

> * 想获取外部指标
~~~
kubectl get scaledobject {scaledObjectName} -o jsonpath={.status.externalMetricNames}
kubectl get --raw /apis/external.metrics.k8s.io/v1beta1/namespaces/{namespaces}/{externalMetricNames}?labelSelector=scaledobject.keda.sh/name={scaledObjectName}
~~~

#### KEDA 与 HPA 结合使用

KEDA 与 HPA 并无冲突关系，可以同时再 Kubernetes 使用，但不要将 KEDA ScaledObject 与 HPA 结合使用来扩展相同的目标资源（例如：Deployment 和 StatefulSet等）。

## 4. Demo
- External Scalers

todo..
