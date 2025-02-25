/*
Copyright 2021 The Kubernetes Authors.

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

package v1alpha4

import (
	clusterv1 "github.com/openshift/hypershift/api/v1alpha1/thirdparty/clusterapi/api/v1alpha4"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// ClusterFinalizer allows ReconcileAWSCluster to clean up AWS resources associated with AWSCluster before
	// removing it from the apiserver.
	ClusterFinalizer = "awscluster.infrastructure.cluster.x-k8s.io"

	// AWSClusterControllerIdentityName is the name of the AWSClusterControllerIdentity singleton
	AWSClusterControllerIdentityName = "default"
)

// AWSClusterSpec defines the desired state of AWSCluster
type AWSClusterSpec struct {
	// NetworkSpec encapsulates all things related to AWS network.
	NetworkSpec NetworkSpec `json:"networkSpec,omitempty"`

	// The AWS Region the cluster lives in.
	Region string `json:"region,omitempty"`

	// SSHKeyName is the name of the ssh key to attach to the bastion host. Valid values are empty string (do not use SSH keys), a valid SSH key name, or omitted (use the default SSH key name)
	// +optional
	SSHKeyName *string `json:"sshKeyName,omitempty"`

	// ControlPlaneEndpoint represents the endpoint used to communicate with the control plane.
	// +optional
	ControlPlaneEndpoint clusterv1.APIEndpoint `json:"controlPlaneEndpoint"`

	// AdditionalTags is an optional set of tags to add to AWS resources managed by the AWS provider, in addition to the
	// ones added by default.
	// +optional
	AdditionalTags Tags `json:"additionalTags,omitempty"`

	// ControlPlaneLoadBalancer is optional configuration for customizing control plane behavior.
	// +optional
	ControlPlaneLoadBalancer *AWSLoadBalancerSpec `json:"controlPlaneLoadBalancer,omitempty"`

	// ImageLookupFormat is the AMI naming format to look up machine images when
	// a machine does not specify an AMI. When set, this will be used for all
	// cluster machines unless a machine specifies a different ImageLookupOrg.
	// Supports substitutions for {{.BaseOS}} and {{.K8sVersion}} with the base
	// OS and kubernetes version, respectively. The BaseOS will be the value in
	// ImageLookupBaseOS or ubuntu (the default), and the kubernetes version as
	// defined by the packages produced by kubernetes/release without v as a
	// prefix: 1.13.0, 1.12.5-mybuild.1, or 1.17.3. For example, the default
	// image format of capa-ami-{{.BaseOS}}-?{{.K8sVersion}}-* will end up
	// searching for AMIs that match the pattern capa-ami-ubuntu-?1.18.0-* for a
	// Machine that is targeting kubernetes v1.18.0 and the ubuntu base OS. See
	// also: https://golang.org/pkg/text/template/
	// +optional
	ImageLookupFormat string `json:"imageLookupFormat,omitempty"`

	// ImageLookupOrg is the AWS Organization ID to look up machine images when a
	// machine does not specify an AMI. When set, this will be used for all
	// cluster machines unless a machine specifies a different ImageLookupOrg.
	// +optional
	ImageLookupOrg string `json:"imageLookupOrg,omitempty"`

	// ImageLookupBaseOS is the name of the base operating system used to look
	// up machine images when a machine does not specify an AMI. When set, this
	// will be used for all cluster machines unless a machine specifies a
	// different ImageLookupBaseOS.
	ImageLookupBaseOS string `json:"imageLookupBaseOS,omitempty"`

	// Bastion contains options to configure the bastion host.
	// +optional
	Bastion Bastion `json:"bastion"`

	// IdentityRef is a reference to a identity to be used when reconciling this cluster
	// +optional
	IdentityRef *AWSIdentityReference `json:"identityRef,omitempty"`
}

type AWSIdentityKind string

var (
	// ControllerIdentityKind defines identity reference kind as AWSClusterControllerIdentity
	ControllerIdentityKind = AWSIdentityKind("AWSClusterControllerIdentity")

	// ClusterRoleIdentityKind defines identity reference kind as AWSClusterRoleIdentity
	ClusterRoleIdentityKind = AWSIdentityKind("AWSClusterRoleIdentity")

	// ClusterStaticIdentityKind defines identity reference kind as AWSClusterStaticIdentity
	ClusterStaticIdentityKind = AWSIdentityKind("AWSClusterStaticIdentity")
)

// AWSIdentityReference specifies a identity.
type AWSIdentityReference struct {
	// Name of the identity.
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// Kind of the identity.
	// +kubebuilder:validation:Enum=AWSClusterControllerIdentity;AWSClusterRoleIdentity;AWSClusterStaticIdentity
	Kind AWSIdentityKind `json:"kind"`
}

type Bastion struct {
	// Enabled allows this provider to create a bastion host instance
	// with a public ip to access the VPC private network.
	// +optional
	Enabled bool `json:"enabled"`

	// DisableIngressRules will ensure there are no Ingress rules in the bastion host's security group.
	// Requires AllowedCIDRBlocks to be empty.
	// +optional
	DisableIngressRules bool `json:"disableIngressRules,omitempty"`

	// AllowedCIDRBlocks is a list of CIDR blocks allowed to access the bastion host.
	// They are set as ingress rules for the Bastion host's Security Group (defaults to 0.0.0.0/0).
	// +optional
	AllowedCIDRBlocks []string `json:"allowedCIDRBlocks,omitempty"`

	// InstanceType will use the specified instance type for the bastion. If not specified,
	// Cluster API Provider AWS will use t3.micro for all regions except us-east-1, where t2.micro
	// will be the default.
	InstanceType string `json:"instanceType,omitempty"`

	// AMI will use the specified AMI to boot the bastion. If not specified,
	// the AMI will default to one picked out in public space.
	// +optional
	AMI string `json:"ami,omitempty"`
}

// AWSLoadBalancerSpec defines the desired state of an AWS load balancer
type AWSLoadBalancerSpec struct {
	// Scheme sets the scheme of the load balancer (defaults to Internet-facing)
	// +kubebuilder:default=Internet-facing
	// +kubebuilder:validation:Enum=Internet-facing;internal
	// +optional
	Scheme *ClassicELBScheme `json:"scheme,omitempty"`

	// CrossZoneLoadBalancing enables the classic ELB cross availability zone balancing.
	//
	// With cross-zone load balancing, each load balancer node for your Classic Load Balancer
	// distributes requests evenly across the registered instances in all enabled Availability Zones.
	// If cross-zone load balancing is disabled, each load balancer node distributes requests evenly across
	// the registered instances in its Availability Zone only.
	//
	// Defaults to false.
	// +optional
	CrossZoneLoadBalancing bool `json:"crossZoneLoadBalancing"`

	// Subnets sets the subnets that should be applied to the control plane load balancer (defaults to discovered subnets for managed VPCs or an empty set for unmanaged VPCs)
	// +optional
	Subnets []string `json:"subnets,omitempty"`

	// AdditionalSecurityGroups sets the security groups used by the load balancer. Expected to be security group IDs.
	// This is optional - if not provided new security groups will be created for the load balancer
	// +optional
	AdditionalSecurityGroups []string `json:"additionalSecurityGroups,omitempty"`
}

// AWSClusterStatus defines the observed state of AWSCluster
type AWSClusterStatus struct {
	// +kubebuilder:default=false
	Ready          bool                     `json:"ready"`
	Network        Network                  `json:"network,omitempty"`
	FailureDomains clusterv1.FailureDomains `json:"failureDomains,omitempty"`
	Bastion        *Instance                `json:"bastion,omitempty"`
	Conditions     clusterv1.Conditions     `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=awsclusters,scope=Namespaced,categories=cluster-api
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Cluster",type="string",JSONPath=".metadata.labels.cluster\\.x-k8s\\.io/cluster-name",description="Cluster to which this AWSCluster belongs"
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.ready",description="Cluster infrastructure is ready for EC2 instances"
// +kubebuilder:printcolumn:name="VPC",type="string",JSONPath=".spec.networkSpec.vpc.id",description="AWS VPC the cluster is using"
// +kubebuilder:printcolumn:name="Endpoint",type="string",JSONPath=".status.apiEndpoints[0]",description="API Endpoint",priority=1
// +kubebuilder:printcolumn:name="Bastion IP",type="string",JSONPath=".status.bastion.publicIp",description="Bastion IP address for breakglass access"

// AWSCluster is the Schema for the awsclusters API
type AWSCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AWSClusterSpec   `json:"spec,omitempty"`
	Status AWSClusterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// AWSClusterList contains a list of AWSCluster
type AWSClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AWSCluster `json:"items"`
}

func (r *AWSCluster) GetConditions() clusterv1.Conditions {
	return r.Status.Conditions
}

func (r *AWSCluster) SetConditions(conditions clusterv1.Conditions) {
	r.Status.Conditions = conditions
}

func init() {
	SchemeBuilder.Register(&AWSCluster{}, &AWSClusterList{})
}
