---
title: Kube-builder
categories: [ Kubernetes ]
tags: [ k8s ]
---

# 引言

kubebuilder是一个官方提供快速实现 Operator 的工具包，可快速生成 k8s 的 CRD、Controller、Webhook，用户只需要实现业务逻辑。

一般创建 Operator 流程如下：

### 安装 kubebuilder 
~~~
curl -L -o kubebuilder "https://go.kubebuilder.io/dl/latest/$(go env GOOS)/$(go env GOARCH)"
chmod +x kubebuilder && mv kubebuilder /usr/local/bin/
~~~

[文档](https://book.kubebuilder.io/quick-start.html#installation)

### 初始化 go 项目
~~~
go mod init ebinsu.cn/m
~~~
### 初始化 kubebuilder 项目结构
~~~
kubebuilder init --domain=ebinsu.cn // 最多二级域名
~~~
### 生成 API 代码模板
~~~
kubebuilder create api --group webapp --version v1 --kind WebApp
~~~
### 编写 API

在 `api\v1\webapplication_types.go` 定义我们需要的字段

~~~
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Name",type=string,JSONPath=`.spec.name`
// +kubebuilder:printcolumn:name="Image",type=string,JSONPath=`.spec.image`
// +kubebuilder:printcolumn:name="Replicas",type=integer,JSONPath=`.spec.replicas`
type WebApp struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WebAppSpec   `json:"spec,omitempty"`
	Status WebAppStatus `json:"status,omitempty"`
}

type WebAppSpec struct {
	Name     string `json:"name"`
	Image    string `json:"image"`
	Replicas *int32 `json:"replicas"`
}
~~~

> kubectl get 的显示列由+kubebuilder:printcolumn控制。

### 编写 Controller 逻辑

在 `internal\controller\webapp_controller.go` Reconcile 方法编写逻辑
~~~go
func (r *WebAppReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := log.FromContext(ctx)

  // 获取 WebApp 资源定义
	webapp := &webappv1.WebApp{}
	err := r.Client.Get(ctx, req.NamespacedName, webapp)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		reqLogger.Error(err, "failed to get WebApp")
		return ctrl.Result{}, err
	}

  // 添加 Finalizers 钩子，WebApp 删除时，同时删除关联的 deployment 和 service
	// 参考文档：https://book.kubebuilder.io/reference/using-finalizers
	if webapp.ObjectMeta.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(webapp, myFinalizerName) {
			controllerutil.AddFinalizer(webapp, myFinalizerName)
			if err := r.Update(ctx, webapp); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		if controllerutil.ContainsFinalizer(webapp, myFinalizerName) {
			r.deleteDeployAndService(ctx, req, *webapp)
			controllerutil.RemoveFinalizer(webapp, myFinalizerName)
			if err := r.Update(ctx, webapp); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

 // 创建或更新deployment
	r.reconcileDeployment(ctx, req, *webapp)
 // 创建或更新service
	r.reconcileService(ctx, req, *webapp)
	return ctrl.Result{}, nil
}
~~~

[完整代码]()

### 验证测试

#### 安装 CRD

> make install

#### 本地运行 operator

> make run

#### Apply Demo

~~~yaml
apiVersion: webapp.ebinsu.cn/v1
kind: WebApp
metadata:
  name: app-demo
spec:
  name: app-demo
  replicas: 1
  image: "ebinsu/spring-mvc-demo:1.0"
~~~

#### 验证结果

kubectl get deploy
~~~
NAME                                    READY   UP-TO-DATE   AVAILABLE   AGE
app-demo-deploy                         1/1     1            1           18s
~~~

kubectl get svc
~~~
NAME                                    TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)          AGE
app-demo                                NodePort    10.43.136.163   <none>        8080:32561/TCP   5s
~~~

### 编译部署

