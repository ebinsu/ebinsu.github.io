---
title: Keda Metrics ApiServer 代码分析
categories: [Kubernetes, KEDA]
tags: keda
---

## 引言

在分析代码之前，我们先来了解如何通过 Kubernetes API Server 暴露的 HTTP API 找到对应的资源信息。例如，如果您想通过 API 接口查询命名空间 keda 下的 Deployment：

> kubectl get --raw /apis/apps/v1/namespaces/keda/deployments

这里的关键概念是：
~~~
GV：group & version
GVK：group & version & kind
GVR：group & version & resource
~~~

资源（resource）总是属于某个 API 组（group）的某个版本（version），称为 GVR。
同样，类型（kind）也属于某个 API 组的某个版本，称为 GVK。
Kubernetes 通过 GVK 来确定资源类型，通过 GVR 来定位具体的资源信息，因此 GVK/GVR 可以视为 Kubernetes 资源的坐标。

简单来说，在编写 YAML 文件时，我们会指定 apiVersion 和 kind，这实际上就是 GVK；而与 Kubernetes API Server 通信是通过 HTTP 请求，这个请求的路径就是 GVR。

#### 如何查找 Kubernetes API Server 暴露的 HTTP API
要获取集群支持的所有 API 版本，可以使用以下命令，输出将以 group/version 的形式展示：
> kubectl api-versions

要获取特定 group/version 下的资源，可以使用：
> kubectl api-resources --api-group {GroupName} -o wide

## 代码结构
~~~
├── ....
├── cmd // 入口
    ├── adapter   // keda-operator-metrics-apiserver
    ├── operator  // keda-operator
    ├── webhooks
├── pkg   // 组件代码实现
├── ....
~~~

## 解析
我们将从获取指标的 API 为入口，了解 KEDA 是如何提供指标的。

kubectl 可以通过以下 API 获取对应的指标：

~~~
/apis/external.metrics.k8s.io/v1beta1/namespaces/{namespaces}/{externalMetricNames}?labelSelector=scaledobject.keda.sh/name={scaledObjectName}
~~~

同样，HPA（Horizontal Pod Autoscaler）也是通过这个路径来获取指标的。
通过查看 keda-operator-metrics-apiserver 的日志，我们可以通过 userAgent 字段区分出来自 kubectl 和 HPA 的请求。

~~~log
"HTTP" verb="LIST" URI="/apis/external.metrics.k8s.io/v1beta1/namespaces/default/s0-redis-mylist?labelSelector=scaledobject.keda.sh%2Fname%3Dtest-so-3" latency="7.408728ms" userAgent="kubectl/v1.28.5+k3s1 (linux/amd64) kubernetes/5b2d127" audit-ID="d6711361-45ea-4ea2-9bc5-2b417b424d44" srcIP="10.42.0.1:46152" resp=200
"HTTP" verb="LIST" URI="/apis/external.metrics.k8s.io/v1beta1/namespaces/default/s0-redis-mylist?labelSelector=scaledobject.keda.sh%2Fname%3Dtest-so-3" latency="12.145371ms" userAgent="k3s/v1.28.5+k3s1 (linux/amd64) kubernetes/5b2d127/system:serviceaccount:kube-system:horizontal-pod-autoscaler" audit-ID="66f37fc6-3683-4d36-89c5-be8758cd154a" srcIP="10.42.0.1:46152" resp=200
~~~

这些 HTTP API 请求最终由 ScaleHandler 的 GetScaledObjectMetrics 方法处理。
因此，我们跳过 Kube Aggregator 转发的细节，直接关注 ScaleHandler 的 GetScaledObjectMetrics 方法。

该方法通过指定的命名空间和 ScaledObject 名称查找 ScalersCache。
ScalersCache 主要包含 ScaledObject 及其指定的 Scaler。
指标数据由实际指定的 Scaler 对象的 GetMetricsAndActivity 方法返回。

例如 External Scaler 的 GetMetricsAndActivity 方法，它会去调用我们声明的 GRPC 服务地址获取指标。
 
详细流程看下图:

## 流程图
![](../assets/images/keda/keda-operator-metrics-apiserver-sq.png)
