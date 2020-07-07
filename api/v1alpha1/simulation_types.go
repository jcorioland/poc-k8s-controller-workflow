/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// SimulationSpec defines the desired state of Simulation
type SimulationSpec struct {
	// The simulation owner.
	Owner string `json:"owner"`

	// The command to execute
	Command string `json:"command"`

	// The name of the building block that is entry point
	EntryPoint string `json:"entryPoint"`

	// BuildingBlocks for the simulation
	BuildingBlocks []BuildingBlock `json:"buildingBlocks"`
}

// SimulationStatus defines the observed state of Simulation
type SimulationStatus struct {
	// SimulationState represents the state of the simulation
	SimulationState SimulationState `json:"simulationState"`
	// CreatedBlocks are the blocks created for the simulation
	CreatedBlocks []BuildingBlock `json:"createdBlocks"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Simulation is the Schema for the simulations API
type Simulation struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SimulationSpec   `json:"spec,omitempty"`
	Status SimulationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SimulationList contains a list of Simulation
type SimulationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Simulation `json:"items"`
}

// SimulationState represents the state of the simulation
type SimulationState string

const (
	// SimulationNotReady means the simulation is not ready to start (basically, waiting for building blocks to be up and running)
	SimulationNotReady SimulationState = "SimulationNotReady"
	// SimulationRunning means the simulation is currently running
	SimulationRunning SimulationState = "SimulationRunning"
	// SimulationCompleted means the simulation run is completed with success
	SimulationCompleted SimulationState = "SimulationCompleted"
	// SimulationFailed means the simulation run has failed
	SimulationFailed SimulationState = "SimulationFailed"
)

// BuildingBlock represents a simulation building block
type BuildingBlock struct {
	Name        string `json:"name"`
	DockerImage string `json:"dockerImage"`
	DockerTag   string `json:"dockerTag"`
	// +kubebuilder:validation:Optional
	Ready bool `json:"ready"`
	// +kubebuilder:validation:Optional
	Created bool `json:"created"`
	// +kubebuilder:validation:Optional
	Failed bool `json:"failed"`
	// +kubebuilder:validation:Optional
	Error string `json:"error"`
}

func init() {
	SchemeBuilder.Register(&Simulation{}, &SimulationList{})
}
