---
title: 从 Pod 中访问 Kubernetes API
categories: [ Kubernetes ]
tags: k8s
---

在 Kubernetes 集群中，当一个 Pod 需要与 API Server 通信时，它通常会以特定的 ServiceAccount 身份进行认证。每个 Kubernetes
命名空间至少会有一个 ServiceAccount，通常被称为 default。如果在创建 Pod 时没有特别指定 ServiceAccount，Kubernetes 会自动为该
Pod 分配所在命名空间的 default ServiceAccount。

ServiceAccount 的认证信息，包括令牌，会被存储在 Pod 内每个容器的文件系统中的特定路径下。具体来说，令牌位于
/var/run/secrets/kubernetes.io/serviceaccount/token；如果提供了证书包，它将被放置在
/var/run/secrets/kubernetes.io/serviceaccount/ca.crt，用于验证 API 服务器的服务证书。此外，Pod 的命名空间信息会存储在
/var/run/secrets/kubernetes.io/serviceaccount/namespace 文件中。

### 使用 kubectl 访问 API

kubectl 是 Kubernetes 的命令行工具，它能够确定是否在 Pod 内运行。当 kubectl 在 Pod 内运行时，它会检查几个关键的环境变量和文件：

- KUBERNETES_SERVICE_HOST 和 KUBERNETES_SERVICE_PORT 环境变量
- 服务账户令牌文件 /var/run/secrets/kubernetes.io/serviceaccount/token

如果这些条件都满足，kubectl 会假定它正在集群内部运行，并使用 ServiceAccount 的命名空间（与 Pod 的命名空间相同）来执行操作。

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

### 使用 Java 客户端访问 API

Java 客户端库提供了一种与 Kubernetes API Server 建立连接的方法。它首先检查环境变量 $KUBECONFIG
来确定配置文件的位置，然后尝试在用户的主目录中查找 ~/.kube/config。如果这些都不成功，它会尝试使用集群内的
ServiceAccount。如果所有方法都失败，它会回退到使用 localhost:8080 作为默认值。

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
