---
categories: [ Terraform ]
tags: [ k8s, terraform ]
title: Terraform Provider 离线安装
---

### 第一步

在有网络的环境中，在对应terraform 项目文件中执行命令：

> terraform providers mirror -platform=linux_amd64 .

- -platform=OS_ARCH: 指定需要运行的 providers 的系统架构，默认是本机架构。
- . : 指定下载后的 <target-dir>，即 providers 镜像将保存在该目录中。

运行命令后，项目的根目录会多出一个名为 registry.terraform.io 的文件夹，里面是已下载的镜像文件

### 第二步

1. 在没网络的系统中，进入用户根目录，例如 cd ~
2. 创建 .terraformrc
3. 配置示例

~~~
provider_installation {
    filesystem_mirror {
        path    = "/terraform-cache/"
        include = ["registry.terraform.io/bangau1/*"]
    }
}
~~~

- path : 下载的镜像文件所在文件夹
- include ： 要被加载的provider 名称
