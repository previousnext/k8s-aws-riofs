package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/golang/glog"
	"github.com/kubernetes-incubator/external-storage/lib/controller"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const apiVersion = "s3.aws.skpr.io/riofs"

func main() {
	flag.Parse()
	flag.Set("logtostderr", "true")

	// Create an InClusterConfig and use it to create a client for the controller
	// to use to communicate with Kubernetes
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Fatal("Failed to create config: %s", err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal("Failed to create client: %s", err)
	}

	// The controller needs to know what the server version is because out-of-tree
	// provisioners aren't officially supported until 1.5
	serverVersion, err := clientset.Discovery().ServerVersion()
	if err != nil {
		glog.Fatal("Error getting server version: %s", err)
	}

	// Create the provisioner: it implements the Provisioner interface expected by the controller.
	provisioner, err := NewProvisioner()
	if err != nil {
		glog.Fatal("Failed to create provisioner: %s", err)
	}

	glog.Infof("Running provisioner: %s", apiVersion)

	// Start the provision controller which will dynamically provision RioFS PVs
	pc := controller.NewProvisionController(
		clientset,
		apiVersion,
		provisioner,
		serverVersion.GitVersion,
		controller.CreateProvisionedPVInterval(time.Minute*10),
		controller.LeaseDuration(time.Minute*10),
	)
	pc.Run(wait.NeverStop)
}

// NewProvisioner is used to build an S3 bucket provisioner.
func NewProvisioner() (controller.Provisioner, error) {
	// Region to provision the storage in.
	region := os.Getenv("AWS_REGION")
	if region == "" {
		return nil, fmt.Errorf("environment variable AWS_REGION not found")
	}

	// S3_BUCKET_NAME_FORMAT allows for backwards compatibility with other S3 tools.
	//   eg. My existing S3 resources use the pattern "{{ .PVC.ObjectMeta.Namespace }}-{{ .PVC.ObjectMeta.Name }}"
	format := os.Getenv("S3_BUCKET_NAME_FORMAT")
	if format == "" {
		format = "{{ .PVC.ObjectMeta.Namespace }}-{{ .PVName }}"
	}

	provisioner := &riofsProvisioner{
		region: region,
		format: format,
		// @todo, Versioning turned on.
	}

	return provisioner, nil
}
