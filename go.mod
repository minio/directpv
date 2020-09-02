module github.com/minio/direct-csi

go 1.14

require (
	github.com/container-storage-interface/spec v1.3.0
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/kubernetes-csi/csi-lib-utils v0.7.0 // indirect
	github.com/kubernetes-csi/drivers v1.0.2
	github.com/minio/minio v0.0.0-20200902071903-37da0c647e65
	github.com/pborman/uuid v1.2.1
	github.com/spf13/cobra v1.0.0
	github.com/spf13/viper v1.7.1
	google.golang.org/grpc v1.27.0
	k8s.io/api v0.19.0
	k8s.io/apiextensions-apiserver v0.19.0
	k8s.io/apimachinery v0.19.0
	k8s.io/client-go v0.19.0
	k8s.io/code-generator v0.19.0 // indirect
	k8s.io/component-base v0.19.0
	k8s.io/klog v1.0.0 // indirect
	k8s.io/utils v0.0.0-20200729134348-d5654de09c73
	github.com/go-logr/logr v0.2.1-0.20200730175230-ee2de8da5be6 // indirect
	sigs.k8s.io/controller-runtime v0.6.2
)
