# DirectPV Admin Client APIs

The DirectPV Admin Golang Client SDK provides APIs to manage DirectPV drives and volumes.

This quickstart guide will show you how to use DirectPV Admin client SDK to list the initialized drives.

```go
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/minio/directpv/pkg/admin"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	MaxThreadCount = 200
)

func getKubeConfig() (*rest.Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	kubeConfig := filepath.Join(home, ".kube", "config")
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfig)
	if err != nil {
		if config, err = rest.InClusterConfig(); err != nil {
			return nil, err
		}
	}
	config.QPS = float32(MaxThreadCount / 2)
	config.Burst = MaxThreadCount
	return config, nil
}

func main() {
	kubeConfig, err := getKubeConfig()
	if err != nil {
		fmt.Printf("%s: Could not connect to kubernetes. %s=%s\n", "Error", "KUBECONFIG", kubeConfig)
		os.Exit(1)
	}
	adminClient, err := admin.NewClient(kubeConfig)
	if err != nil {
		log.Fatalf("unable to initialize client; %v", err)
	}
	drives, err := adminClient.NewDriveLister().Get(context.Background())
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	for _, drive := range drives {
		fmt.Printf("\n DeviceName: %v", drive.GetDriveName())
		fmt.Printf("\n Node: %v", drive.GetNodeID())
		fmt.Printf("\n Make: %v", drive.Status.Make)
		fmt.Println()
	}
}
```

## Install DirectPV

Installs DirectPV components

### Install(ctx context.Context, args InstallArgs, installerTasks []installer.Task) error

__Example__

```go
args := admin.InstallArgs{
	Image:            image,
	Registry:         registry,
	Org:              org,
	ImagePullSecrets: imagePullSecrets,
	NodeSelector:     nodeSelector,
	Tolerations:      tolerations,
	SeccompProfile:   seccompProfile,
	AppArmorProfile:  apparmorProfile,
	EnableLegacy:     legacyFlag,
	PluginVersion:    pluginVersion,
	Quiet:            quietFlag,
	KubeVersion:      kubeVersion,
	DryRun:           dryRunPrinter != nil,
	OutputFormat:     outputFormat,
	Declarative:      declarativeFlag,
	Openshift:        openshiftFlag,
	AuditWriter:      file,
}
var installedComponents []installer.Component
legacyClient, err := legacyclient.NewClient(adminClient.K8s())
if err != nil {
	log.Fatalf("error creating legacy client:", err)
}
installerTasks := installer.GetDefaultTasks(adminClient.Client, legacyClient)
if err := adminClient.Install(ctx, args, installerTasks); err != nil {
	log.Fatalf("unable to complete installation; %v", err)
}
```

## Refresh Nodes

Refreshes the nodes to get the latest devices list, this is used for discovering the nodes and devices present in the cluster.

### RefreshNodes(ctx context.Context, selectedNodes []string) (<-chan directpvtypes.NodeID, <-chan error, error)

__Example__

```go
nodeCh, errCh, err := adminClient.RefreshNodes(ctx, []string{"praveen-thinkpad-x1-carbon-6th"})
if err != nil {
	log.Fatalln(err)
}

for {
	select {
		case nodeID, ok := <-nodeCh:
			if !ok {
				return
			}
			log.Println("Refreshing node ", nodeID)
		case err, ok := <-errCh:
			if !ok {
				return
			}
			log.Fatalln(err)
		case <-ctx.Done():
			return
	}
}
```

## DirectPV Info

Returns the overall information about DirectPV installation

### Info(ctx context.Context) (map[string]NodeLevelInfo, error)

__Example__

```go
nodeInfoMap, err := adminClient.Info(context.Background())
if err != nil {
	log.Fatalf("unable to get info; %v", err)
}
```

## Label DirectPV drives

Label the directpv drives

### LabelDrives(ctx context.Context, args LabelDriveArgs, labels []Label, log logFn) (results []LabelDriveResult, err error)

__Example__

```go
labels := []admin.Label{
	{
		Key:   directpvtypes.LabelKey("example-key"),
		Value: directpvtypes.LabelValue("example-value"),
	},
}
if _, err := adminClient.LabelDrives(context.Background(), admin.LabelDriveArgs{
	Nodes:  []string{"praveen-thinkpad-x1-carbon-6th"},
	Drives: []string{"dm-0"},
}, labels, log); err != nil {
	log.Fatalf("unable to label the drive; %v", err)
}
fmt.Println("successfully labeled the drive(s)")
```

## Label DirectPV volumes

Label the directpv volumes

### LabelVolumes(ctx context.Context, args LabelVolumeArgs, labels []Label, log logFn) (results []LabelVolumeResult, err error)

__Example__

```go
labels := []admin.Label{
	{
		Key:   directpvtypes.LabelKey("example-key"),
		Value: directpvtypes.LabelValue("example-value"),
	},
}
if _, err := adminClient.LabelVolumes(context.Background(), admin.LabelVolumeArgs{
	Nodes:  []string{"praveen-thinkpad-x1-carbon-6th"},
	Drives: []string{"dm-0"},
}, labels, log); err != nil {
	log.Fatalf("unable to label the volume; %v", err)
}
```

## Cordon the drive

Cordon the drive to make it unschedulable

