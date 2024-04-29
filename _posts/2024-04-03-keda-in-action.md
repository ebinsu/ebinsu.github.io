---
categories: [ KEDA ]
tags: [ k8s, keda ]
---

## 引言
在我们学习 KEDA 之前，我们需要简单了解一些关于HPA的知识。本文还将从 Demo 与源码入手，进一步了解 KEDA 的实现原理。

## 1. HPA 是如何扩缩容？

1. 从聚合API（metrics.k8s.io、custom.metrics.k8s.io或external.metrics.k8s.io）获取指标。

2. 从scaleTargetRef获取扩缩容的目标资源（例如：Deployment、StatefulSet等。这些资源都有一个名为scale的子资源，该接口允许你动态设置副本的数量并检查它们的当前状态）。 

3. kube-controller-manager根据扩缩容的目标的资源使用率的平均值或原始值计算出扩缩的目标值，进而确定目标副本数。

## 2. Metrics 如何获取？

- 对于资源指标，使用metrics.k8s.io API，通常由metrics-server提供。它可以作为集群插件启动。

- 对于自定义指标，使用custom.metrics.k8s.io API。它由其他度量指标方案厂商的“适配器（Adapter）”API服务器提供。

- 对于外部指标，使用external.metrics.k8s.io API。这可能也是由上述自定义指标适配器提供的。

## 3. KEDA 是什么

1. KEDA 提供了一套自定义资源定义（CRD），其常见用途是声明ScaledObject。
keda-operator 根据声明的 ScaledObject 创建对应的HPA，并能激活HPA的部署（从0扩展到1）。[代码分析](../keda-operator-analysis)
> * 这里的“0 -> 1”指的是，如果当前副本数为0，但HPA定义的最小副本数不为0，则认为关闭了弹性伸缩能力，HPA将不再进行伸缩操作。 而 KEDA 实现了一种控制，一旦缩放到1后，伸缩操作将取决于HPA。

2. KEDA 提供了 keda-operator-metrics-apiserver，实现了 external.metrics.k8s.io API ，为 HPA 获取 Kubernetes 集群外部的指标源的指标。 [代码分析](../keda-operator-metrics-apiserver-analysis)
> * 想要了解当前注册了哪些外部指标服务器，可以使用以下命令：
~~~
kubectl get APIService/v1beta1.external.metrics.k8s.io
~~~
> * 想要通过外部指标服务器获取外部指标，可以使用以下命令：
~~~
kubectl get scaledobject {scaledObjectName} -o jsonpath={.status.externalMetricNames}
kubectl get --raw /apis/external.metrics.k8s.io/v1beta1/namespaces/{namespaces}/{externalMetricNames}?labelSelector=scaledobject.keda.sh/name={scaledObjectName}
~~~

3. KEDA 与 HPA并无冲突关系，它们可以同时在 Kubernetes 中使用。但请注意，不要将 KEDA 的 ScaledObject 与 HPA 结合使用来扩展相同的目标资源（例如：Deployment、StatefulSet等）。

## 4. Demo

- [External Scalers](../keda-external-scaler-demo)

- [Redis Scalers](../keda-redis-scaler-demo)

- [Mysql Scalers](../keda-mysql-scaler-demo)
