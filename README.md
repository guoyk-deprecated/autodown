# autodown

automatically scale kubernetes workload to zero after a period of time

自动在一定时间后，将 Kubernetes 工作负载副本数减小到 0

## 动机

我司的基于 Kubernetes 的测试环境非常拥挤，很多人经常部署项目后就不再管了，而实际上当前正在测试的项目数量很，却占用了很多资源。

因此我突发奇想设计了这样一个工具，在一定时间后，自动将工作负载副本数减小到 0，如果有需要继续使用，直接重新调整回正常副本数就可以了。

## 使用方法

1. 部署 autodown 到 Kubernetes 集群

```yaml
// TODO: 尚未完成
```

2. 为需要自动调整为 0 的工作负载（deployment/statefulset 等) 添加注解

```
# 7d 代表 7 天
net.guoyk.autodown/lease=7d
```

## 许可证

Guo Y.K., MIT License
