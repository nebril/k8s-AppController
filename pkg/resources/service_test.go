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
	"testing"

	"fmt"

	"github.com/Mirantis/k8s-AppController/pkg/interfaces"
	"github.com/Mirantis/k8s-AppController/pkg/mocks"
)

// TestCheckServiceStatusReady checks if the service status check is fine for healthy service
func TestCheckServiceStatusReady(t *testing.T) {
	svc := mocks.MakeService("success")
	c := mocks.NewClient(svc)

	def := MakeDefinition(svc)
	tested := NewService(def, c)

	status, err := tested.Status(nil)

	if err != nil {
		t.Errorf("%s", err)
	}

	if status != interfaces.ResourceReady {
		t.Errorf("service should be `ready`, is `%s` instead", status)
	}
}

// TestCheckServiceStatusPodNotReady tests if service which selects failed pods is not ready
func TestCheckServiceStatusPodNotReady(t *testing.T) {
	svcName := "failedpod"
	podName := "error"
	svc := mocks.MakeService(svcName)

	pod := mocks.MakePod(podName)
	pod.Labels = svc.Spec.Selector

	c := mocks.NewClient(svc, pod)

	def := MakeDefinition(svc)
	tested := NewService(def, c)

	status, err := tested.Status(nil)

	if err == nil {
		t.Fatal("Error should be returned, got nil")
	}
	expectedError := fmt.Sprintf("Resource pod/%v is not ready", pod.Name)
	if err.Error() != expectedError {
		t.Errorf("Expected `%s` as error, got `%s`", expectedError, err.Error())
	}

	if status != interfaces.ResourceNotReady {
		t.Errorf("service should be `not ready`, is `%s` instead", status)
	}
}

// TestCheckServiceStatusJobNotReady tests if service which selects failed jobs is not ready
func TestCheckServiceStatusJobNotReady(t *testing.T) {
	svcName := "failedjob"
	jobName := "error"
	svc := mocks.MakeService(svcName)

	job := mocks.MakeJob(jobName)
	job.Labels = svc.Spec.Selector

	c := mocks.NewClient(svc, job)

	def := MakeDefinition(svc)
	tested := NewService(def, c)

	status, err := tested.Status(nil)

	if err == nil {
		t.Error("Error should be returned, got nil")
	}

	expectedError := fmt.Sprintf("Resource job/%v is not ready", job.Name)
	if err.Error() != expectedError {
		t.Errorf("Expected `%s` as error, got `%s`", expectedError, err.Error())
	}

	if status != interfaces.ResourceNotReady {
		t.Errorf("service should be `not ready`, is `%s` instead", status)
	}
}

// TestCheckServiceStatusReplicaSetNotReady tests if service which selects failed replicasets is not ready
func TestCheckServiceStatusReplicaSetNotReady(t *testing.T) {
	svcName := "failedrc"
	rsName := "fail"
	svc := mocks.MakeService(svcName)

	rc := mocks.MakeReplicaSet(rsName)
	rc.Labels = svc.Spec.Selector

	c := mocks.NewClient(svc, rc)

	def := MakeDefinition(svc)
	tested := NewService(def, c)

	status, err := tested.Status(nil)

	if err == nil {
		t.Error("Error should be returned, got nil")
	}

	expectedError := fmt.Sprintf("Resource replicaset/%v is not ready", rc.Name)
	if err.Error() != expectedError {
		t.Errorf("Expected `%s` as error, got `%s`", expectedError, err.Error())
	}

	if status != interfaces.ResourceNotReady {
		t.Errorf("service should be `not ready`, is `%s` instead", status)
	}
}

// TestServiceUpgraded tests status behaviour with resource definition differing from object in cluster
func TestServiceUpgraded(t *testing.T) {
	name := "notfail"

	client := mocks.NewClient(mocks.MakeService(name))

	def := MakeDefinition(mocks.MakeService(name))
	//Make definition differ from client version
	def.Service.ObjectMeta.Labels = map[string]string{
		"trolo": "lolo",
	}
	tested := NewService(def, client)

	status, err := tested.Status(nil)

	if err == nil {
		t.Error("Error not found, expected error")
	}
	if status != interfaces.ResourceWaitingForUpgrade {
		t.Errorf("Status should be `waiting for upgrade`, is `%s` instead.", status)
	}
}
