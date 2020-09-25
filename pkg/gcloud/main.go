package main

import (
	"flag"
	"log"
	"os"
	"strings"

	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/compute/v1"

	"github.com/Matts966/kubernetes-cloudflare-syncer/pkg/core"
)

var options = struct {
	Project string
	Filter  string
}{
	Project: os.Getenv("PROJECT"),
	Filter:  os.Getenv("FILTER"),
}

type gcloud_ip_lister struct {
	service *compute.Service
}

func (gcloud_ip_lister *gcloud_ip_lister) Setup() {
	flag.StringVar(&options.Project, "project", options.Project, "GCP project ID")
	flag.StringVar(&options.Filter, "filter", options.Filter, "instance filters")

	log.SetOutput(os.Stdout)

	ctx := context.Background()
	client, err := google.DefaultClient(ctx, compute.ComputeScope)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	computeService, err := compute.New(client)
	gcloud_ip_lister.service = computeService
}

func (gcloud_ip_lister *gcloud_ip_lister) List() []string {
	var ips []string
	zoneListCall := gcloud_ip_lister.service.Zones.List(options.Project)
	zoneList, err := zoneListCall.Do()
	if err != nil {
		log.Println("Error", err)
	} else {
		for _, zone := range zoneList.Items {
			instanceListCall := gcloud_ip_lister.service.Instances.List(project, zone.Name)
			instanceListCall.Filter(strings.Join(options.Filter))
			instanceList, err := instanceListCall.Do()
			if err != nil {
				log.Println("Error", err)
			} else {
				for _, instance := range instanceList.Items {
					for _, networkInterface := range instance.NetworkInterfaces {
						for _, accessConfig := range networkInterface.AccessConfigs {
							ips = append(ips, accessConfig.NatIP)
						}
					}
				}
			}
		}
	}
	return ips
}

func main() {
	ip_lister := gcloud_ip_lister{}
	core.Main(&ip_lister)
}