### Cordon(ctx context.Context, args CordonArgs, log logFn) (results []CordonResult, err error)

__Example__

```go
if _, err := adminClient.Cordon(context.Background(), admin.CordonArgs{
	Drives: []string{"dm-1"},
}, log); err != nil {
	log.Fatalf("unable to cordon the drive; %v", err)
}
```

## Uncordon the drive

Mark drives as schedulable

### Uncordon(ctx context.Context, args UncordonArgs, log logFn) (results []UncordonResult, err error)

__Example__

```go
if _, err := adminClient.Uncordon(context.Background(), admin.UncordonArgs{
	Drives: []string{"dm-1"},
}, log); err != nil {
	log.Fatalf("unable to uncordon the drive; %v", err)
}
```

## Migrate the legacy drives and volumes

Migrates the legacy direct-csi drives and volumes

### Migrate(ctx context.Context, args MigrateArgs) error

__Example__

```go
suffix := time.Now().Format(time.RFC3339)
if err := adminClient.Migrate(ctx, admin.MigrateArgs{
	DrivesBackupFile:  "directcsidrives-" + suffix + ".yaml",
	VolumesBackupFile: "directcsivolumes-" + suffix + ".yaml",
}); err != nil {
	log.Fatalf("migration failed; %v", err)
}
```

## Move the volume references from one drive to another

Move volumes excluding data from source drive to destination drive on a same node

### Move(ctx context.Context, args MoveArgs, log logFn) error

__Example__

```go
if err := adminClient.Move(context.Background(), admin.MoveArgs{
	Source:      directpvtypes.DriveID("2786de98-2a84-40d4-8cee-8f73686928f8"),
	Destination: directpvtypes.DriveID("b35f1f8e-6bf3-4747-9976-192b23c1a019"),
}, log); err != nil {
	log.Fatalf("unable to move the drive; %v", err)
}
fmt.Println("successfully moved the drive")
```

## Cleanup volumes

Cleanup stale volumes

### Clean(ctx context.Context, args CleanArgs, log logFn) (removedVolumes []string, err error)

__Example__

```go
if _, err := adminClient.Clean(context.Background(), admin.CleanArgs{
	Nodes:  []string{"praveen-thinkpad-x1-carbon-6th"},
	Drives: []string{"dm-0"},
}, log); err != nil {
	log.Fatalf("unable to clean the volume; %v", err)
}
fmt.Println("successfully cleaned the volume(s)")
```

## Suspend drives

Suspend the drives (CAUTION: This will make the corresponding volumes as read-only)

### SuspendDrives(ctx context.Context, args SuspendDriveArgs, log logFn) (results []SuspendDriveResult, err error)

__Example__

```go
if _, err := adminClient.SuspendDrives(context.Background(), admin.SuspendDriveArgs{
	Nodes:  []string{"praveen-thinkpad-x1-carbon-6th"},
	Drives: []string{"dm-0"},
}, log); err != nil {
	log.Fatalf("unable to suspend the drive; %v", err)
}
fmt.Println("successfully suspended the drive(s)")
```

## Suspend volumes

Suspend the volumes (CAUTION: This will make the corresponding volumes as read-only)

### SuspendVolumes(ctx context.Context, args SuspendVolumeArgs, log logFn) (results []SuspendVolumeResult, err error)

__Example__

```go
if _, err := adminClient.SuspendVolumes(context.Background(), admin.SuspendVolumeArgs{
	Nodes:  []string{"praveen-thinkpad-x1-carbon-6th"},
	Drives: []string{"dm-0"},
}, log); err != nil {
	log.Fatalf("unable to suspend the volume; %v", err)
}
fmt.Println("successfully suspended the volume(s)")
```

## Resume drives

Resume suspended drives

### ResumeDrives(ctx context.Context, args ResumeDriveArgs, log logFn) (results []ResumeDriveResult, err error)

__Example__

```go
if _, err := adminClient.ResumeDrives(context.Background(), admin.SuspendDriveArgs{
	Nodes:  []string{"praveen-thinkpad-x1-carbon-6th"},
	Drives: []string{"dm-0"},
}, log); err != nil {
	log.Fatalf("unable to resume the drive; %v", err)
}
fmt.Println("successfully resumed the drive(s)")
```

## Resume volumes

Resume suspended volumes

### ResumeVolumes(ctx context.Context, args ResumeVolumeArgs, log logFn) (results []ResumeVolumeResult, err error)

__Example__

```go
if _, err := adminClient.ResumeVolumes(context.Background(), admin.ResumeVolumeArgs{
	Nodes:  []string{"praveen-thinkpad-x1-carbon-6th"},
	Drives: []string{"dm-0"},
}, log); err != nil {
	log.Fatalf("unable to resume the volume; %v", err)
}
fmt.Println("successfully resumed the volume(s)")
```

## Remove drives

Remove unused drives from DirectPV

### Remove(ctx context.Context, args RemoveArgs, log logFn) (results []RemoveResult, err error)

__Example__

```go
if _, err := adminClient.Remove(context.Background(), admin.RemoveArgs{
	Nodes:  []string{"praveen-thinkpad-x1-carbon-6th"},
	Drives: []string{"dm-0"},
}, log); err != nil {
	log.Fatalf("unable to remove the drive; %v", err)
}
fmt.Println("successfully removed the drive(s)")
```
