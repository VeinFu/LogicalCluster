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
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	schedulermgrv1 "LogicalCluster/api/v1"
)

// LogicalClusterReconciler reconciles a LogicalCluster object
type LogicalClusterReconciler struct {
	client.Client
	ExtraClient *kubernetes.Clientset
	Log         logr.Logger
	Scheme      *runtime.Scheme
}

// +kubebuilder:rbac:groups=scheduler-mgr.ucloud.io,resources=logicalclusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=scheduler-mgr.ucloud.io,resources=logicalclusters/status,verbs=get;update;patch

func (r *LogicalClusterReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("logicalcluster", req.NamespacedName)

	// your logic here
	logicalClusterIns := &schedulermgrv1.LogicalCluster{}
	if err := r.Get(ctx, req.NamespacedName, logicalClusterIns); err != nil {
		if errors.IsNotFound(err) {
			log.Info("can not find logical cluster resource")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// 设置终结器，删除逻辑集群
	lClusterFinalizerName := "clusterDelete"
	if logicalClusterIns.ObjectMeta.DeletionTimestamp.IsZero() {
		if !ContainsString(logicalClusterIns.ObjectMeta.Finalizers, lClusterFinalizerName, nil) {
			logicalClusterIns.ObjectMeta.Finalizers = append(logicalClusterIns.ObjectMeta.Finalizers, lClusterFinalizerName)
			if err := r.Update(ctx, logicalClusterIns); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		if ContainsString(logicalClusterIns.Finalizers, lClusterFinalizerName, nil) {
			if err := DeleteCluster(r.ExtraClient, logicalClusterIns.Name); err != nil {
				return ctrl.Result{}, err
			}
			logicalClusterIns.ObjectMeta.Finalizers = RemoveString(logicalClusterIns.ObjectMeta.Finalizers, lClusterFinalizerName, nil)
			if err := r.Update(ctx, logicalClusterIns); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}
	}

	// 同步逻辑集群的状态信息
	if err := r.syncLogicalClusterStatus(logicalClusterIns, log); err != nil {
		log.Info("sync LogicalCluster status failed")
		return ctrl.Result{}, err
	}

	// 逻辑集群协调主逻辑
	if err := r.reconcileLogicalClusterIns(logicalClusterIns, log); err != nil {
		log.Info("reconcile LogicalCluster error")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *LogicalClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&schedulermgrv1.LogicalCluster{}).
		Complete(r)
}

func (r *LogicalClusterReconciler) syncLogicalClusterStatus(logicalClusterIns *schedulermgrv1.LogicalCluster, log logr.Logger) error {
	if logicalClusterIns.Status.CurrentLabeledNodeNum != 0 && logicalClusterIns.Status.CurrentLabeledNodeNum == len(logicalClusterIns.Spec.Nodes) {
		log.Info("Labels of all nodes in cluster have updated")
		return nil
	}

	ctx := context.Background()

	newStatus, err := r.calcuteStatus(logicalClusterIns, log)
	if err != nil {
		return err
	}

	logicalClusterIns.Status = newStatus
	if err := r.Status().Update(ctx, logicalClusterIns); err != nil {
		return err
	}

	return nil
}

func (r *LogicalClusterReconciler) reconcileLogicalClusterIns(logicalClusterIns *schedulermgrv1.LogicalCluster, log logr.Logger) error {
	if logicalClusterIns.Status.CurrentLabeledNodeNum != 0 && logicalClusterIns.Status.CurrentLabeledNodeNum == logicalClusterIns.Status.ExpectedLabeledNodeNum {
		log.Info("Labels of all nodes in cluster have updated")
		return nil
	}

	nodeList, err := GetNodesFromCluster(r.ExtraClient, logicalClusterIns.Name)
	if err != nil {
		return err
	}
	var labelDealedNodes []string
	// 根据逻辑集群主机节点数目的添加/移除，相应的对节点进行标签的添加/移除
	if logicalClusterIns.Status.ExpectedLabeledNodeNum > logicalClusterIns.Status.CurrentLabeledNodeNum {
		log.Info("Some nodes label need to be added.")
		for _, node := range logicalClusterIns.Spec.Nodes {
			if !ContainsString(nodeList, node, nil) {
				labelDealedNodes = append(labelDealedNodes, node)
			}
			continue
		}

		if err := AddOrUpdateNodeLabel(r.ExtraClient, labelDealedNodes, logicalClusterIns.Name); err != nil {
			return err
		}
	} else if len(logicalClusterIns.Spec.Nodes) < logicalClusterIns.Status.CurrentLabeledNodeNum {
		log.Info("Some nodes label need to be removed.")
		for _, node := range nodeList {
			if !ContainsString(logicalClusterIns.Spec.Nodes, node, nil) {
				labelDealedNodes = append(labelDealedNodes, node)
			}
			continue
		}
		if err := RemoveLabelFromNode(r.ExtraClient, labelDealedNodes); err != nil {
			return err
		}
	}

	return nil
}

func (r *LogicalClusterReconciler) calcuteStatus(logicalClusterIns *schedulermgrv1.LogicalCluster, log logr.Logger) (schedulermgrv1.LogicalClusterStatus, error) {
	var nodeList []string

	newStatus := schedulermgrv1.LogicalClusterStatus{
		CurrentLabeledNodeNum:  len(nodeList),
		ExpectedLabeledNodeNum: len(logicalClusterIns.Spec.Nodes),
	}

	nodeList, err := GetNodesFromCluster(r.ExtraClient, logicalClusterIns.Name)
	if err != nil {
		return newStatus, err
	}

	newStatus.CurrentLabeledNodeNum = len(nodeList)

	return newStatus, nil
}
