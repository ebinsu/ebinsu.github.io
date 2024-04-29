---
categories: [ JAVA ]
tags: k8s, java
---

## shell，exec 执行脚本的区别

### sh

使用$ sh script.sh执行脚本时，当前shell是父进程，生成一个子shell进程，在子shell中执行脚本。脚本执行完毕，退出子shell，回到当前shell。

### exec

使用exec command方式，会用command进程替换当前shell进程，并且保持PID不变。执行完毕，直接退出，不回到之前的shell环境。

## 容器中的PID 1

在容器中被标志为PID 1的进程实际上就是一个普普通通的用户进程，也就是我们制作镜像时在Dockerfile中指定的ENTRYPOINT的那个进程。而这个进程在宿主机上有一个普普通通的进程ID，而在容器中之所以变成PID
1，是因为linux内核提供的[PID namespaces](https://lwn.net/Articles/531419/)。

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

如果在容器外部kiss -9 265108,那整个容器便会处于退出状态。

## 差别

- `ENTRYPOINT ["java","-jar","spring.jar"]`

- `CMD ["java","-jar","spring.jar"]`

- `ENTRYPOINT ["sh","./run.sh"]`

~~~shell
#!/bin/sh
exec java -jar spring-mvc-demo-0.0.1-SNAPSHOT.jar
~~~

java命令进程都是PID 1。

当 shell 脚本不是 exec command 方式执行时， PID 为 sh ./run.sh 命令的进程
~~~shell
#!/bin/sh
java -jar spring-mvc-demo-0.0.1-SNAPSHOT.jar
~~~

