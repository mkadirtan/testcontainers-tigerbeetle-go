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

// These constants are not very flexible for now
const (
	// tbImage Tigerbeetle official image in latest tag
	tbImage = "ghcr.io/tigerbeetle/tigerbeetle:latest"
	// tbPort the default port used in the Tigerbeetle docs
	tbPort = "3000"
	// clusterID Cluster ID for Tigerbeetle
	clusterID = "0"
	// replicaID Replica ID for testing
	replicaID = "0"
	// replicaCount Number of replicas for this cluster
	replicaCount = "1"
)

type Container struct {
	testcontainers.Container
	Host    string
	Port    string
	dataDir string
}

// RunContainer creates a temporary directory with 0_0.tigerbeetle cluster file and starts the Tigerbeetle at default 3000 port
// The temporary directory is cleaned-up upon Terminate
// The port is unchangeable for now
func RunContainer(ctx context.Context, opts ...testcontainers.ContainerCustomizer) (*Container, error) {
	// Create the temporary directory to store 0_0.tigerbeetle cluster file
	dataDir, err := os.MkdirTemp("", "tbdata")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary directory: %w", err)
	}

	// hostConfigModifier mounts the temporary directory to the container
	hostConfigModifier := func(hostConfig *container.HostConfig) {
		hostConfig.Mounts = []mount.Mount{
			{
				Type:           mount.TypeBind,
				Source:         dataDir,
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
		Image: tbImage,
		Cmd: []string{
			"format",
			"--cluster=" + clusterID,
			"--replica=" + replicaID,
			"--replica-count=" + replicaCount,
			"/data/0_0.tigerbeetle",
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
		return nil, fmt.Errorf("failed to start Tigerbeetle formatContainer: %w", err)
	}

	// Wait for the formatContainer to terminate
	if err = formatContainer.Terminate(ctx); err != nil {
		return nil, fmt.Errorf("failed to terminate Tigerbeetle formatContainer: %w", err)
	}

	// Define the main Tigerbeetle container request
	req := testcontainers.ContainerRequest{
		Image:        tbImage,
		ExposedPorts: []string{tbPort + "/tcp"},
		Cmd: []string{
			"start",
			fmt.Sprintf("--addresses=0.0.0.0:%s", tbPort),
			fmt.Sprintf("/data/%s", "0_0.tigerbeetle"),
		},
		WaitingFor:         wait.ForListeningPort(tbPort),
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
		return nil, fmt.Errorf("failed to start Tigerbeetle formatContainer: %w", err)
	}

	// Get the Tigerbeetle host and port information
	host, err := tbContainer.Host(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get formatContainer host: %w", err)
	}
	port, err := tbContainer.MappedPort(ctx, tbPort)
	if err != nil {
		return nil, fmt.Errorf("failed to get formatContainer port: %w", err)
	}

	return &Container{
		Container: tbContainer,
		Host:      host,
		Port:      port.Port(),
		dataDir:   dataDir,
	}, nil
}

// Terminate exits the container then cleans-up the temporary directory containing cluster file
func (t *Container) Terminate(ctx context.Context) error {
	err := t.Container.Terminate(ctx)
	// During termination remove the temporary folder containing cluster file
	_ = os.RemoveAll(t.dataDir)
	return err
}
