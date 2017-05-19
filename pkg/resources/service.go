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
	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apis/apps/v1beta1"
	"k8s.io/client-go/pkg/labels"

	"github.com/Mirantis/k8s-AppController/pkg/client"
	"github.com/Mirantis/k8s-AppController/pkg/interfaces"
	"github.com/Mirantis/k8s-AppController/pkg/report"
)

type Service struct {
	Base
	Service   *v1.Service
	Client    corev1.ServiceInterface
	APIClient client.Interface
}

func serviceStatus(service *v1.Service, apiClient client.Interface) (interfaces.ResourceStatus, error) {
	log.Printf("Checking service status for selector %v", service.Spec.Selector)
	for k, v := range service.Spec.Selector {
		stringSelector := fmt.Sprintf("%s=%s", k, v)
		log.Printf("Checking status for %s", stringSelector)
		selector, err := labels.Parse(stringSelector)
		if err != nil {
			return interfaces.ResourceError, err
		}

		options := v1.ListOptions{LabelSelector: selector.String()}

		pods, err := apiClient.Pods().List(options)
		if err != nil {
			return interfaces.ResourceError, err
		}
		jobs, err := apiClient.Jobs().List(options)
		if err != nil {
			return interfaces.ResourceError, err
		}
		replicasets, err := apiClient.ReplicaSets().List(options)
		if err != nil {
			return interfaces.ResourceError, err
		}
		resources := make([]interfaces.BaseResource, 0, len(pods.Items)+len(jobs.Items)+len(replicasets.Items))
		for _, pod := range pods.Items {
			def := MakeDefinition(&pod)
			resources = append(resources, NewPod(def, apiClient.Pods()))
		}
		for _, j := range jobs.Items {
			jobDef := MakeDefinition(&j)
			resources = append(resources, NewJob(jobDef, apiClient.Jobs()))
		}
		for _, r := range replicasets.Items {
			rsDef := MakeDefinition(&r)
			resources = append(resources, NewReplicaSet(rsDef, apiClient.ReplicaSets()))
		}
		if apiClient.IsEnabled(v1beta1.SchemeGroupVersion) {
			statefulsets, err := apiClient.StatefulSets().List(options)
			if err != nil {
				return interfaces.ResourceError, err
			}
			for _, ps := range statefulsets.Items {
				def := MakeDefinition(&ps)
				resources = append(resources, NewStatefulSet(def, apiClient))
			}
		} else {
			petsets, err := apiClient.PetSets().List(api.ListOptions{LabelSelector: selector})
			if err != nil {
				return interfaces.ResourceError, err
			}
			for _, ps := range petsets.Items {
				petSetDef := MakeDefinition(&ps)
				resources = append(resources, NewPetSet(petSetDef, apiClient))
			}
		}
		status, err := resourceListStatus(resources)
		if status != interfaces.ResourceReady || err != nil {
			return status, err
		}
	}

	return interfaces.ResourceReady, nil
}

func serviceKey(name string) string {
	return "service/" + name
}

func (s Service) Key() string {
	return serviceKey(s.Service.Name)
}

func (s Service) Create() error {
	if err := checkExistence(s); err != nil {
		log.Println("Creating ", s.Key())
		s.Service, err = s.Client.Create(s.Service)
		return err
	}
	return nil
}

// Delete deletes Service from the cluster
func (s Service) Delete() error {
	return s.Client.Delete(s.Service.Name, nil)
}

// Status returns Service Status. It is based on the status of all objects which match the service selector. If all of them are ready, the Service is considered ready.
func (s Service) Status(meta map[string]string) (interfaces.ResourceStatus, error) {
	service, err := s.Client.Get(s.Service.Name)
	if err != nil {
		return interfaces.ResourceError, err
	}

	if !s.EqualToDefinition(service) {
		return interfaces.ResourceWaitingForUpgrade, fmt.Errorf(string(interfaces.ResourceWaitingForUpgrade))
	}
	return serviceStatus(service, s.APIClient)
}

// EqualToDefinition checks if definition in object is compatible with provided object
func (s Service) EqualToDefinition(serviceiface interface{}) bool {
	service := serviceiface.(*v1.Service)

	return reflect.DeepEqual(service.ObjectMeta, s.Service.ObjectMeta) && reflect.DeepEqual(service.Spec, s.Service.Spec)
}

// NameMatches gets resource definition and a name and checks if
// the Service part of resource definition has matching name.
func (s Service) NameMatches(def client.ResourceDefinition, name string) bool {
	return def.Service != nil && def.Service.Name == name
}

// New returns new Service based on resource definition
func (s Service) New(def client.ResourceDefinition, c client.Interface) interfaces.Resource {
	return NewService(def, c)
}

// NewExisting returns new ExistingService based on resource definition
func (s Service) NewExisting(name string, c client.Interface) interfaces.Resource {
	return NewExistingService(name, c.Services())
}

// NewService is Service constructor. Needs apiClient for service status checks
func NewService(def client.ResourceDefinition, apiClient client.Interface) interfaces.Resource {
	return report.SimpleReporter{
		BaseResource: Service{
			Base: Base{
				Definition: def,
				meta:       def.Meta,
			},
			Service:   def.Service,
			Client:    apiClient.Services(),
			APIClient: apiClient,
		},
	}
}

// StatusIsCacheable for service always returns false since the status must be
// checked on each request and not be cached
func (s Service) StatusIsCacheable(meta map[string]string) bool {
	return false
}

type ExistingService struct {
	Base
	Name      string
	Client    corev1.ServiceInterface
	APIClient client.Interface
}

func (s ExistingService) Key() string {
	return serviceKey(s.Name)
}

func (s ExistingService) Create() error {
	return createExistingResource(s)
}

// Status returns Service Status. It is based on the status of all objects which match the service selector. If all of them are ready, the Service is considered ready.
func (s ExistingService) Status(meta map[string]string) (interfaces.ResourceStatus, error) {
	service, err := s.Client.Get(s.Name)

	if err != nil {
		return interfaces.ResourceError, err
	}
	return serviceStatus(service, s.APIClient)
}

// Delete deletes Service from the cluster
func (s ExistingService) Delete() error {
	return s.Client.Delete(s.Name, nil)
}

// StatusIsCacheable for service always returns false since the status must be
// checked on each request and not be cached
func (s ExistingService) StatusIsCacheable(meta map[string]string) bool {
	return false
}

func NewExistingService(name string, client corev1.ServiceInterface) interfaces.Resource {
	return report.SimpleReporter{BaseResource: ExistingService{Name: name, Client: client}}
}
