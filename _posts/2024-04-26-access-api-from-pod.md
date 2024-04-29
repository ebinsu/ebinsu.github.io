---
title: 从 Pod 中访问 Kubernetes API
categories: [ Kubernetes ]
tags: k8s
---

当 Pod 访问 Kubernetes API Server 时，Pod 会被认证为某个特定的 ServiceAccount（例如：default）。 在每个名字空间中，至少存在一个
ServiceAccount。

每个 Kubernetes 名字空间至少包含一个 ServiceAccount：也就是该名字空间的默认 ServiceAccount 名为 default。如果你在创建 Pod
时没有指定
ServiceAccount，Kubernetes 会自动将该名字空间中名为 default 的 ServiceAccount 分配给该 Pod。

该 ServiceAccount 的凭据（令牌）放置在此 Pod 中每个容器的文件系统树中的
/var/run/secrets/kubernetes.io/serviceaccount/token 处。

如果证书包可用，则凭据包被放入每个容器的文件系统树中的 /var/run/secrets/kubernetes.io/serviceaccount/ca.crt 处， 且将被用于验证
API 服务器的服务证书。

最后，用于命名空间域 API 操作的默认命名空间放置在每个容器中的 /var/run/secrets/kubernetes.io/serviceaccount/namespace
文件中。

通过工具从 Pod 中访问 API：

### kubectl

kubectl 命令首先确定它是否在 Pod 中运行，从而被视为在集群中运行。 它首先检查 `KUBERNETES_SERVICE_HOST` 和
`KUBERNETES_SERVICE_PORT` 环境变量以及 `/var/run/secrets/kubernetes.io/serviceaccount/token` 中是否存在服务帐户令牌文件。
如果三个条件都被满足，则假定在集群内进行身份验证。

kubectl 如何处理 ServiceAccount 令牌

假设：

- 有 Kubernetes 服务帐户令牌文件挂载在 /var/run/secrets/kubernetes.io/serviceaccount/token 上，并且
- 设置了 KUBERNETES_SERVICE_HOST 环境变量，并且
- 设置了 KUBERNETES_SERVICE_PORT 环境变量，并且
- 你没有在 kubectl 命令行上明确指定命名空间。

然后 kubectl 假定它正在你的集群中运行。 kubectl 工具查找该 ServiceAccount 的命名空间 （该命名空间与 Pod
的命名空间相同）并针对该命名空间进行操作。

config.go

~~~go
func InClusterConfig() (*Config, error) {
	const (
		tokenFile  = "/var/run/secrets/kubernetes.io/serviceaccount/token"
		rootCAFile = "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"
	)
	host, port := os.Getenv("KUBERNETES_SERVICE_HOST"), os.Getenv("KUBERNETES_SERVICE_PORT")
	if len(host) == 0 || len(port) == 0 {
		return nil, ErrNotInCluster
	}

	token, err := os.ReadFile(tokenFile)
	if err != nil {
		return nil, err
	}

	tlsClientConfig := TLSClientConfig{}

	if _, err := certutil.NewPool(rootCAFile); err != nil {
		klog.Errorf("Expected to load root CA config from %s, but got err: %v", rootCAFile, err)
	} else {
		tlsClientConfig.CAFile = rootCAFile
	}

	return &Config{
		// TODO: switch to using cluster DNS.
		Host:            "https://" + net.JoinHostPort(host, port),
		TLSClientConfig: tlsClientConfig,
		BearerToken:     string(token),
		BearerTokenFile: tokenFile,
	}, nil
}
~~~

### JAVA Client

JAVA Client 通过以下方式与 Kubernetes API Server 建立连接。

- 如果定义了$KUBECONFIG，则使用该配置文件。
- 如果$ HOME /。可以找到Kube /config，使用它。
- 如果可以找到集群内的服务帐户，则假设在集群配置中。
- 默认为localhost:8080作为最后的手段。

~~~java
public class ClientBuilder {

  public static ClientBuilder standard(boolean persistConfig) throws IOException {
    final File kubeConfig = findConfigFromEnv();
    ClientBuilder clientBuilderEnv = getClientBuilder(persistConfig, kubeConfig);
    if (clientBuilderEnv != null) return clientBuilderEnv;
    final File config = findConfigInHomeDir();
    ClientBuilder clientBuilderHomeDir = getClientBuilder(persistConfig, config);
    if (clientBuilderHomeDir != null) return clientBuilderHomeDir;
    final File clusterCa = new File(SERVICEACCOUNT_CA_PATH);
    if (clusterCa.exists()) {
      return cluster();
    }
    return new ClientBuilder();
  }

}
~~~
