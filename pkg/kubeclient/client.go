package kubeclient

import (
	"context"
	"fmt"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
)

func createClient() (*kubernetes.Clientset, error) {
  config, err := rest.InClusterConfig()
  if err != nil {
    return nil, fmt.Errorf("could not connect to kubernetes: %v", err)
  }

  client, err := kubernetes.NewForConfig(config)
  if err != nil {
    return nil, fmt.Errorf("could not create kubernetes config: %v", err)
  }

  return client, nil
}

func UpdateWebhookCA(webhookName string, ca []byte) (error) {
  client, err := createClient()
  if err != nil {
    return err
  }

  webhook, err := client.AdmissionregistrationV1().
    MutatingWebhookConfigurations().
    Get(context.Background(), webhookName, metav1.GetOptions{})
  if err != nil {
    return fmt.Errorf("could not find resource: %v", err)
  }

  webhook.Webhooks[0].ClientConfig.CABundle = ca

  patch, err := json.Marshal(webhook)
  if err != nil {
    return err
  }

  // fmt.Printf("Applied patch:\n%s\n", string(patch))

  opts := metav1.PatchOptions{
      TypeMeta: metav1.TypeMeta{Kind: "admissionregistration.k8s.io", APIVersion: "v1"},
      FieldManager: "xcxc.dev/csi-secret-injector",
      // FieldValidation: "Warn",
    }
  _, err = client.AdmissionregistrationV1().MutatingWebhookConfigurations().Patch(context.Background(), webhookName, types.StrategicMergePatchType, patch, opts)
  if err != nil {
    return err
  }

  // fmt.Printf("Got result:\n%v", res)

  return nil
}
