---
categories: [Kubernetes, KEDA]
tags: keda
---

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

KEDA 项目是用 kubebuilder SDK来完成 ScaledObject 的 Operator 编写。

组件启动入口在于main.go （cmd\operator\main.go）文件中： 通过 controller-runtime 组件启动 ScaledObjectReconciler。 

ScaledObjectReconciler 是由 kubebuilder 生成的代码， 所以我们主要关注 ScaledObjectReconciler（controllers\keda\scaledobject_controller.go） 的 Reconcile 方法。

我们可以带着 `Scaled Object 配置项的作用是什么？` 这个问题来阅读这个方法的代码，方便我们理解方法执行流程，下列配置说明可以结合时序图理解。

1. ScaledObject 对象被删除的时。 当 `ScaledObject.spec.advanced.restoreToOriginalReplicaCount = true` 的时候，`scaleTargetRef.Spec.Replicas` 会被设置为 Replicas 子对象声明的 replicas 数量。

2. ScaledObject 对象有 `autoscaling.keda.sh/paused: "true"` 这个 annotations 的时候，会暂停自动缩放。 scaledobject 处于暂停状态，
如果当前的 replicas 数量不等于 `autoscaling.keda.sh/paused-replicas` 数，则触发缩放逻辑，让当前副本书等于暂停副本数。

3. ensureHPAForScaledObjectExists 方法按配置的 ScaledObject 对象创建HPA资源。
~~~go
	hpa := &autoscalingv2.HorizontalPodAutoscaler{
		Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
		    // 如果 ScaledObject 对象有 paused 的 annotation，
		    // 则 minReplicas 和 maxReplicas 就会等于 paused-replicas。
			MinReplicas: minReplicas,
			MaxReplicas: maxReplicas,
			// getScaledObjectMetricSpecs 方法会获取 triggers 中的 metrics 指标。
			// 获取的时候会区分External metrics和Resource metrics，
			// 因为 CPU/Memory scaler 是通过 resource metrics 来获取的。
			Metrics:     scaledObjectMetricSpecs,
			Behavior:    behavior,
			ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
				Name:       scaledObject.Spec.ScaleTargetRef.Name,
				Kind:       gvkr.Kind,
				APIVersion: gvkr.GroupVersion().String(),
			}},
		ObjectMeta: metav1.ObjectMeta{
			Name:        getHPAName(scaledObject),
			Namespace:   scaledObject.Namespace,
			Labels:      labels,
			Annotations: scaledObject.Annotations,
		},
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v2",
		},
	}
~~~

4. requestScaleLoop 方法用来循环检查 Scaler 中的 IsActive 状态并作出对应的处理，详细流程见下面 Active/InActive 活动图。
- isActive = true
> currentReplicas < minReplicas
> 
>: : updateScaleOnScaleTarget -> max(minReplicaCount, 1)
> 
> currentReplicas == 0
> 
>: : updateScaleOnScaleTarget -> max(minReplicaCount, 1)
> 
> isError = true 
> 
>: : ScaledObject.Status.ReadyCondition -> Unknown
- isActive = false
  - isError = true
> fallback.replicas != 0 
> 
>: : updateScaleOnScaleTarget -> fallback.replicas
> 
> else ：
> 
>: ：ScaledObject.Status.ReadyCondition -> False
  - isError = false
> idleReplicaCount != nil && currentReplicas > idleReplicaCount && ScaledObject.LastActiveTime.Add(cooldownPeriod).Before(now)
> 
>: : updateScaleOnScaleTarget -> idleReplicaCount
> 
> currentReplicas > 0 && minReplicas == 0 && ScaledObject.LastActiveTime.Add(cooldownPeriod).Before(now)
> 
>: : updateScaleOnScaleTarget -> 0
> 
> currentReplicas < minReplicaCount && idleReplicaCount == nil
> 
>: : updateScaleOnScaleTarget -> minReplicaCount

总结：IsActive 为 true 的时候，如果 ScaleTarget Replicas 为 0，至少将它扩容为 1。
为 false 的时候，有 error 时把 Replicas 扩缩容至 fallback.replicas 。 
没 error 时，优先使用 idleReplicaCount ，其次再用 minReplicas 进行缩容。这个缩容的过程受上次 Active 时间至 cooldownPeriod 等待时间的影响。

## 时序图
![](../assets/images/keda/keda-operator-sq.png)

## Active 活动图
![](../assets/images/keda/keda-operator-active-ad.png)

## InActive 活动图
![](../assets/images/keda/keda-operator-inactive-ad.png)
