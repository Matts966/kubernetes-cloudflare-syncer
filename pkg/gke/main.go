package main

import (
	"flag"
	"log"
	"os"
	"time"

	"github.com/Matts966/kubernetes-cloudflare-syncer/pkg/core"
	k8s_core_v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	lister_v1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/rest"
)

var options = struct {
	UseInternalIP  bool
	SkipExternalIP bool
	NodeSelector   string
}{
	UseInternalIP:  os.Getenv("USE_INTERNAL_IP") != "",
	SkipExternalIP: os.Getenv("SKIP_EXTERNAL_IP") != "",
	NodeSelector:   os.Getenv("NODE_SELECTOR"),
}

type gke_ip_lister struct {
	client       *kubernetes.Clientset
	lister       lister_v1.NodeLister
	nodeSelector labels.Selector
}

func (gke_ip_lister *gke_ip_lister) Setup() {
	flag.BoolVar(&options.UseInternalIP, "use-internal-ip", options.UseInternalIP, "use internal ips too if external ip's are not available")
	flag.BoolVar(&options.SkipExternalIP, "skip-external-ip", options.SkipExternalIP, "don't sync external IPs (use in conjunction with --use-internal-ip)")
	flag.StringVar(&options.NodeSelector, "node-selector", options.NodeSelector, "node selector query")

	gke_ip_lister.nodeSelector = labels.NewSelector()
	if options.NodeSelector != "" {
		selector, err := labels.Parse(options.NodeSelector)
		if err != nil {
			log.Printf("node selector is invalid: %v\n", err)
		} else {
			gke_ip_lister.nodeSelector = selector
		}
	}

	log.SetOutput(os.Stdout)

	cfg, err := rest.InClusterConfig()
	if err != nil {
		log.Fatalln(err)
	}

	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		log.Fatalln(err)
	}
	gke_ip_lister.client = client

	factory := informers.NewSharedInformerFactory(gke_ip_lister.client, time.Minute)
	gke_ip_lister.lister = factory.Core().V1().Nodes().Lister()
}

func (gke_ip_lister *gke_ip_lister) List() ([]string, error) {
	nodes, err := gke_ip_lister.lister.List(gke_ip_lister.nodeSelector)
	if err != nil {
		return nil, err
	}

	var ips []string
	if !options.SkipExternalIP {
		for _, node := range nodes {
			if nodeIsReady(node) {
				for _, addr := range node.Status.Addresses {
					if addr.Type == k8s_core_v1.NodeExternalIP {
						ips = append(ips, addr.Address)
					}
				}
			}
		}
	}
	if options.UseInternalIP && len(ips) == 0 {
		for _, node := range nodes {
			if nodeIsReady(node) {
				for _, addr := range node.Status.Addresses {
					if addr.Type == k8s_core_v1.NodeInternalIP {
						ips = append(ips, addr.Address)
					}
				}
			}
		}
	}
	return ips, nil
}

func nodeIsReady(node *k8s_core_v1.Node) bool {
	for _, condition := range node.Status.Conditions {
		if condition.Type == k8s_core_v1.NodeReady && condition.Status == k8s_core_v1.ConditionTrue {
			return true
		}
	}

	return false
}

func main() {
	ip_lister := gke_ip_lister{}
	core.Main(&ip_lister)
}
