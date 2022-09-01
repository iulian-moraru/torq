package virtual_network

import (
	"archive/tar"
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/cockroachdb/errors"
	"github.com/docker/docker/api/types"
	dockercontainer "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/go-connections/nat"
	"io"
	"io/ioutil"
	"log"
	"os"
	"time"
)

type ContainerConfig struct {
	Name       string
	Image      string
	Env        []string
	Binds      []string
	Cmd        []string
	MappedPort string
	Instance   dockercontainer.ContainerCreateCreatedBody
}

type DockerDevEnvironment struct {
	Client           *client.Client
	NetworkName      string
	NetworkingConfig network.NetworkingConfig
	Containers       map[string]*ContainerConfig
	SharedVolumeName string
}

func (de *DockerDevEnvironment) CleanupContainers(ctx context.Context) error {
	for _, container := range de.Containers {
		err := de.FindAndRemoveContainer(ctx, container.Name)
		if err != nil {
			return err
		}
	}
	return nil
}

func (de *DockerDevEnvironment) CleanupDefaultVolumes(ctx context.Context) error {
	for _, container := range de.Containers {
		err := de.FindAndRemoveVolume(ctx, container.Name)
		if err != nil {
			return err
		}
	}
	return nil
}

func (de *DockerDevEnvironment) AddContainer(name string, image string, binds []string, env []string, cmd []string,
	mappedPort string) *ContainerConfig {
	if de.Containers == nil {
		de.Containers = make(map[string]*ContainerConfig)
	}

	de.Containers[name] = &ContainerConfig{
		Name:       name,
		Image:      image,
		Binds:      binds,
		Env:        env,
		Cmd:        cmd,
		MappedPort: mappedPort,
		Instance:   dockercontainer.ContainerCreateCreatedBody{},
	}
	return de.Containers[name]
}

func (de *DockerDevEnvironment) CreateContainer(ctx context.Context, container *ContainerConfig) (err error) {
	hostConfig := &dockercontainer.HostConfig{
		Binds: container.Binds,
	}
	openPorts := nat.PortSet{}
	if container.MappedPort != "" {
		hostConfig.PortBindings = nat.PortMap{
			nat.Port(container.MappedPort) + "/tcp": []nat.PortBinding{
				{
					HostIP:   "0.0.0.0",
					HostPort: container.MappedPort,
				},
			},
		}
		openPorts = nat.PortSet{
			nat.Port(container.MappedPort) + "/tcp": struct{}{},
		}
	}

	r, err := de.Client.ContainerCreate(ctx, &dockercontainer.Config{
		Image:        container.Image,
		Env:          container.Env,
		Cmd:          container.Cmd,
		ExposedPorts: openPorts,
	}, hostConfig, &de.NetworkingConfig, nil, container.Name)
	if err != nil {
		return errors.Newf("Creating %s container: %v", container.Name, err)
	}
	container.Instance = r
	return nil
}

func (de *DockerDevEnvironment) InitContainer(ctx context.Context, container *ContainerConfig) error {
	err := de.CreateContainer(ctx, container)
	if err != nil {
		return err
	}
	if err := de.Client.ContainerStart(ctx, container.Instance.ID, types.ContainerStartOptions{}); err != nil {
		return errors.Newf("can't start %s container: %v", container.Name, err)
	}
	return nil
}

func (de *DockerDevEnvironment) StartContainer(ctx context.Context, cc *ContainerConfig) (err error) {
	container, err := de.FindContainerByName(ctx, cc.Name)
	if err != nil {
		return errors.Newf("Can't find container with the name %s: %v", cc.Name, err)
	}
	if err := de.Client.ContainerStart(ctx, container.ID, types.ContainerStartOptions{}); err != nil {
		return errors.Newf("can't start %s container: %v", cc.Name, err)
	}
	cc.Instance = dockercontainer.ContainerCreateCreatedBody{ID: container.ID}
	return nil
}

func (de *DockerDevEnvironment) StopContainer(ctx context.Context, name string) (err error) {
	container, err := de.FindContainerByName(ctx, name)
	if err != nil {
		return errors.Newf("can't find container with the name %s: %v", name, err)
	}
	timeout := 30 * time.Second
	if err := de.Client.ContainerStop(ctx, container.ID, &timeout); err != nil {
		return errors.Newf("can't stop %s container: %v", name, err)
	}
	return nil
}

func (de *DockerDevEnvironment) CreateNetwork(ctx context.Context) (nc network.NetworkingConfig, err error) {
	nw, err := de.Client.NetworkCreate(ctx, de.NetworkName, types.NetworkCreate{})
	if err != nil {
		return nc, errors.Newf("Creating %s network: %v", de.NetworkName, err)
	}
	nc = network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{},
	}
	nc.EndpointsConfig[nw.ID] = &network.EndpointSettings{Links: []string{de.NetworkName + "-btcd:blockchain"}}
	return nc, err
}

