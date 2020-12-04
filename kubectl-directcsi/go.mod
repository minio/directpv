module github.com/minio/kubectl-directcsi

go 1.14

require (
	github.com/dustin/go-humanize v1.0.0
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/jedib0t/go-pretty v4.3.0+incompatible
	github.com/mattn/go-runewidth v0.0.9 // indirect
	github.com/minio/direct-csi v0.2.1
	github.com/minio/minio v0.0.0-20200622032605-a521907ab497
	github.com/minio/minio-go/v6 v6.0.58-0.20200612001654-a57fec8037ec
	github.com/rakyll/statik v0.1.7
	github.com/spf13/cobra v1.1.1
	gopkg.in/yaml.v2 v2.3.0 // indirect
	k8s.io/api v0.19.3
	k8s.io/apiextensions-apiserver v0.19.3
	k8s.io/apimachinery v0.19.3
	k8s.io/cli-runtime v0.19.3
	k8s.io/client-go v0.19.3
)
