package tigerbeetle

import (
	"context"
	"fmt"
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
	clusterFileDir string
}

// Address returns the connection address of the Tigerbeetle container
// The ClusterID is set to 0 for containers started using this package
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

// Run creates a temporary directory for 0_0.tigerbeetle cluster file and starts the Tigerbeetle at default 3000 port
func Run(ctx context.Context, img string, opts ...testcontainers.ContainerCustomizer) (*Container, error) {
	clusterFileDir, err := os.MkdirTemp("", "")
	if err != nil {
		return nil, fmt.Errorf("could not create temporary directory for cluster file: %w", err)
	}

	// hostConfigModifier mounts cluster file directory to container
	hostConfigModifier := func(hostConfig *container.HostConfig) {
		hostConfig.SecurityOpt = []string{"seccomp=unconfined"}
		hostConfig.Mounts = []mount.Mount{
			{
				Type:           mount.TypeBind,
				Source:         clusterFileDir,
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
		HostConfigModifier: hostConfigModifier,
	}

	// start the formatContainer

	formatContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: formatContainerReq,
		ProviderType:     testcontainers.ProviderDocker,
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
		Container:      tbContainer,
		clusterFileDir: clusterFileDir,
	}, nil
}

// Terminate exits the container then cleans-up the temporary directory containing cluster file
func (c *Container) Terminate(ctx context.Context) error {
	err := c.Container.Terminate(ctx)
	_ = os.RemoveAll(c.clusterFileDir)
	return err
}
