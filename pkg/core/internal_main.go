package core

import (
	"flag"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

var options = struct {
	CloudflareAPIEmail string
	CloudflareAPIKey   string
	CloudflareProxy    string
	CloudflareTTL      string
	DNSName            string
}{
	CloudflareAPIEmail: os.Getenv("CF_API_EMAIL"),
	CloudflareAPIKey:   os.Getenv("CF_API_KEY"),
	CloudflareProxy:    os.Getenv("CF_PROXY"),
	CloudflareTTL:      os.Getenv("CF_TTL"),
	DNSName:            os.Getenv("DNS_NAME"),
}

// IPLister should implement Setup and List function
// to efficientlly plugin the mechanism of listing
// IP address.
type IPLister interface {
	Setup()
	List() []string
}

func Main(iplister IPLister) {
	iplister.Setup()

	flag.StringVar(&options.DNSName, "dns-name", options.DNSName, "the dns name for the nodes, comma-separated for multiple (same root)")
	flag.StringVar(&options.CloudflareAPIEmail, "cloudflare-api-email", options.CloudflareAPIEmail, "the email address to use for cloudflare")
	flag.StringVar(&options.CloudflareAPIKey, "cloudflare-api-key", options.CloudflareAPIKey, "the key to use for cloudflare")
	flag.StringVar(&options.CloudflareProxy, "cloudflare-proxy", options.CloudflareProxy, "enable cloudflare proxy on dns (default false)")
	flag.StringVar(&options.CloudflareTTL, "cloudflare-ttl", options.CloudflareTTL, "ttl for dns (default 120)")
	flag.Parse()

	if options.CloudflareAPIEmail == "" {
		flag.Usage()
		log.Fatalln("cloudflare api email is required")
	}
	if options.CloudflareAPIKey == "" {
		flag.Usage()
		log.Fatalln("cloudflare api key is required")
	}

	dnsNames := strings.Split(options.DNSName, ",")
	if len(dnsNames) == 1 && dnsNames[0] == "" {
		flag.Usage()
		log.Fatalln("dns name is required")
	}

	cloudflareProxy, err := strconv.ParseBool(options.CloudflareProxy)
	if err != nil {
		log.Println("CloudflareProxy config not found or incorrect, defaulting to false")
		cloudflareProxy = false
	}

	cloudflareTTL, err := strconv.Atoi(options.CloudflareTTL)
	if err != nil {
		log.Println("CloudflareTTL config not found or incorrect, defaulting to 120")
		cloudflareTTL = 120
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

	stop := make(chan struct{})
	defer close(stop)

	var lastIPs []string
	resync := func() {
		log.Println("resyncing")

		ips := iplister.List()

		sort.Strings(ips)
		log.Println("ips:", ips)
		if strings.Join(ips, ",") == strings.Join(lastIPs, ",") {
			log.Println("no change detected")
			return
		}
		lastIPs = ips

		err = sync(ips, dnsNames, cloudflareTTL, cloudflareProxy)
		if err != nil {
			log.Println("failed to sync", err)
		}
	}

	factory := informers.NewSharedInformerFactory(client, time.Minute)
	informer := factory.Core().V1().Nodes().Informer()
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			resync()
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			resync()
		},
		DeleteFunc: func(obj interface{}) {
			resync()
		},
	})
	informer.Run(stop)

	select {}
}
