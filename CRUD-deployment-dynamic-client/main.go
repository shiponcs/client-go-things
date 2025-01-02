package main

import (
	"context"
	"fmt"
	"github.com/shiponcs/client-go-things/utils"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/client-go/util/retry"
)

func main() {
	config, err := clientcmd.BuildConfigFromFlags("", homedir.HomeDir()+"/.kube/config")
	if err != nil {
		panic(err.Error())
	}
	client, err := dynamic.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	deploymentRes := schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}

	deployment := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"metadata": map[string]interface{}{
				"name": "dynamic-client-speaking",
			},
			"spec": map[string]interface{}{
				"replicas": 2,
				"selector": map[string]interface{}{
					"matchLabels": map[string]interface{}{
						"app": "dynamic-client-speaking",
					},
				},
				"template": map[string]interface{}{
					"metadata": map[string]interface{}{
						"labels": map[string]interface{}{
							"app": "dynamic-client-speaking",
						},
					},
					"spec": map[string]interface{}{
						"containers": []map[string]interface{}{
							{
								"name":  "web",
								"image": "nginx:1.12",
								"ports": []map[string]interface{}{
									{
										"containerPort": 80,
										"name":          "http",
										"protocol":      "TCP",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	// create Deployment
	fmt.Println("Create deployment")
	result, err := client.Resource(deploymentRes).Namespace(v1.NamespaceDefault).Create(context.TODO(), deployment, metav1.CreateOptions{})
	if err != nil {
		panic(err.Error())
	}
	fmt.Printf("Create deployment successfully, result is %v\n", result.GetName())

	// Update Deployment
	utils.Prompt()
	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		result, getErr := client.Resource(deploymentRes).Namespace(v1.NamespaceDefault).Get(context.TODO(), "dynamic-client-speaking", metav1.GetOptions{})
		if getErr != nil {
			panic(fmt.Errorf("Failed to get resource: %v", getErr))
		} else {
			fmt.Printf("Retrieved deployment successfully, result is %v\n", result)
		}

		// update replica
		if err := unstructured.SetNestedField(result.Object, int64(1), "spec", "replicas"); err != nil {
			panic(fmt.Errorf("Failed to set spec replicas: %v", err))
		}

		// extract containers and update image
		containers, found, err := unstructured.NestedSlice(result.Object, "spec", "template", "spec", "containers")
		if err != nil || !found || containers == nil {
			panic(fmt.Errorf("Failed to get spec template containers: %v", err))
		}
		if err := unstructured.SetNestedField(containers[0].(map[string]interface{}), "nginx:1.13", "image"); err != nil {
			panic(fmt.Errorf("Failed to set nginx image: %v", err))
		}

		// extract ports and update container port
		ports, found, err := unstructured.NestedSlice(containers[0].(map[string]interface{}), "ports")
		if err != nil || !found || ports == nil {
			panic(fmt.Errorf("Failed to get spec ports: %v", err))
		} else {
			fmt.Printf("ports: %v\n", ports)
		}
		if err := unstructured.SetNestedField(ports[0].(map[string]interface{}), int64(88), "containerPort"); err != nil {
			panic(fmt.Errorf("Failed to set ports: %v", err))
		} else {
			fmt.Printf("ports after update: %v\n", ports)
		}
		// alternative:
		//port := ports[0].(map[string]interface{})
		//port["containerPort"] = interface{}(int64(88))
		//ports[0] = port

		if err := unstructured.SetNestedField(containers[0].(map[string]interface{}), ports, "ports"); err != nil {
			panic(fmt.Errorf("Failed to set containers: %v", err))
		} else {
			fmt.Printf("containers after update: %v\n", containers)
		}
		if err := unstructured.SetNestedField(result.Object, containers, "spec", "template", "spec", "containers"); err != nil {
			panic(fmt.Errorf("Failed to set result: %v", err))
		} else {
			fmt.Printf("result after update: %v\n", containers)
		}

		_, updateErr := client.Resource(deploymentRes).Namespace(v1.NamespaceDefault).Update(context.TODO(), result, metav1.UpdateOptions{})
		return updateErr
	})
	if retryErr != nil {
		panic(fmt.Errorf("retry error, %v", retryErr))
	}
	fmt.Printf("update deployment successfully\n")
}
