// Copyright 2016 Mirantis
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package resources

import (
	"fmt"
	"log"
	"reflect"

	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/pkg/api/v1"

	"github.com/Mirantis/k8s-AppController/pkg/client"
	"github.com/Mirantis/k8s-AppController/pkg/interfaces"
	"github.com/Mirantis/k8s-AppController/pkg/report"
)

type PersistentVolumeClaim struct {
	Base
	PersistentVolumeClaim *v1.PersistentVolumeClaim
	Client                corev1.PersistentVolumeClaimInterface
}

func persistentVolumeClaimKey(name string) string {
	return "persistentvolumeclaim/" + name
}

func (p PersistentVolumeClaim) Key() string {
	return persistentVolumeClaimKey(p.PersistentVolumeClaim.Name)
}

func persistentVolumeClaimStatus(persistentVolumeClaim *v1.PersistentVolumeClaim) (interfaces.ResourceStatus, error) {
	if persistentVolumeClaim.Status.Phase == v1.ClaimBound {
		return interfaces.ResourceReady, nil
	}

	return interfaces.ResourceNotReady, nil
}

func (p PersistentVolumeClaim) Create() error {
	if err := checkExistence(p); err != nil {
		log.Println("Creating ", p.Key())
		p.PersistentVolumeClaim, err = p.Client.Create(p.PersistentVolumeClaim)
		return err
	}
	return nil
}

// Delete deletes persistentVolumeClaim from the cluster
func (p PersistentVolumeClaim) Delete() error {
	return p.Client.Delete(p.PersistentVolumeClaim.Name, &v1.DeleteOptions{})
}

// Status returns PVC status.
func (p PersistentVolumeClaim) Status(meta map[string]string) (interfaces.ResourceStatus, error) {
	pvc, err := p.Client.Get(p.PersistentVolumeClaim.Name)
	if err != nil {
		return interfaces.ResourceError, err
	}

	if !p.EqualToDefinition(pvc) {
		return interfaces.ResourceWaitingForUpgrade, fmt.Errorf(string(interfaces.ResourceWaitingForUpgrade))
	}

	return persistentVolumeClaimStatus(pvc)
}

// EqualToDefinition returns whether the resource has the same values as provided object
func (p PersistentVolumeClaim) EqualToDefinition(claim interface{}) bool {
	pvc := claim.(*v1.PersistentVolumeClaim)

	return reflect.DeepEqual(pvc.ObjectMeta, p.PersistentVolumeClaim.ObjectMeta) && reflect.DeepEqual(pvc.Spec, p.PersistentVolumeClaim.Spec)
}

// NameMatches gets resource definition and a name and checks if
// the PersistentVolumeClaim part of resource definition has matching name.
func (p PersistentVolumeClaim) NameMatches(def client.ResourceDefinition, name string) bool {
	return def.PersistentVolumeClaim != nil && def.PersistentVolumeClaim.Name == name
}

// New returns new PersistentVolumeClaim based on resource definition
func (p PersistentVolumeClaim) New(def client.ResourceDefinition, c client.Interface) interfaces.Resource {
	return NewPersistentVolumeClaim(def, c.PersistentVolumeClaims())
}

// NewExisting returns new ExistingPersistentVolumeClaim based on resource definition
func (p PersistentVolumeClaim) NewExisting(name string, c client.Interface) interfaces.Resource {
	return NewExistingPersistentVolumeClaim(name, c.PersistentVolumeClaims())
}

func NewPersistentVolumeClaim(def client.ResourceDefinition, client corev1.PersistentVolumeClaimInterface) interfaces.Resource {
	return report.SimpleReporter{
		BaseResource: PersistentVolumeClaim{
			Base: Base{
				Definition: def,
				meta:       def.Meta,
			},
			PersistentVolumeClaim: def.PersistentVolumeClaim,
			Client:                client,
		},
	}
}

type ExistingPersistentVolumeClaim struct {
	Base
	Name   string
	Client corev1.PersistentVolumeClaimInterface
}

func (p ExistingPersistentVolumeClaim) Key() string {
	return persistentVolumeClaimKey(p.Name)
}

func (p ExistingPersistentVolumeClaim) Create() error {
	return createExistingResource(p)
}

// Status returns PVC status.
func (p ExistingPersistentVolumeClaim) Status(meta map[string]string) (interfaces.ResourceStatus, error) {
	pvc, err := p.Client.Get(p.Name)
	if err != nil {
		return interfaces.ResourceError, err
	}

	return persistentVolumeClaimStatus(pvc)
}

// Delete deletes persistentVolumeClaim from the cluster
func (p ExistingPersistentVolumeClaim) Delete() error {
	return p.Client.Delete(p.Name, nil)
}

func NewExistingPersistentVolumeClaim(name string, client corev1.PersistentVolumeClaimInterface) interfaces.Resource {
	return report.SimpleReporter{BaseResource: ExistingPersistentVolumeClaim{Name: name, Client: client}}
}
