package kubernetes

const (
	KUBERNETES_LABEL_PODNAME                = "io.kubernetes.pod.name"
	KUBERNETES_LABEL_NAMESPACE              = "io.kubernetes.pod.namespace"
	KUBERNETES_LABEL_PODUID                 = "io.kubernetes.pod.uid"
	KUBERNETES_LABEL_IMAGE                  = "org.opencontainers.image.base.name"
	KUBERNETES_LABEL_CONTAINER_NAME         = "io.kubernetes.container.name"
	KUBERNETES_LABEL_CONTAINER_RESTARTCOUNT = "annotation.io.kubernetes.container.restartCount"
	KUBERNETES_LABEL_CONTAINER_PORTS        = "annotation.io.kubernetes.container.ports"
	KUBERNETES_ANNOTATION_CONTAINER_PORTS   = "io.kubernetes.container.ports"
)
