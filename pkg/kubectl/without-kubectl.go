package kubectl

import (
	"bytes"
	"context"
	"io"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	apiLabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/apimachinery/pkg/types"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
)

type YAMLClient struct {
	k8sClient     *kubernetes.Clientset
	dynamicClient dynamic.Interface
	restMapper    meta.RESTMapper
}

func (yc *YAMLClient) ApplyYAML(ctx context.Context, yamls ...[]byte) error {
	jYamls := bytes.Join(yamls, []byte("\n---\n"))
	decoder := yamlutil.NewYAMLOrJSONDecoder(bytes.NewReader(jYamls), 100)
	for {
		var rawObj runtime.RawExtension
		if err := decoder.Decode(&rawObj); err != nil {
			if err != io.EOF {
				return err
			}
			break
		}

		if rawObj.Raw == nil {
			continue
		}

		obj, gvk, err := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme).Decode(rawObj.Raw, nil, nil)
		if err != nil {
			return err
		}
		unstructuredMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
		if err != nil {
			// log.Fatal(err)
			return err
		}

		unstructuredObj := &unstructured.Unstructured{Object: unstructuredMap}

		var dri dynamic.ResourceInterface

		mapping, err := yc.restMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			// log.Fatal(err)
			return err
		}
		if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
			if unstructuredObj.GetNamespace() == "" {
				unstructuredObj.SetNamespace("default")
			}
			dri = yc.dynamicClient.Resource(mapping.Resource).Namespace(unstructuredObj.GetNamespace())
		} else {
			dri = yc.dynamicClient.Resource(mapping.Resource)
		}

		if _, err := dri.Patch(
			context.Background(),
			unstructuredObj.GetName(),
			types.MergePatchType,
			rawObj.Raw,
			metav1.PatchOptions{},
		); err != nil {
			if errors.IsNotFound(err) {
				if _, err := dri.Create(ctx, unstructuredObj, metav1.CreateOptions{}); err != nil {
					// log.Fatal(err)
					return err
				}
				continue
			}
			// log.Fatal(err)
			return err
		}
	}
	return nil
}

func (yc *YAMLClient) DeleteYAML(ctx context.Context, yamls ...[]byte) error {
	jYamls := bytes.Join(yamls, []byte("\n---\n"))
	decoder := yamlutil.NewYAMLOrJSONDecoder(bytes.NewReader(jYamls), 100)
	for {
		var rawObj runtime.RawExtension
		if err := decoder.Decode(&rawObj); err != nil {
			if err != io.EOF {
				return err
			}
			break
		}

		if rawObj.Raw == nil {
			continue
		}

		obj, gvk, err := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme).Decode(rawObj.Raw, nil, nil)
		if err != nil {
			return err
		}
		unstructuredMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
		if err != nil {
			// log.Fatal(err)
			return err
		}

		unstructuredObj := &unstructured.Unstructured{Object: unstructuredMap}

		var dri dynamic.ResourceInterface

		mapping, err := yc.restMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			// log.Fatal(err)
			return err
		}
		if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
			if unstructuredObj.GetNamespace() == "" {
				unstructuredObj.SetNamespace("default")
			}
			dri = yc.dynamicClient.Resource(mapping.Resource).Namespace(unstructuredObj.GetNamespace())
		} else {
			dri = yc.dynamicClient.Resource(mapping.Resource)
		}

		if err := dri.Delete(ctx, unstructuredObj.GetName(), metav1.DeleteOptions{}); err != nil {
			if errors.IsNotFound(err) {
				return nil
			}
			return err
		}
	}

	return nil
}

type Restartable string

const (
	Deployment  Restartable = "deployment"
	StatefulSet Restartable = "statefulset"
)

func (yc *YAMLClient) RolloutRestart(ctx context.Context, kind Restartable, namespace string, labels map[string]string) error {
	switch kind {
	case Deployment:
		{
			dl, err := yc.k8sClient.AppsV1().Deployments(namespace).List(
				ctx, metav1.ListOptions{
					LabelSelector: apiLabels.FormatLabels(labels),
				},
			)
			if err != nil {
				return err
			}
			for _, d := range dl.Items {
				if d.Annotations == nil {
					d.Annotations = map[string]string{}
				}
				d.Annotations["kubectl.kubernetes.io/restartedAt"] = time.Now().Format(time.RFC3339)
				yc.k8sClient.AppsV1().Deployments(namespace).Update(ctx, &d, metav1.UpdateOptions{})
			}
		}
	case StatefulSet:
		{
			sl, err := yc.k8sClient.AppsV1().StatefulSets(namespace).List(
				ctx, metav1.ListOptions{
					LabelSelector: apiLabels.FormatLabels(labels),
				},
			)
			if err != nil {
				return err
			}
			for _, d := range sl.Items {
				if d.Annotations == nil {
					d.Annotations = map[string]string{}
				}
				d.Annotations["kubectl.kubernetes.io/restartedAt"] = time.Now().Format(time.RFC3339)
				yc.k8sClient.AppsV1().StatefulSets(namespace).Update(ctx, &d, metav1.UpdateOptions{})
			}
		}
	}

	return nil
}

func NewYAMLClient(config *rest.Config) (*YAMLClient, error) {
	c, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	dc, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	gr, err := restmapper.GetAPIGroupResources(c.Discovery())
	if err != nil {
		// log.Fatal(err)
		return nil, err
	}

	mapper := restmapper.NewDiscoveryRESTMapper(gr)

	return &YAMLClient{
		k8sClient:     c,
		dynamicClient: dc,
		restMapper:    mapper,
	}, nil
}

func NewYAMLClientOrDie(config *rest.Config) *YAMLClient {
	client, err := NewYAMLClient(config)
	if err != nil {
		panic(err)
	}
	return client
}