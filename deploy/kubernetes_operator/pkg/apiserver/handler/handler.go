package handler

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/bytedance/fedlearner/deploy/kubernetes_operator/pkg/apis/fedlearner.k8s.io/v1alpha1"
	crdclientset "github.com/bytedance/fedlearner/deploy/kubernetes_operator/pkg/client/clientset/versioned"
	crdinformers "github.com/bytedance/fedlearner/deploy/kubernetes_operator/pkg/client/informers/externalversions"
)

// Handler .
type Handler struct {
	kubeClient *clientset.Clientset
	crdClient  *crdclientset.Clientset

	informerFactory    informers.SharedInformerFactory
	crdInformerFactory crdinformers.SharedInformerFactory
}

// NewHandler returns a new handler.
func NewHandler(
	kubeClient *clientset.Clientset,
	crdClientset *crdclientset.Clientset,
	informerFactory informers.SharedInformerFactory,
	crdInformerFactory crdinformers.SharedInformerFactory,
) *Handler {
	return &Handler{
		kubeClient:         kubeClient,
		crdClient:          crdClientset,
		informerFactory:    informerFactory,
		crdInformerFactory: crdInformerFactory,
	}
}

// Run .
func (h *Handler) Run(stopCh <-chan struct{}) error {
	if !cache.WaitForCacheSync(
		stopCh,
		h.informerFactory.Core().V1().Pods().Informer().HasSynced,
		h.informerFactory.Core().V1().Namespaces().Informer().HasSynced,
		h.crdInformerFactory.Fedlearner().V1alpha1().FLApps().Informer().HasSynced,
	) {
		return fmt.Errorf("timed out waiting for cache to sync")
	}

	return nil
}

// ListNamespaces .
func (h *Handler) ListNamespaces(c *gin.Context) {
	namespaces, err := h.informerFactory.Core().V1().Namespaces().Lister().List(labels.Everything())
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(200, gin.H{
		"namespaces": namespaces,
	})
}

// ListPods returns pods in namespace.
func (h *Handler) ListPods(c *gin.Context) {
	namespace := c.Param("namespace")

	pods, err := h.informerFactory.Core().V1().Pods().Lister().Pods(namespace).List(labels.Everything())
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(200, gin.H{
		"pods": pods,
	})
}

// GetPod returns a pod with name.
func (h *Handler) GetPod(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")

	pod, err := h.informerFactory.Core().V1().Pods().Lister().Pods(namespace).Get(name)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(200, gin.H{
		"pod": pod,
	})
}

// ListPodEvents returns pods' events.
func (h *Handler) ListPodEvents(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")

	events, err := h.kubeClient.CoreV1().Events(namespace).List(metav1.ListOptions{
		FieldSelector: fields.OneTermEqualSelector("involvedObject.name", name).String(),
	})
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(200, gin.H{
		"events": events,
	})
}

// GetFLApp .
func (h *Handler) GetFLApp(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")

	flapp, err := h.crdClient.FedlearnerV1alpha1().FLApps(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(200, gin.H{
		"flapp": flapp,
	})
}

// ListFLAppPods .
func (h *Handler) ListFLAppPods(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")

	pods, err := h.informerFactory.Core().V1().Pods().Lister().Pods(namespace).List(labels.Set{
		"app-name": name,
	}.AsSelector())
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(200, gin.H{
		"pods": pods,
	})
}

// ListFLApps .
func (h *Handler) ListFLApps(c *gin.Context) {
	namespace := c.Param("namespace")

	flapps, err := h.crdClient.FedlearnerV1alpha1().FLApps(namespace).List(metav1.ListOptions{})
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(200, gin.H{
		"flapps": flapps,
	})
}

// CreateFLApp .
func (h *Handler) CreateFLApp(c *gin.Context) {
	namespace := c.Param("namespace")

	flapp := &v1alpha1.FLApp{}
	if err := c.BindJSON(&flapp); err != nil {
		h.handleError(c, err)
		return
	}

	newFlapp, err := h.crdClient.FedlearnerV1alpha1().FLApps(namespace).Create(flapp)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(200, gin.H{
		"flapp": newFlapp,
	})
}

// DeleteFLApp .
func (h *Handler) DeleteFLApp(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")

	if err := h.crdClient.FedlearnerV1alpha1().FLApps(namespace).Delete(name, &metav1.DeleteOptions{}); err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(200, gin.H{})
}

func (h *Handler) handleError(c *gin.Context, err error) {
	statusCode := 500
	if errors.IsNotFound(err) {
		statusCode = 404
	}

	c.JSON(statusCode, gin.H{
		"error": err.Error(),
	})

}