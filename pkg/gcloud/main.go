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

type arrayFlag []string

func (af *arrayFlag) String() string {
	return strings.Join(*af, " ")
}

func (af *arrayFlag) Set(value string) error {
	*af = append(*af, value)
	return nil
}

var options = struct {
	Projects arrayFlag
	Filters  arrayFlag
}{
	Projects: strings.Split(os.Getenv("PROJECTS"), ","),
	Filters:  strings.Split(os.Getenv("FILTERS"), ","),
}

type gcloud_ip_lister struct {
	service *compute.Service
}

func (gcloud_ip_lister *gcloud_ip_lister) Setup() {
	flag.Var(&options.Projects, "projects", "GCP projects")
	flag.Var(&options.Filters, "filters", "instance filters")

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
	for _, project := range options.Projects {
		zoneListCall := gcloud_ip_lister.service.Zones.List(project)
		zoneList, err := zoneListCall.Do()
		if err != nil {
			log.Println("Error", err)
		} else {
			for _, zone := range zoneList.Items {
				instanceListCall := gcloud_ip_lister.service.Instances.List(project, zone.Name)
				instanceListCall.Filter(strings.Join(options.Filters[:], " "))
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
	}
	return ips
}

func main() {
	ip_lister := gcloud_ip_lister{}
	core.Main(&ip_lister)
}
