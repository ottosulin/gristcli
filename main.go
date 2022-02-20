package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/go-connections/nat"
)

func main() {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}

	defaultemail := "test@example.com"
	defaultctrname := "gristcontainer"

	c := make(chan os.Signal, 1)
	done := make(chan bool, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-c
		fmt.Println(sig)
		log.Println("Stopping the Grist container...")
		err := cli.ContainerStop(ctx, defaultctrname, nil)
		if err != nil {
			panic(err)
		}
		err = cli.ContainerRemove(ctx, defaultctrname, types.ContainerRemoveOptions{})
		if err != nil {
			panic(err)
		}
		done <- true
	}()
	runtime.Gosched()

	homedir := flag.String("dir", "", "a string")
	useremail := flag.String("email", "", "a string")

	if len(*useremail) < 5 {
		useremail = &defaultemail
		log.Println("Did not specify email, using default: ", defaultemail)
	}

	if len(*homedir) < 1 {
		dir, err := os.UserHomeDir()
		if err != nil {
			log.Panicln("Did not receive Grist home directory and unable to find user home directory: ", err)
		}
		newpath := filepath.Join(dir, "grist")
		err = os.MkdirAll(newpath, os.ModePerm)
		if err != nil {
			log.Panicln("Did not receive Grist home directory and unable to create the default directory: ", err)
		}
		fmt.Println("Using default Grist home directory: ", newpath)
		homedir = &newpath
	}
	fmt.Println("Mounting: ", *homedir)

	reader, err := cli.ImagePull(ctx, "gristlabs/grist", types.ImagePullOptions{})
	if err != nil {
		panic(err)
	}

	defer reader.Close()
	io.Copy(os.Stdout, reader)

	hostConfig := &container.HostConfig{
		PortBindings: nat.PortMap{
			"8484/tcp": []nat.PortBinding{
				{
					HostIP:   "127.0.0.1",
					HostPort: "8484",
				},
			},
		},
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeBind,
				Source: *homedir,
				Target: "/persist",
			},
		},
	}

	envs := make([]string, 0)
	emailstring := fmt.Sprintf("%s=%s", "GRIST_DEFAULT_EMAIL", *useremail)
	envs = append(envs, emailstring)

	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: "gristlabs/grist",
		Tty:   false,
		ExposedPorts: nat.PortSet{
			"8484/tcp": struct{}{},
		},
		Env: envs,
	}, hostConfig, nil, nil, defaultctrname)
	if err != nil {
		panic(err)
	}

	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		panic(err)
	}

	statusCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			panic(err)
		}
	case <-statusCh:
	}

	out, err := cli.ContainerLogs(ctx, resp.ID, types.ContainerLogsOptions{ShowStdout: true})
	if err != nil {
		panic(err)
	}
	stdcopy.StdCopy(os.Stdout, os.Stderr, out)

	<-done
	fmt.Println("Exiting..")
}
