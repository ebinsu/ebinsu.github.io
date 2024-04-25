---
title: Java 命令启动时从环境变量读取参数的方式
categories: [ JAVA ]
tags: java
---

## 引言

环境变量的区别
在启动 Java 应用程序时，不同的 JDK 厂商会使用不同的环境变量来读取 JVM 参数：

- **_JAVA_OPTIONS** : 通常由 Oracle 和 OpenJDK 使用。
- **IBM_JAVA_OPTIONS** : IBM 的 JDK 会读取此环境变量。
- **OPENJ9_JAVA_OPTIONS** : OpenJ9 JDK 使用此环境变量。

## 区别

- _JAVA_OPTIONS

> JDK厂商的扩展环境变量， 比如，Oracle/OpenJDK 使用 *_JAVA_OPTIONS* ，IBM会读取 *IBM_JAVA_OPTIONS* ，OpenJ9 使用
*OPENJ9_JAVA_OPTIONS*。

- JDK_JAVA_OPTIONS

> 这是 JDK 9 及以上版本提供的变量，它主要用于影响 java 启动命令。此外，你还可以通过 export JDK_JAVA_OPTIONS='@file'
> 的方式从指定文件中读取参数。

- JAVA_TOOL_OPTIONS

> 对于一些不便设置 JVM 参数的应用，如命令行工具等，JDK 提供了此系统环境变量。设置此变量后，JNI_CreateJavaVM 函数（在 JNI 调用
> API 中）会将其值添加到 JavaVMInitArgs 参数中。但请注意，这个参数不仅对 java 命令有效，还会影响其他命令如 jar, jstack,
> jmap 等。在容器环境中，所有进程默认都会读取外部传入的环境变量，这意味着一旦设置，该值会被容器内所有进程共享，可能会影响排查工作。

- JAVA_OPTS:

> 这不是 JDK 提供的官方环境变量，而是启动 Java 应用程序时脚本约定俗成使用的变量名，如 Tomcat 启动脚本。
>
>启动 java 的脚本示例：

~~~sh
#!/bin/sh
exec java $JAVA_OPTS org.springframework.boot.loader.JarLauncher $@
~~~

## 优先级

### 结论

> _JAVA_OPTIONS > CMD > JDK_JAVA_OPTIONS > JAVA_TOOL_OPTIONS

_JAVA_OPTIONS & JDK_JAVA_OPTIONS & JAVA_TOOL_OPTIONS & CMD

实验结果显示 _JAVA_OPTIONS 的参数被优先采纳。

JDK_JAVA_OPTIONS & JAVA_TOOL_OPTIONS & CMD

当没有设置 _JAVA_OPTIONS 时，JDK_JAVA_OPTIONS 的参数被采纳。

JDK_JAVA_OPTIONS & JAVA_TOOL_OPTIONS

如果没有设置 _JAVA_OPTIONS 和 JDK_JAVA_OPTIONS，那么 JAVA_TOOL_OPTIONS 的参数将被使用。

### 实验代码

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