func (de *DockerDevEnvironment) FindAndRemoveContainer(ctx context.Context, name string) error {
	container, err := de.FindContainerByName(ctx, name)
	if err != nil {
		return errors.Newf("Removing %s container: %v", name, err)
	}
	if container != nil {
		log.Printf("%s container found; removing\n", name)

		if container.State == "running" {
			if err := de.Client.ContainerStop(ctx, container.ID, nil); err != nil {
				return errors.Newf("Stopping %s container: %v", name, err)
			}
		}
		if err := de.Client.ContainerRemove(ctx, container.ID, types.ContainerRemoveOptions{}); err != nil {
			return errors.Newf("Removing %s container: %v", name, err)
		}
	}
	return nil
}

func (de *DockerDevEnvironment) FindAndRemoveVolume(ctx context.Context, name string) error {
	volume, err := de.FindVolumeByName(ctx, name)
	if err != nil {
		return errors.Newf("Removing old %s volume: %v", name, err)
	}
	if volume != nil {
		log.Printf("Old %s volume found; removing\n", name)
		if err := de.Client.VolumeRemove(ctx, volume.Name, false); err != nil {
			return errors.Newf("Removing old %s volume: %v", name, err)
		}
	}
	return nil
}

func (de *DockerDevEnvironment) FindVolumeByName(ctx context.Context, name string) (*types.Volume, error) {
	volumes, err := de.Client.VolumeList(ctx, filters.Args{})
	if err != nil {
		return nil, err
	}
	for _, volume := range volumes.Volumes {
		if volume.Name == name {
			return volume, nil
		}
	}
	return nil, nil
}

func (de *DockerDevEnvironment) FindAndRemoveNetwork(ctx context.Context, name string) error {
	network, err := de.FindNetworkByName(ctx, name)
	if err != nil {
		return errors.Newf("Removing old %s network: %v", name, err)
	}
	if network != nil {
		log.Printf("Old %s network found; removing\n", name)
		if err := de.Client.NetworkRemove(ctx, network.ID); err != nil {
			return errors.Newf("Removing old %s network: %v", name, err)
		}
	}
	return nil
}

func (de *DockerDevEnvironment) FindNetworkByName(ctx context.Context, name string) (*types.NetworkResource, error) {
	networks, err := de.Client.NetworkList(ctx, types.NetworkListOptions{})
	if err != nil {
		return nil, err
	}
	for _, network := range networks {
		if network.Name == name {
			return &network, nil
		}
	}
	return nil, nil
}

func (de *DockerDevEnvironment) FindContainerByName(ctx context.Context, name string) (*types.Container, error) {
	containers, err := de.Client.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		return nil, err
	}
	for _, container := range containers {
		for _, containerName := range container.Names {
			// internal docker names have leading slashes; trim off
			if containerName[1:] == name {
				return &container, nil
			}
		}
	}
	return nil, nil
}

func (de *DockerDevEnvironment) BuildImage(ctx context.Context, path string, name string) error {
	tar, err := archive.TarWithOptions(path, &archive.TarOptions{ExcludePatterns: []string{"web/node_modules", ".git"}})
	if err != nil {
		return errors.Newf("Creating %s archive: %v", name, err)
	}

	opts := types.ImageBuildOptions{
		Dockerfile: "Dockerfile",
		Tags:       []string{name},
		Remove:     true,
	}

	res, err := de.Client.ImageBuild(ctx, tar, opts)
	if err != nil {
		return errors.Newf("Building %s docker image: %v", name, err)
	}
	defer res.Body.Close()
	err = printBuildOutput(res.Body)
	if err != nil {
		return errors.Newf("Printing build output for %s docker image: %v", name, err)
	}
	return nil
}

type ErrorLine struct {
	Error       string      `json:"error"`
	ErrorDetail ErrorDetail `json:"errorDetail"`
}

type ErrorDetail struct {
	Message string `json:"message"`
}

