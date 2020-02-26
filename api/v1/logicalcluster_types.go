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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// LogicalClusterSpec defines the desired state of LogicalCluster
type LogicalClusterSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of LogicalCluster. Edit LogicalCluster_types.go to remove/update
	Name  string   `json:"name"`
	Nodes []string `json:"nodes"`
}

// LogicalClusterStatus defines the observed state of LogicalCluster
type LogicalClusterStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	CurrentLabeledNodeNum  int `json:"currentLabeledNodeNum,omitempty"`
	ExpectedLabeledNodeNum int `json:"expectedLabeledNodeNum,omitempty"`
}

// +kubebuilder:subresource:status
// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster,shortName=lc;lcs
// +kubebuilder:printcolumn:JSONPath=".status.currentLabeledNodeNum",name=Ready Nodes,type=string
// +kubebuilder:printcolumn:JSONPath=".status.expectedLabeledNodeNum",name=Expected Nodes,type=string
// +kubebuilder:printcolumn:JSONPath=".metadata.creationTimestamp",name=Age,type=date

// LogicalCluster is the Schema for the logicalclusters API
type LogicalCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LogicalClusterSpec   `json:"spec,omitempty"`
	Status LogicalClusterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// LogicalClusterList contains a list of LogicalCluster
type LogicalClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LogicalCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&LogicalCluster{}, &LogicalClusterList{})
}
