package main

import (
	"github.com/kubernetes-incubator/external-storage/lib/controller"
)

var _ controller.Provisioner = &riofsProvisioner{}

type riofsProvisioner struct {
	// Region to provision the new S3 bucket.
	region string

	// Formatting used to derive the name used in EFS.
	format string
}
