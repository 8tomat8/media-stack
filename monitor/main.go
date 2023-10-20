package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

var (
	setupName               = flag.String("setupName", "media-stack", "")
	containerRestartTimeout = flag.Int("restartTimeout", 30, "in seconds")
)

const (
	projectNameLabel = "com.docker.compose.project"
)

var (
	sonarrPort      = flag.String("sonarrPort", "8989", "")
	radarrPort      = flag.String("radarrPort", "7878", "")
	bazarrPort      = flag.String("bazarrPort", "6767", "")
	prowlarrPort    = flag.String("prowlarrPort", "9696", "")
	qbittorrentPort = flag.String("qbittorrentPort", "5080", "")
)

type service struct {
	name        string
	containerID string
	port        *string
	host        string
}

var services = map[string]*service{
	"prowlarr": {
		name: "prowlarr",
		port: prowlarrPort,
		host: "localhost",
	},
	"bazarr": {
		name: "bazarr",
		port: bazarrPort,
		host: "localhost",
	},
	"sonarr": {
		name: "sonarr",
		port: sonarrPort,
		host: "localhost",
	},
	"radarr": {
		name: "radarr",
		port: radarrPort,
		host: "localhost",
	},
	"qbittorrent": {
		name: "qbittorrent",
		port: qbittorrentPort,
		host: "localhost",
	},
}

func init() {
	http.DefaultClient.Timeout = time.Second * 3
}

func main() {
	flag.Parse()
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}
	defer cli.Close()

	// ss, err := cli.ContainerList(ctx, types.ContainerListOptions{
	// 	All: true,
	// })
	// if err != nil {
	// 	panic(err)
	// }
	// for _, s := range ss {
	// 	fmt.Println(s)
	// }

	wg := &sync.WaitGroup{}
	for {
		log.Info("starting probe")
		wg.Wait()
		time.Sleep(time.Minute)
		for _, service := range services {
			log.Infof("probing %s, on port %s", service.name, *service.port)
			containerID, err := getContainerID(ctx, cli, service.name)
			if err != nil {
				log.Infof("service %s is not running", service.name)
				continue
			}

			wg.Add(1)
			service := service
			go func() {
				defer wg.Done()
				if !livenessProbe(ctx, service) {
					log.Infof("service %s is not alive, restarting", service.name)
					err := restartContainer(ctx, cli, containerID)
					if err != nil {
						log.Error(errors.Wrap(err, service.name))
					}
				}
			}()
		}
	}
}

func getContainerID(ctx context.Context, cli *client.Client, name string) (string, error) {
	list, err := cli.ContainerList(ctx, types.ContainerListOptions{
		All: true,
	})
	if err != nil {
		return "", errors.Wrap(err, "failed to list containers")
	}
	for _, cont := range list {
		proj, ok := cont.Labels[projectNameLabel]
		if !ok || proj != *setupName {
			continue
		}
		if len(cont.Names) > 0 && strings.Contains(cont.Names[0], name) {
			return cont.ID, nil
		}
	}
	return "", errors.Errorf("container %s not found", name)
}

func livenessProbe(ctx context.Context, svc *service) (success bool) {
	// Check both, localhost and the service name to handle both VPN and VPNless setups
	for _, host := range []string{svc.host, svc.name} {
		uri := fmt.Sprintf("http://%s:%s", host, *svc.port)
		resp, err := http.Get(uri)
		if err != nil && !os.IsTimeout(err) {
			log.Infof("probing %s: %s", uri, err)
			continue
		}
		resp.Body.Close()
		log.Infof("probing %s: %d", uri, resp.StatusCode)
		if resp.StatusCode != 0 {
			svc.host = host // optimize for next time
			return true
		}
	}
	return false
}

func restartContainer(ctx context.Context, cli *client.Client, id string) error {
	err := cli.ContainerRestart(ctx, id, container.StopOptions{
		Timeout: containerRestartTimeout,
	})
	if err != nil {
		return errors.Wrap(err, "failed to restart")
	}
	return nil
}
