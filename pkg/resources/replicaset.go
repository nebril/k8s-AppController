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

	"k8s.io/client-go/kubernetes/typed/extensions/v1beta1"
	extbeta1 "k8s.io/client-go/pkg/apis/extensions/v1beta1"

	"github.com/Mirantis/k8s-AppController/pkg/client"
	"github.com/Mirantis/k8s-AppController/pkg/interfaces"
	"github.com/Mirantis/k8s-AppController/pkg/report"
)

const SuccessFactorKey = "success_factor"

type ReplicaSet struct {
	Base
	ReplicaSet *extbeta1.ReplicaSet
	Client     v1beta1.ReplicaSetInterface
}

func replicaSetStatus(rs *extbeta1.ReplicaSet, meta map[string]string) (interfaces.ResourceStatus, error) {

	successFactor, err := getPercentage(SuccessFactorKey, meta)
	if err != nil {
		return interfaces.ResourceError, err
	}

	if rs.Status.Replicas*100 < *rs.Spec.Replicas*successFactor {
		return interfaces.ResourceNotReady, nil
	}

	return interfaces.ResourceReady, nil
}

func replicaSetReport(r v1beta1.ReplicaSetInterface, name string, meta map[string]string) interfaces.DependencyReport {
	rs, err := r.Get(name)
	if err != nil {
		return report.ErrorReport(name, err)
	}
	successFactor, err := getPercentage(SuccessFactorKey, meta)
	if err != nil {
		return report.ErrorReport(name, err)
	}
	percentage := (*rs.Spec.Replicas * 100 / rs.Status.Replicas)
	message := fmt.Sprintf(
		"%d of %d replicas up (%d %%, needed %d%%)",
		rs.Status.Replicas,
		rs.Spec.Replicas,
		percentage,
		successFactor,
	)
	if percentage >= successFactor {
		return interfaces.DependencyReport{
			Dependency: name,
			Blocks:     false,
			Percentage: int(percentage),
			Needed:     int(successFactor),
			Message:    message,
		}
	}
	return interfaces.DependencyReport{
		Dependency: name,
		Blocks:     false,
		Percentage: int(percentage),
		Needed:     int(successFactor),
		Message:    message,
	}
}

func replicaSetKey(name string) string {
	return "replicaset/" + name
}

func (r ReplicaSet) Key() string {
	return replicaSetKey(r.ReplicaSet.Name)
}

func (r ReplicaSet) Create() error {
	if err := checkExistence(r); err != nil {
		log.Println("Creating ", r.Key())
		r.ReplicaSet, err = r.Client.Create(r.ReplicaSet)
		return err
	}
	return nil
}

// Delete deletes ReplicaSet from the cluster
func (r ReplicaSet) Delete() error {
	return r.Client.Delete(r.ReplicaSet.Name, nil)
}

// Status returns ReplicaSet status based on provided meta.
func (r ReplicaSet) Status(meta map[string]string) (interfaces.ResourceStatus, error) {
	rs, err := r.Client.Get(r.ReplicaSet.Name)
	if err != nil {
		return interfaces.ResourceError, err
	}

	if !r.EqualToDefinition(rs) {
		return interfaces.ResourceWaitingForUpgrade, fmt.Errorf(string(interfaces.ResourceWaitingForUpgrade))
	}
	return replicaSetStatus(rs, meta)
}

// EqualToDefinition checks if definition in object is compatible with provided object
func (r ReplicaSet) EqualToDefinition(replicaSetiface interface{}) bool {
	replicaSet := replicaSetiface.(*extbeta1.ReplicaSet)

	return reflect.DeepEqual(replicaSet.ObjectMeta, r.ReplicaSet.ObjectMeta) && reflect.DeepEqual(replicaSet.Spec, r.ReplicaSet.Spec)
}

// NameMatches gets resource definition and a name and checks if
// the ReplicaSet part of resource definition has matching name.
func (r ReplicaSet) NameMatches(def client.ResourceDefinition, name string) bool {
	return def.ReplicaSet != nil && def.ReplicaSet.Name == name
}

// New returns new ReplicaSet based on resource definition
func (r ReplicaSet) New(def client.ResourceDefinition, c client.Interface) interfaces.Resource {
	return NewReplicaSet(def, c.ReplicaSets())
}

// NewExisting returns new ExistingReplicaSet based on resource definition
func (r ReplicaSet) NewExisting(name string, c client.Interface) interfaces.Resource {
	return NewExistingReplicaSet(name, c.ReplicaSets())
}

// GetDependencyReport returns a DependencyReport for this replicaset
func (r ReplicaSet) GetDependencyReport(meta map[string]string) interfaces.DependencyReport {
	return replicaSetReport(r.Client, r.ReplicaSet.Name, meta)
}

// StatusIsCacheable returns false if meta contains SuccessFactorKey
func (r ReplicaSet) StatusIsCacheable(meta map[string]string) bool {
	_, ok := meta[SuccessFactorKey]
	return !ok
}

func NewReplicaSet(def client.ResourceDefinition, client v1beta1.ReplicaSetInterface) ReplicaSet {
	return ReplicaSet{
		Base: Base{
			Definition: def,
			meta:       def.Meta,
		},
		ReplicaSet: def.ReplicaSet,
		Client:     client,
	}
}

type ExistingReplicaSet struct {
	Base
	Name   string
	Client v1beta1.ReplicaSetInterface
}

func (r ExistingReplicaSet) Key() string {
	return replicaSetKey(r.Name)
}

func (r ExistingReplicaSet) Create() error {
	return createExistingResource(r)
}

// Status returns ReplicaSet status based on provided meta.
func (r ExistingReplicaSet) Status(meta map[string]string) (interfaces.ResourceStatus, error) {
	rs, err := r.Client.Get(r.Name)
	if err != nil {
		return interfaces.ResourceError, err
	}
	return replicaSetStatus(rs, meta)
}

// Delete deletes ReplicaSet from the cluster
func (r ExistingReplicaSet) Delete() error {
	return r.Client.Delete(r.Name, nil)
}

func NewExistingReplicaSet(name string, client v1beta1.ReplicaSetInterface) ExistingReplicaSet {
	return ExistingReplicaSet{Name: name, Client: client}
}

// GetDependencyReport returns a DependencyReport for this replicaset
func (r ExistingReplicaSet) GetDependencyReport(meta map[string]string) interfaces.DependencyReport {
	return replicaSetReport(r.Client, r.Name, meta)
}

// StatusIsCacheable returns false if meta contains SuccessFactorKey
func (r ExistingReplicaSet) StatusIsCacheable(meta map[string]string) bool {
	_, ok := meta[SuccessFactorKey]
	return !ok
}
