//go:build !ignore_autogenerated
// +build !ignore_autogenerated

// Code generated by controller-gen. DO NOT EDIT.

package v1

import (
	corev1 "k8s.io/api/core/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CertManagerValues) DeepCopyInto(out *CertManagerValues) {
	*out = *in
	if in.Tolerations != nil {
		in, out := &in.Tolerations, &out.Tolerations
		*out = make([]corev1.Toleration, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.NodeSelector != nil {
		in, out := &in.NodeSelector, &out.NodeSelector
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.PodLabels != nil {
		in, out := &in.PodLabels, &out.PodLabels
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	in.ClusterIssuer.DeepCopyInto(&out.ClusterIssuer)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CertManagerValues.
func (in *CertManagerValues) DeepCopy() *CertManagerValues {
	if in == nil {
		return nil
	}
	out := new(CertManagerValues)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Cloudflare) DeepCopyInto(out *Cloudflare) {
	*out = *in
	out.SecretKeyRef = in.SecretKeyRef
	if in.DnsNames != nil {
		in, out := &in.DnsNames, &out.DnsNames
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Cloudflare.
func (in *Cloudflare) DeepCopy() *Cloudflare {
	if in == nil {
		return nil
	}
	out := new(Cloudflare)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterIssuer) DeepCopyInto(out *ClusterIssuer) {
	*out = *in
	if in.Cloudflare != nil {
		in, out := &in.Cloudflare, &out.Cloudflare
		*out = new(Cloudflare)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterIssuer.
func (in *ClusterIssuer) DeepCopy() *ClusterIssuer {
	if in == nil {
		return nil
	}
	out := new(ClusterIssuer)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GithubReleaseArtifacts) DeepCopyInto(out *GithubReleaseArtifacts) {
	*out = *in
	if in.Artifacts != nil {
		in, out := &in.Artifacts, &out.Artifacts
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	out.TokenSecret = in.TokenSecret
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GithubReleaseArtifacts.
func (in *GithubReleaseArtifacts) DeepCopy() *GithubReleaseArtifacts {
	if in == nil {
		return nil
	}
	out := new(GithubReleaseArtifacts)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *IngressValues) DeepCopyInto(out *IngressValues) {
	*out = *in
	if in.PodLabels != nil {
		in, out := &in.PodLabels, &out.PodLabels
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	in.Resources.DeepCopyInto(&out.Resources)
	if in.NodeSelector != nil {
		in, out := &in.NodeSelector, &out.NodeSelector
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Tolerations != nil {
		in, out := &in.Tolerations, &out.Tolerations
		*out = make([]corev1.Toleration, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IngressValues.
func (in *IngressValues) DeepCopy() *IngressValues {
	if in == nil {
		return nil
	}
	out := new(IngressValues)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KloudliteCreds) DeepCopyInto(out *KloudliteCreds) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KloudliteCreds.
func (in *KloudliteCreds) DeepCopy() *KloudliteCreds {
	if in == nil {
		return nil
	}
	out := new(KloudliteCreds)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KloudliteDnsApi) DeepCopyInto(out *KloudliteDnsApi) {
	*out = *in
	out.BasicAuthCreds = in.BasicAuthCreds
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KloudliteDnsApi.
func (in *KloudliteDnsApi) DeepCopy() *KloudliteDnsApi {
	if in == nil {
		return nil
	}
	out := new(KloudliteDnsApi)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *LokiValues) DeepCopyInto(out *LokiValues) {
	*out = *in
	out.S3 = in.S3
	in.Resources.DeepCopyInto(&out.Resources)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new LokiValues.
func (in *LokiValues) DeepCopy() *LokiValues {
	if in == nil {
		return nil
	}
	out := new(LokiValues)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ManagedCluster) DeepCopyInto(out *ManagedCluster) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ManagedCluster.
func (in *ManagedCluster) DeepCopy() *ManagedCluster {
	if in == nil {
		return nil
	}
	out := new(ManagedCluster)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ManagedCluster) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ManagedClusterList) DeepCopyInto(out *ManagedClusterList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ManagedCluster, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ManagedClusterList.
func (in *ManagedClusterList) DeepCopy() *ManagedClusterList {
	if in == nil {
		return nil
	}
	out := new(ManagedClusterList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ManagedClusterList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ManagedClusterSpec) DeepCopyInto(out *ManagedClusterSpec) {
	*out = *in
	if in.Domain != nil {
		in, out := &in.Domain, &out.Domain
		*out = new(string)
		**out = **in
	}
	out.KloudliteCreds = in.KloudliteCreds
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ManagedClusterSpec.
func (in *ManagedClusterSpec) DeepCopy() *ManagedClusterSpec {
	if in == nil {
		return nil
	}
	out := new(ManagedClusterSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NetworkingValues) DeepCopyInto(out *NetworkingValues) {
	*out = *in
	if in.DnsNames != nil {
		in, out := &in.DnsNames, &out.DnsNames
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NetworkingValues.
func (in *NetworkingValues) DeepCopy() *NetworkingValues {
	if in == nil {
		return nil
	}
	out := new(NetworkingValues)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Operators) DeepCopyInto(out *Operators) {
	*out = *in
	if in.Manifests != nil {
		in, out := &in.Manifests, &out.Manifests
		*out = make([]GithubReleaseArtifacts, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Operators.
func (in *Operators) DeepCopy() *Operators {
	if in == nil {
		return nil
	}
	out := new(Operators)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PrimaryCluster) DeepCopyInto(out *PrimaryCluster) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PrimaryCluster.
func (in *PrimaryCluster) DeepCopy() *PrimaryCluster {
	if in == nil {
		return nil
	}
	out := new(PrimaryCluster)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *PrimaryCluster) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PrimaryClusterList) DeepCopyInto(out *PrimaryClusterList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]PrimaryCluster, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PrimaryClusterList.
func (in *PrimaryClusterList) DeepCopy() *PrimaryClusterList {
	if in == nil {
		return nil
	}
	out := new(PrimaryClusterList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *PrimaryClusterList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PrimaryClusterSpec) DeepCopyInto(out *PrimaryClusterSpec) {
	*out = *in
	in.Networking.DeepCopyInto(&out.Networking)
	if in.NodeSelector != nil {
		in, out := &in.NodeSelector, &out.NodeSelector
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Tolerations != nil {
		in, out := &in.Tolerations, &out.Tolerations
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	out.StripeCreds = in.StripeCreds
	in.CloudflareCreds.DeepCopyInto(&out.CloudflareCreds)
	out.HarborAdminCreds = in.HarborAdminCreds
	out.WebhookAuthzCreds = in.WebhookAuthzCreds
	if in.ImgPullSecrets != nil {
		in, out := &in.ImgPullSecrets, &out.ImgPullSecrets
		*out = make([]SecretReference, len(*in))
		copy(*out, *in)
	}
	in.LokiValues.DeepCopyInto(&out.LokiValues)
	in.PrometheusValues.DeepCopyInto(&out.PrometheusValues)
	in.CertManagerValues.DeepCopyInto(&out.CertManagerValues)
	in.IngressValues.DeepCopyInto(&out.IngressValues)
	in.Operators.DeepCopyInto(&out.Operators)
	out.OAuthCreds = in.OAuthCreds
	in.RedpandaValues.DeepCopyInto(&out.RedpandaValues)
	if in.SharedConstants != nil {
		in, out := &in.SharedConstants, &out.SharedConstants
		*out = new(SharedConstants)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PrimaryClusterSpec.
func (in *PrimaryClusterSpec) DeepCopy() *PrimaryClusterSpec {
	if in == nil {
		return nil
	}
	out := new(PrimaryClusterSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PrometheusValues) DeepCopyInto(out *PrometheusValues) {
	*out = *in
	in.Resources.DeepCopyInto(&out.Resources)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PrometheusValues.
func (in *PrometheusValues) DeepCopy() *PrometheusValues {
	if in == nil {
		return nil
	}
	out := new(PrometheusValues)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RedpandaValues) DeepCopyInto(out *RedpandaValues) {
	*out = *in
	in.Resources.DeepCopyInto(&out.Resources)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RedpandaValues.
func (in *RedpandaValues) DeepCopy() *RedpandaValues {
	if in == nil {
		return nil
	}
	out := new(RedpandaValues)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *S3) DeepCopyInto(out *S3) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new S3.
func (in *S3) DeepCopy() *S3 {
	if in == nil {
		return nil
	}
	out := new(S3)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SecondaryCluster) DeepCopyInto(out *SecondaryCluster) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SecondaryCluster.
func (in *SecondaryCluster) DeepCopy() *SecondaryCluster {
	if in == nil {
		return nil
	}
	out := new(SecondaryCluster)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *SecondaryCluster) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SecondaryClusterList) DeepCopyInto(out *SecondaryClusterList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]SecondaryCluster, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SecondaryClusterList.
func (in *SecondaryClusterList) DeepCopy() *SecondaryClusterList {
	if in == nil {
		return nil
	}
	out := new(SecondaryClusterList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *SecondaryClusterList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SecondaryClusterSpec) DeepCopyInto(out *SecondaryClusterSpec) {
	*out = *in
	out.SharedConstants = in.SharedConstants
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SecondaryClusterSpec.
func (in *SecondaryClusterSpec) DeepCopy() *SecondaryClusterSpec {
	if in == nil {
		return nil
	}
	out := new(SecondaryClusterSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SecondarySharedConstants) DeepCopyInto(out *SecondarySharedConstants) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SecondarySharedConstants.
func (in *SecondarySharedConstants) DeepCopy() *SecondarySharedConstants {
	if in == nil {
		return nil
	}
	out := new(SecondarySharedConstants)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SecretKeyReference) DeepCopyInto(out *SecretKeyReference) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SecretKeyReference.
func (in *SecretKeyReference) DeepCopy() *SecretKeyReference {
	if in == nil {
		return nil
	}
	out := new(SecretKeyReference)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SecretReference) DeepCopyInto(out *SecretReference) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SecretReference.
func (in *SecretReference) DeepCopy() *SecretReference {
	if in == nil {
		return nil
	}
	out := new(SecretReference)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SharedConstants) DeepCopyInto(out *SharedConstants) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SharedConstants.
func (in *SharedConstants) DeepCopy() *SharedConstants {
	if in == nil {
		return nil
	}
	out := new(SharedConstants)
	in.DeepCopyInto(out)
	return out
}
