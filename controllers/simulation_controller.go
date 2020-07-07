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

package controllers

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	simulationworkflowv1alpha1 "poc-simulation-workflow.io/api/v1alpha1"
)

// SimulationReconciler reconciles a Simulation object
type SimulationReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=simulationworkflow.poc-simulation-workflow.io,resources=simulations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=simulationworkflow.poc-simulation-workflow.io,resources=simulations/status,verbs=get;update;patch

func (r *SimulationReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("simulation", req.NamespacedName)

	// your logic here
	var simulation simulationworkflowv1alpha1.Simulation
	if err := r.Get(ctx, req.NamespacedName, &simulation); err != nil {
		log.Error(err, "Unable to fetch simulation from request")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	log.Info(fmt.Sprintf("Reconcile called for simulation %s", simulation.Name))

	// simulation state empty means it has just been submitted
	if simulation.Status.SimulationState == "" {
		for _, buildingBlock := range simulation.Spec.BuildingBlocks {
			if !buildingBlock.Created {
				// create pod
				_, err := createPod(buildingBlock.Name, req.Namespace, buildingBlock.DockerImage, buildingBlock.DockerTag)
				if err != nil {
					buildingBlock.Error = err.Error()
					buildingBlock.Failed = true
				}
				buildingBlock.Created = true
				simulation.Status.CreatedBlocks = append(simulation.Status.CreatedBlocks, buildingBlock)
			}
		}

		simulation.Status.SimulationState = simulationworkflowv1alpha1.SimulationNotReady
	} else if simulation.Status.SimulationState == simulationworkflowv1alpha1.SimulationNotReady {
		allBlocksReady := true
		for _, buildingBlock := range simulation.Status.CreatedBlocks {
			if !buildingBlock.Ready {
				// check status
				pod, err := getPod(buildingBlock.Name, req.Namespace)
				if err != nil {
					allBlocksReady = false
					buildingBlock.Failed = true
					buildingBlock.Error = err.Error()
					simulation.Status.SimulationState = simulationworkflowv1alpha1.SimulationFailed
				}

				if pod.Status.Phase == v1.PodRunning {
					buildingBlock.Ready = true
				} else if pod.Status.Phase == v1.PodFailed {
					allBlocksReady = false
					buildingBlock.Failed = true
					buildingBlock.Error = pod.Status.Reason
					simulation.Status.SimulationState = simulationworkflowv1alpha1.SimulationFailed
				}
			}
		}

		if allBlocksReady {
			// todo: run the simulation and mark the simulation as running
			simulation.Status.SimulationState = simulationworkflowv1alpha1.SimulationRunning
		}
	} else if simulation.Status.SimulationState == simulationworkflowv1alpha1.SimulationRunning {
		// todo: wait until the simulation is completed or failed
	}

	// update the status
	if err := r.Status().Update(ctx, &simulation); err != nil {
		log.Error(err, "unable to update simulation status")
		return ctrl.Result{}, err
	}

	// if simulation is completed or failed, no need to reconcile anymore
	if simulation.Status.SimulationState == simulationworkflowv1alpha1.SimulationCompleted || simulation.Status.SimulationState == simulationworkflowv1alpha1.SimulationFailed {
		return ctrl.Result{}, nil
	}

	// requeue every 10 sec until it's completed or failed
	return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
}

func (r *SimulationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&simulationworkflowv1alpha1.Simulation{}).
		Complete(r)
}

func createPod(name string, namespace string, dockerImage string, dockerTag string) (*v1.Pod, error) {
	// retrieve in cluster k8s config
	config, err := loadConfig()
	if err != nil {
		return nil, err
	}

	// create a k8s client using the config
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	// get a pod interface
	podsInterface := clientset.CoreV1().Pods(namespace)

	// create a pod spec
	pod := &v1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:  name,
					Image: fmt.Sprintf("%s:%s", dockerImage, dockerTag),
				},
			},
		},
	}

	return podsInterface.Create(pod)
}

func getPod(name string, namespace string) (*v1.Pod, error) {
	// retrieve in cluster k8s config
	config, err := loadConfig()
	if err != nil {
		return nil, err
	}

	// create a k8s client using the config
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	// get a pod interface
	podsInterface := clientset.CoreV1().Pods(namespace)
	return podsInterface.Get(name, metav1.GetOptions{})
}

func loadConfig() (*rest.Config, error) {
	executionMode := os.Getenv("EXECUTION_MODE")
	if executionMode == ExecutionModeInCluster {
		return rest.InClusterConfig()
	} else if executionMode == ExecutionModeLocal {
		home := homeDir()
		kubeconfig := filepath.Join(home, ".kube", "config")
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}

	return nil, fmt.Errorf("EXECUTION_MODE env var is not set. Cannot load Local or InCluster kubernetes configuration")
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

const (
	// ExecutionModeLocal means the controller runs locally on a dev machine, for example.
	ExecutionModeLocal string = "Local"
	// ExecutionModeInCluster means the controller runs inside a kubernetes cluster
	ExecutionModeInCluster string = "InCluster"
)
