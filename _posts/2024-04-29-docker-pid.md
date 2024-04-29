---
title: Docker容器中的PID 1解析
categories: [ Kubernetes ]
tags: [ k8s, docker ]
---

## 容器中的PID 1

在Docker容器中，PID 1通常指的是容器内的第一个进程，它实际上是一个普通的用户进程，由我们在构建Docker镜像时在Dockerfile中指定的ENTRYPOINT所启动。
这个进程在宿主机上拥有一个普通的进程ID，但在容器内部，它被赋予了PID 1的特殊身份，这得益于Linux内核的PID命名空间功能。

#### 容器外对PID 1进程的影响

值得注意的是，如果在容器外部通过kill -9命令强制终止PID 1进程，整个容器将会随之退出。

Dockerfile ：

~~~dockerfile
FROM openjdk:17-jdk-alpine
RUN addgroup --system app && adduser --system --no-create-home --ingroup app app
USER app
WORKDIR /app
COPY run.sh ./
COPY dependencies/ ./
COPY spring-boot-loader/ ./
COPY snapshot-dependencies/ ./
COPY application/ ./
ENTRYPOINT ["sh","./run.sh"]
~~~

容器进程：

~~~
/ # ps -ef
PID   USER     TIME  COMMAND
1     root     3:22  java org.springframework.boot.loader.launch.JarLauncher
~~~

宿主机进程：

~~~
/ # ps -ef
UID     PID      USER     C   TIME             COMMAND
root    265108   265055   0   Apr26 00:03:22   java org.springframework.boot.loader.launch.JarLauncher
~~~

## Shell脚本执行与exec命令的区别

执行脚本时，如果我们使用sh script.sh，当前的shell会生成一个子shell来运行脚本。一旦脚本执行完毕，子shell会退出，控制权重新交回给父shell。

而当我们采用exec命令执行时，它会用指定的命令替换当前的shell进程，同时保持进程ID（PID）不变。这意味着，一旦exec执行的命令结束，它不会返回到之前的shell环境，而是直接退出。



## ENTRYPOINT与CMD指令的对比

Dockerfile中的ENTRYPOINT和CMD指令都用于定义容器的启动命令，但它们之间存在一些关键的差异：

- `ENTRYPOINT ["java", "-jar", "spring.jar"]` 直接启动Java进程，该进程将作为PID 1运行。
- `CMD ["java", "-jar", "spring.jar"]` 作为参数传递给ENTRYPOINT，如果ENTRYPOINT以shell脚本形式定义，CMD中的命令不会成为PID 1。
- `ENTRYPOINT ["sh", "./run.sh"]` 通过shell脚本启动Java应用。如果脚本使用exec，则Java进程将成为PID 1；否则，shell脚本的进程会是PID 1。

例：
~~~shell
#!/bin/sh
exec java -jar spring.jar
~~~

~~~shell
#!/bin/sh
java -jar spring.jar
~~~


