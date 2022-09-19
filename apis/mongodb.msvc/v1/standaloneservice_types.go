package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ct "operators.kloudlite.io/apis/common-types"
	"operators.kloudlite.io/lib/constants"
	rApi "operators.kloudlite.io/lib/operator"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// StandaloneServiceSpec defines the desired state of StandaloneService
type StandaloneServiceSpec struct {
	CloudProvider ct.CloudProvider  `json:"cloudProvider"`
	NodeSelector  map[string]string `json:"nodeSelector,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=1
	ReplicaCount int          `json:"replicaCount,omitempty"`
	Storage      ct.Storage   `json:"storage"`
	Resources    ct.Resources `json:"resources"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// StandaloneService is the Schema for the standaloneservices API
type StandaloneService struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   StandaloneServiceSpec `json:"spec,omitempty"`
	Status rApi.Status           `json:"status,omitempty"`
}

func (s *StandaloneService) GetStatus() *rApi.Status {
	return &s.Status
}

func (s *StandaloneService) GetEnsuredLabels() map[string]string {
	return map[string]string{constants.MsvcNameKey: s.Name}
}

func (s *StandaloneService) GetEnsuredAnnotations() map[string]string {
	return map[string]string{}
}

// +kubebuilder:object:root=true

// StandaloneServiceList contains a list of StandaloneService
type StandaloneServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []StandaloneService `json:"items"`
}

func init() {
	SchemeBuilder.Register(&StandaloneService{}, &StandaloneServiceList{})
}