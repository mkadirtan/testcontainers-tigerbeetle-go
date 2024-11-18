package tigerbeetle

import (
	"context"
	"fmt"
	"math/rand"
	"os"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	defaultPort     = "3000"
	clusterFileName = "0_0.tigerbeetle"
	clusterID       = 0
	replicaID       = 0
	replicaCount    = 1
)

const (
	DefaultImage = "ghcr.io/tigerbeetle/tigerbeetle:0.16.12"
)

type Container struct {
	testcontainers.Container
	clusterFileVolume string
}

// Address returns the connection address of the Tigerbeetle container
// The Cluster ID is 0
// Example usage:
// ```go
//
//	address, err := tbContainer.Address(ctx)
//	tbClient, err := tigerbeetle_go.NewClient(types.ToUint128(0), []string{address})
//
// ```
func (c *Container) Address(ctx context.Context) (string, error) {
	mappedPort, err := c.MappedPort(ctx, defaultPort)
	if err != nil {
		return "", err
	}

	return mappedPort.Port(), nil
}

// Run creates a temporary volume for 0_0.tigerbeetle cluster file and starts the Tigerbeetle at default 3000 port
func Run(ctx context.Context, img string, opts ...testcontainers.ContainerCustomizer) (*Container, error) {
	// tmpDir := os.TempDir()
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	tmpDir := wd
	suffix := fmt.Sprintf("tmp-tb-%d", rand.Uint64())
	var clusterFileDir string
	if len(tmpDir) > 0 && os.IsPathSeparator(tmpDir[len(tmpDir)-1]) {
		clusterFileDir = tmpDir + suffix
	} else {
		clusterFileDir = tmpDir + string(os.PathSeparator) + suffix
	}

	fmt.Printf("cluster file dir: %s\n", clusterFileDir)
	err = os.Mkdir(clusterFileDir, 0777)
	// clusterFileDir, err := os.MkdirTemp("", "")
	if err != nil {
		return nil, fmt.Errorf("could not create temporary directory for cluster file: %w", err)
	}

	stats, err := os.Stat(clusterFileDir)
	if err != nil {
		return nil, err
	}

	fmt.Printf("cluster file stats: %+v\n", stats)

	//err = os.Chown(clusterFileDir, os.Getuid(), os.Getgid())
	//if err != nil {
	//	return nil, fmt.Errorf("could not change owner of the temporary directory for cluster file: %w", err)
	//}

	//suffix := rand.Uint64()
	//clusterFileVolume := fmt.Sprintf("tmp-tigerbeetle-%x", suffix)

	// hostConfigModifier mounts volume to container cluster file
	hostConfigModifier := func(hostConfig *container.HostConfig) {
		hostConfig.Mounts = []mount.Mount{
			{
				Type:           mount.TypeBind,
				Source:         clusterFileDir + "/",
				Target:         "/data",
				ReadOnly:       false,
				Consistency:    "",
				BindOptions:    nil,
				VolumeOptions:  nil,
				TmpfsOptions:   nil,
				ClusterOptions: nil,
			},
		}
	}

	// Run a container to format 0_0.tigerbeetle cluster file and wait for it to complete
	formatContainerReq := testcontainers.ContainerRequest{
		Image: img,
		Cmd: []string{
			"format",
			fmt.Sprintf("--cluster=%d", clusterID),
			fmt.Sprintf("--replica=%d", replicaID),
			fmt.Sprintf("--replica-count=%d", replicaCount),
			fmt.Sprintf("/data/%s", clusterFileName),
		},
		WaitingFor:         wait.ForExit(),
		Privileged:         true,
		HostConfigModifier: hostConfigModifier,
	}

	// start the formatContainer
	formatContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: formatContainerReq,
		Started:          true,
	})
	if err != nil {
		return nil, fmt.Errorf("error while formatting the Tigerbeetle cluster file: %w", err)
	}

	// Wait for the formatContainer to terminate
	if err = formatContainer.Terminate(ctx); err != nil {
		return nil, fmt.Errorf("error while formatting the Tigerbeetle cluster file, failed to terminate temporary Tigerbeetle container: %w", err)
	}

	// Define the main Tigerbeetle container request
	req := testcontainers.ContainerRequest{
		Image:        img,
		ExposedPorts: []string{defaultPort + "/tcp"},
		Cmd: []string{
			"start",
			fmt.Sprintf("--addresses=0.0.0.0:%s", defaultPort),
			fmt.Sprintf("/data/%s", clusterFileName),
		},
		Privileged:         true,
		WaitingFor:         wait.ForListeningPort(defaultPort),
		HostConfigModifier: hostConfigModifier,
	}

	genericContainerRequest := testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	}

	// Apply user defined options, it is undefined behavior to modify ExposedPorts, since it is hardcoded in other commands
	for _, opt := range opts {
		if err = opt.Customize(&genericContainerRequest); err != nil {
			return nil, fmt.Errorf("failed to apply option: %w", err)
		}
	}

	// Start the Tigerbeetle container
	tbContainer, err := testcontainers.GenericContainer(ctx, genericContainerRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to start Tigerbeetle container: %w", err)
	}

	return &Container{
		Container:         tbContainer,
		clusterFileVolume: clusterFileDir,
	}, nil
}

// Terminate exits the container then cleans-up the temporary directory containing cluster file
func (c *Container) Terminate(ctx context.Context) error {
	err := c.Container.Terminate(ctx)
	// c.clusterFileVolume
	return err
}
