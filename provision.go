package main

import (
	"bytes"
	"text/template"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/golang/glog"
	"github.com/kubernetes-incubator/external-storage/lib/controller"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// @todo, Find an upstream const for this.
const BucketAlreadyExists = 409

// Provision creates a storage asset and returns a PV object representing it.
func (p *riofsProvisioner) Provision(options controller.VolumeOptions) (*v1.PersistentVolume, error) {
	// This is a consistent naming pattern for provisioning our S3 bucket.
	name, err := formatName(p.format, options)
	if err != nil {
		return nil, err
	}

	glog.Infof("Provisioning bucket: %s", name)

	// Check if the bucket exists, if it does not, create the bucket if it does not exist.
	svc := s3.New(session.New(&aws.Config{Region: aws.String(p.region)}))
	params := &s3.CreateBucketInput{
		Bucket: aws.String(name),
	}
	_, err = svc.CreateBucket(params)
	if err != nil {
		if reqErr, ok := err.(awserr.RequestFailure); ok {
			if reqErr.StatusCode() != BucketAlreadyExists {
				return nil, err
			}
		}
	}

	// Turn on bucket versioning.
	_, err = svc.PutBucketVersioning(&s3.PutBucketVersioningInput{
		Bucket: aws.String(name),
		VersioningConfiguration: &s3.VersioningConfiguration{
			Status: aws.String(s3.BucketVersioningStatusEnabled),
		},
	})
	if err != nil {
		return nil, err
	}

	glog.Infof("Responding with persistent volume spec: %s", name)

	pv := &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			// The name of the bucket we are creating, is the same name we want to use
			// when mounting.
			Name: name,
		},
		Spec: v1.PersistentVolumeSpec{
			// PersistentVolumeReclaimPolicy, AccessModes and Capacity are required fields.
			PersistentVolumeReclaimPolicy: v1.PersistentVolumeReclaimRetain,
			AccessModes:                   options.PVC.Spec.AccessModes,
			Capacity: v1.ResourceList{
				// AWS S3 is UNLIMITED!.
				v1.ResourceName(v1.ResourceStorage): resource.MustParse("8.0E"),
			},
			PersistentVolumeSource: v1.PersistentVolumeSource{
				FlexVolume: &v1.FlexVolumeSource{
					Driver: "pnx/volume",
					FSType: "fuse",
					Options: map[string]string{
						"Name": name,
					},
				},
			},
		},
	}

	return pv, nil
}

// Helper function for building hostname.
func formatName(format string, options controller.VolumeOptions) (string, error) {
	var formatted bytes.Buffer

	t := template.Must(template.New("name").Parse(format))

	err := t.Execute(&formatted, options)
	if err != nil {
		return "", err
	}

	return formatted.String(), nil
}