func printBuildOutput(rd io.Reader) error {
	var lastLine string

	scanner := bufio.NewScanner(rd)
	for scanner.Scan() {
		lastLine = scanner.Text()
		if len(os.Getenv("DEBUG")) > 0 {
			fmt.Println(scanner.Text())
		}
	}

	errLine := &ErrorLine{}
	json.Unmarshal([]byte(lastLine), errLine)
	if errLine.Error != "" {
		return errors.New(errLine.Error)
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}

func ExecJSONReturningCommand(ctx context.Context, cli *client.Client,
	container dockercontainer.ContainerCreateCreatedBody,
	cmd []string, returnObject interface{}) error {

	bufStdout, bufStderr, err := ExecCommand(ctx, cli, container, cmd)
	if err != nil {
		return errors.Wrap(err, "Exec command on container")
	}
	if len(bufStderr.Bytes()) > 0 {
		log.Println("std error not empty")
		return errors.New("Stderr not empty")
	}

	err = json.Unmarshal(bufStdout.Bytes(), returnObject)
	if err != nil {
		return errors.Wrap(err, "json unmarshal")
	}
	return nil
}

func ExecCommand(ctx context.Context, cli *client.Client,
	container dockercontainer.ContainerCreateCreatedBody,
	cmd []string) (bufStdout bytes.Buffer, bufStderr bytes.Buffer, err error) {

	c := types.ExecConfig{AttachStdout: true, AttachStderr: true,
		Cmd: cmd}
	execID, _ := cli.ContainerExecCreate(ctx, container.ID, c)

	res, err := cli.ContainerExecAttach(ctx, execID.ID, types.ExecStartCheck{})
	if err != nil {
		return bufStdout, bufStderr, errors.Wrap(err, "Container exec start")
	}

	err = cli.ContainerExecStart(ctx, execID.ID, types.ExecStartCheck{})
	if err != nil {
		return bufStdout, bufStderr, errors.Wrap(err, "Container exec start")
	}

	// stdcopy.StdCopy(os.Stdout, os.Stderr, res.Reader)
	stdcopy.StdCopy(&bufStdout, &bufStderr, res.Reader)
	// DEBUG Tip: uncomment below to see raw output of commands
	if len(os.Getenv("DEBUG")) > 0 {
		log.Printf("%s\n", string(bufStdout.Bytes()))
		log.Printf("%s\n", string(bufStderr.Bytes()))
	}
	return bufStdout, bufStderr, nil
}

type noRetryError struct{}

func (nre noRetryError) Error() string {
	return "Skip retries"
}

func Retry(operation func() error, delayMilliseconds int, maxWaitMilliseconds int) error {
	totalWaited := 0
	for {
		if totalWaited > maxWaitMilliseconds {
			return errors.New("Exceeded maximum wait period")
		}
		err := operation()
		var noRetry noRetryError
		if errors.As(err, &noRetry) {
			return err
		}
		if err == nil {
			break
		}
		log.Println("Checking...")
		time.Sleep(time.Duration(delayMilliseconds) * time.Millisecond)
		totalWaited += delayMilliseconds
	}
	return nil
}

func WriteConnectionDetails(ctx context.Context, cli *client.Client, name string, nodeIp string) error {
	// Copy bobs macaroon and tls file to local directory
	tlsFileReader, _, err := cli.CopyFromContainer(ctx, name, "/root/.lnd/tls.cert")
	if err != nil {
		return errors.Newf("Copying tls file: %v\n", err)
	}
	// close fi on exit and check for its returned error
	defer func() {
		if err := tlsFileReader.Close(); err != nil {
			panic(err)
		}
	}()

	tlsTar := tar.NewReader(tlsFileReader)
	// hdr gives you the header of the tar file
	_, err = tlsTar.Next()
	if err == io.EOF || err != nil {
		// EOF == end of tar archive
		return errors.Newf("Reading tls tar header: %v\n", err)
	}
	tlsBuf := new(bytes.Buffer)
	_, err = tlsBuf.ReadFrom(tlsTar)
	if err != nil {
		return errors.Newf("Reading tls tar: %v\n", err)
	}
	// write the whole body at once
	err = ioutil.WriteFile("virtual_network/generated_files/tls.cert", tlsBuf.Bytes(), 0644)
	if err != nil {
		panic(err)
	}

	macaroonFileReader, _, err := cli.CopyFromContainer(ctx, name,
		"/root/.lnd/data/chain/bitcoin/simnet/admin.macaroon")
	if err != nil {
		return errors.Newf("Copying tls file: %v\n", err)
	}
	// close fi on exit and check for its returned error
	defer func() {
		if err := macaroonFileReader.Close(); err != nil {
			panic(err)
		}
	}()

	// file comes out as a tar, untar it
	macaroonTar := tar.NewReader(macaroonFileReader)
	// hdr gives you the header of the tar file
	_, err = macaroonTar.Next()
	if err == io.EOF || err != nil {
		// EOF == end of tar archive
		return errors.Newf("Reading macaroon tar header: %v\n", err)
	}
	macaroonBuf := new(bytes.Buffer)
	_, err = macaroonBuf.ReadFrom(macaroonTar)
	if err != nil {
		return errors.Newf("Reading macaroon tar: %v\n", err)
	}

	// write the whole body at once
	err = ioutil.WriteFile("virtual_network/generated_files/admin.macaroon", macaroonBuf.Bytes(), 0644)
	if err != nil {
		panic(err)
	}

	//// write the whole body at once
	//err = ioutil.WriteFile("virtual_network/generated_files/conn_details.txt", []byte("Connect to localhost:10009"),
	//	0644)
	//if err != nil {
	//	panic(err)
	//}
	return nil
}
