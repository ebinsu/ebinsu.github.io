@startuml
collections "HPA / Kubectl" as api

box "cmd\adapter\main.go"
participant main
participant Adapter as adapter
end box

participant KedaProvider as provider

box "pkg\metricsservice"
participant client.go as client
participant server.go as server
end box

participant ScaleHandler as handler

participant ScalersCaches as sc

participant Scaler as scaler

main -> main ++ : main
main -> main : new Adapter
main -> adapter ++ : makeProvider
adapter -> client ++ : NewGrpcClient
return GrpcClient
adapter -> provider ++ : NewProvider
provider -> main -- : KedaProvider
adapter  --
main -> adapter : WithExternalMetrics
main -> main : RunMetricsServer
note left : runs a http listener and handles the /metrics endpoint
main -> adapter ++ : Run
adapter -> adapter ++ : Server
note left : registers the api server in Kube Aggregator
return
return
return

api -> provider ++ : /apis/external.metrics.k8s.io/v1beta1
provider -> provider ++ : GetExternalMetric
provider -> client ++ : GetMetrics
client -> server ++ : GetMetrics
server -> handler ++ : GetScaledObjectMetrics
handler -> handler : getScalersCacheForScaledObject
handler -> sc ++ : GetMetricsAndActivityForScaler
sc -> scaler ++ : GetMetricsAndActivity
return external_metrics.ExternalMetricValue
return
return
return
return 
return
return external_metrics.ExternalMetricValueList

@enduml