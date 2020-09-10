# autodown

automatically scale kubernetes deployments to zero after a period of time

自动在一定时间后，将 Kubernetes Deployment 副本数减小到 0

## 动机

我司的基于 Kubernetes 的测试环境非常拥挤，很多人经常部署项目后就不再管了，而实际上当前正在测试的项目数量很，却占用了很多资源。

因此我突发奇想设计了这样一个工具，在一定时间后，自动将 Deployment 副本数减小到 0，如果有需要继续使用，直接重新调整回正常副本数就可以了。

## 使用方法

1. 创建命名空间 `autodown`

2. 部署 `autodown` 到 Kubernetes 集群

```yaml
# 在 autodown 命名空间创建专用的 ServiceAccount
apiVersion: v1
kind: ServiceAccount
metadata:
  name: autodown
  namespace: autodown
---
# 创建 ClusterRole autodown
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRole
metadata:
  name: autodown
rules:
  - apiGroups: [""]
    resources: ["namespaces"]
    verbs: ["list"]
  - apiGroups: ["apps"]
    resources: ["deployments"]
    verbs: ["list", "patch"]
---
# 创建 ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRoleBinding
metadata:
  name: autodown
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: autodown
subjects:
  - kind: ServiceAccount
    name: autodown
    namespace: autodown
---
# 创建 CronJob
apiVersion: batch/v1beta1
kind: CronJob
metadata:
  name: autodown
  namespace: autodown
spec:
  schedule: "5 2 * * *"
  jobTemplate:
    spec:
      template:
        spec:
          serviceAccount: autodown
          containers:
            - name: autodown
              image: guoyk/autodown
          restartPolicy: OnFailure
```

3. 为需要自动调整为 0 的 Deployment 添加注解

```
apiVersion: apps/v1
kind: Deployment
annotations:
    # 168h 等于 7 天
    net.guoyk.autodown/lease: 168h
# ....
```

## 测试

可以使用环境变量 `AUTODOWN_DRY_RUN=true` 来启动干跑模式，通过日志检查要执行的动作是否合理

## 许可证

Guo Y.K., MIT License
