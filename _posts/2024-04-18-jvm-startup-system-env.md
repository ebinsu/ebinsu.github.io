---
title: Java 命令启动时从环境变量读取参数的方式
categories: [ JAVA ]
tags: java
---

## 区别

1. _JAVA_OPTIONS
> JDK厂商的扩展环境变量， 比如，Oracle/OpenJDK 使用 *_JAVA_OPTIONS* ，IBM会读取 *IBM_JAVA_OPTIONS* ，OpenJ9 使用 *OPENJ9_JAVA_OPTIONS*。

2. JDK_JAVA_OPTIONS
>JDK 9+ 提供的变量，它只会支持影响到 java启动命令；同时还支持了通过 export JDK_JAVA_OPTIONS='@file' 的方式从指定的文件读取的相关的内容。

3. JAVA_TOOL_OPTIONS
> 有些应用不方便设置JVM参数，如命令行应用等场景。JDK 提供了该系统环境变量，设置此环境变量后，JNI_CreateJavaVM函数（在 JNI 调用 API 中）会将环境变量的值添加到其JavaVMInitArgs参数中提供的选项中。
> 
> 所以这个参数在一些场景存在问题：
- 不仅针对 java 命令生效：其他的管控命令如：jar, jstack, jmap等也一样会生效。而容器内的进程默认都会读取外部传入的环境变量的值，即一旦设置，这个值会被容器内所有的进程共享，意味着当我们想进入到容器进行一些 java 程序的排查工作时，默认都会受到 JAVA_TOOL_OPTIONS 这个变量的影响而得不到预期的结果。

4. JAVA_OPTS:
>不是JDK提供的，而是由的启动 java 的脚本使用，约定俗成使用这个名字 （例如，tomcat 启动脚本）。
>
>启动 java 的脚本示例：
~~~sh
#!/bin/sh
exec java $JAVA_OPTS org.springframework.boot.loader.JarLauncher $@
~~~

## 优先级
### 结论
> _JAVA_OPTIONS > CMD > JDK_JAVA_OPTIONS > JAVA_TOOL_OPTIONS

### 实验
测试类:
~~~java
public class Test9 {

    public static void main(String[] args) {
        System.out.println("arg: " + System.getProperty("arg"));
    }
}
~~~
场景：

1. _JAVA_OPTIONS & JDK_JAVA_OPTIONS & JAVA_TOOL_OPTIONS & CMD
~~~shell
docker run --rm \
-e _JAVA_OPTIONS=-Darg=_JAVA_OPTIONS \
-e JDK_JAVA_OPTIONS=-Darg=JDK_JAVA_OPTIONS \
-e JAVA_TOOL_OPTIONS=-Darg=JAVA_TOOL_OPTIONS  \
-v /workspaces/otel:/usr/src/myapp -w /usr/src/myapp openjdk:23-slim \
java -Darg=CMD Test9
~~~
~~~
NOTE: Picked up JDK_JAVA_OPTIONS: -Darg=JDK_JAVA_OPTIONS
Picked up JAVA_TOOL_OPTIONS: -Darg=JAVA_TOOL_OPTIONS
Picked up _JAVA_OPTIONS: -Darg=_JAVA_OPTIONS
arg: _JAVA_OPTIONS
~~~

2. JDK_JAVA_OPTIONS & JAVA_TOOL_OPTIONS & CMD
~~~shell
docker run --rm \
-e JDK_JAVA_OPTIONS=-Darg=JDK_JAVA_OPTIONS \
-e JAVA_TOOL_OPTIONS=-Darg=JAVA_TOOL_OPTIONS  \
-v /workspaces/otel:/usr/src/myapp -w /usr/src/myapp openjdk:23-slim \
java -Darg=CMD Test9
~~~
~~~
NOTE: Picked up JDK_JAVA_OPTIONS: -Darg=JDK_JAVA_OPTIONS
Picked up JAVA_TOOL_OPTIONS: -Darg=JAVA_TOOL_OPTIONS
arg: CMD
~~~

3. JDK_JAVA_OPTIONS & JAVA_TOOL_OPTIONS
~~~shell
docker run --rm \
-e JDK_JAVA_OPTIONS=-Darg=JDK_JAVA_OPTIONS \
-e JAVA_TOOL_OPTIONS=-Darg=JAVA_TOOL_OPTIONS  \
-v /workspaces/otel:/usr/src/myapp -w /usr/src/myapp openjdk:23-slim \
java Test9
~~~
~~~
NOTE: Picked up JDK_JAVA_OPTIONS: -Darg=JDK_JAVA_OPTIONS
Picked up JAVA_TOOL_OPTIONS: -Darg=JAVA_TOOL_OPTIONS
arg: JDK_JAVA_OPTIONS
~~~
