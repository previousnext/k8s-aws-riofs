package main

import (
	"k8s.io/api/core/v1"
)

// Delete removes the storage asset that was created by Provision represented
// by the given PV.
// @todo, Tag Bucket as "ready for removal"
// @todo, Tag Bucket with a date to show how old it is.
func (p *riofsProvisioner) Delete(volume *v1.PersistentVolume) error {
	return nil
}
