// Copyright 2017 Mirantis
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
	"errors"
	"fmt"
	"log"
	"reflect"

	"k8s.io/client-go/kubernetes/typed/extensions/v1beta1"
	extbeta1 "k8s.io/client-go/pkg/apis/extensions/v1beta1"

	"github.com/Mirantis/k8s-AppController/pkg/client"
	"github.com/Mirantis/k8s-AppController/pkg/interfaces"
	"github.com/Mirantis/k8s-AppController/pkg/report"
)

// Deployment is wrapper for K8s Deployment object
type Deployment struct {
	Base
	Deployment *extbeta1.Deployment
	Client     v1beta1.DeploymentInterface
}

func deploymentKey(name string) string {
	return "deployment/" + name
}

func deploymentStatus(deployment *extbeta1.Deployment) (interfaces.ResourceStatus, error) {
	if deployment.Status.UpdatedReplicas >= *deployment.Spec.Replicas && deployment.Status.AvailableReplicas >= *deployment.Spec.Replicas {
		return interfaces.ResourceReady, nil
	}
	return interfaces.ResourceNotReady, nil
}

// Key return Deployment key
func (d Deployment) Key() string {
	return deploymentKey(d.Deployment.Name)
}

// Status returns Deployment status. interfaces.ResourceReady means that its dependencies can be created
func (d Deployment) Status(meta map[string]string) (interfaces.ResourceStatus, error) {
	deployment, err := d.Client.Get(d.Deployment.Name)
	if err != nil {
		return interfaces.ResourceError, err
	}

	if !d.EqualToDefinition(deployment) {
		return interfaces.ResourceWaitingForUpgrade, fmt.Errorf(string(interfaces.ResourceWaitingForUpgrade))
	}

	return deploymentStatus(deployment)
}

func (d Deployment) EqualToDefinition(deployment interface{}) bool {
	dep := deployment.(*extbeta1.Deployment)

	return reflect.DeepEqual(dep.ObjectMeta, d.Deployment.ObjectMeta) && reflect.DeepEqual(dep.Spec, d.Deployment.Spec)
}

// Create looks for Deployment in K8s and creates it if not present
func (d Deployment) Create() error {
	log.Println("Looking for deployment", d.Deployment.Name)
	status, err := d.Status(nil)

	if err == nil {
		log.Printf("Found deployment %s, status: %s", d.Deployment.Name, status)
		log.Println("Skipping creation of deployment", d.Deployment.Name)
	}
	log.Println("Creating deployment", d.Deployment.Name)
	d.Deployment, err = d.Client.Create(d.Deployment)
	return err
}

// Delete deletes Deployment from the cluster
func (d Deployment) Delete() error {
	return d.Client.Delete(d.Deployment.Name, nil)
}

// NameMatches gets resource definition and a name and checks if
// the Deployment part of resource definition has matching name.
func (d Deployment) NameMatches(def client.ResourceDefinition, name string) bool {
	return def.Deployment != nil && def.Deployment.Name == name
}

// New returns new Deployment based on resource definition
func (d Deployment) New(def client.ResourceDefinition, c client.Interface) interfaces.Resource {
	//TODO: add ResDef to Base object in all resources
	return NewDeployment(def, c.Deployments())
}

// NewExisting returns new ExistingDeployment based on resource definition
func (d Deployment) NewExisting(name string, c client.Interface) interfaces.Resource {
	return NewExistingDeployment(name, c.Deployments())
}

// NewDeployment is a constructor
func NewDeployment(def client.ResourceDefinition, client v1beta1.DeploymentInterface) interfaces.Resource {
	return report.SimpleReporter{
		BaseResource: Deployment{
			Base: Base{
				Definition: def,
				meta:       def.Meta,
			},
			Deployment: def.Deployment,
			Client:     client,
		},
	}
}

// ExistingDeployment is a wrapper for K8s Deployment object which is deployed on a cluster before AppController
type ExistingDeployment struct {
	Base
	Name   string
	Client v1beta1.DeploymentInterface
}

// UpdateMeta does nothing at the moment
func (d ExistingDeployment) UpdateMeta(meta map[string]string) error {
	return nil
}

// Key returns Deployment name
func (d ExistingDeployment) Key() string {
	return deploymentKey(d.Name)
}

// Status returns Deployment status. interfaces.ResourceReady means that its dependencies can be created
func (d ExistingDeployment) Status(meta map[string]string) (interfaces.ResourceStatus, error) {
	deployment, err := d.Client.Get(d.Name)
	if err != nil {
		return interfaces.ResourceError, err
	}
	return deploymentStatus(deployment)
}

// Create looks for existing Deployment and returns error if there is no such Deployment
func (d ExistingDeployment) Create() error {
	log.Println("Looking for deployment", d.Name)
	status, err := d.Status(nil)

	if err == nil {
		log.Printf("Found deployment %s, status: %s", d.Name, status)
		return nil
	}

	log.Fatalf("Deployment %s not found", d.Name)
	return errors.New("Deployment not found")
}

// Delete deletes Deployment from the cluster
func (d ExistingDeployment) Delete() error {
	return d.Client.Delete(d.Name, nil)
}

// NewExistingDeployment is a constructor
func NewExistingDeployment(name string, client v1beta1.DeploymentInterface) interfaces.Resource {
	return report.SimpleReporter{BaseResource: ExistingDeployment{Name: name, Client: client}}
}
