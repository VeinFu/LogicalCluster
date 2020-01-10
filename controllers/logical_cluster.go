package controllers

import (
	"encoding/json"
	"errors"
	//"flag"
	"fmt"
	"log"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	//"k8s.io/client-go/tools/clientcmd"
)

type logicalCluster struct {
	ClusterName string
	Hosts       []string
}

type hostList []string

// 创建逻辑集群
func CreateCluster(clientset *kubernetes.Clientset, clusterName string, hostsInCluster []string) error {
	if err := AddOrUpdateNodeLabel(clientset, hostsInCluster, clusterName); err != nil {
		return err
	}
	return nil
}

// 获取所有逻辑集群及其主机列表
func ListClusters(clientset *kubernetes.Clientset, clustersInfo *string) error {
	var LogicalClusters []logicalCluster
	labelSelectorRequirement := []metav1.LabelSelectorRequirement{
		{Key: "logical-cluster", Operator: metav1.LabelSelectorOpExists}}
	labelSelector := metav1.LabelSelector{MatchExpressions: labelSelectorRequirement}
	labelSelectorMap, _ := metav1.LabelSelectorAsMap(&labelSelector)
	listOptions := metav1.ListOptions{LabelSelector: labels.SelectorFromSet(labelSelectorMap).String()}
	ContainerNodes, err := clientset.CoreV1().Nodes().List(listOptions)
	if err != nil {
		fmt.Println(err.Error())
		return err
	}
	if len(ContainerNodes.Items) == 0 {
		err := errors.New("no logical cluster resource exists")
		return err
	}

	var clusterNames []string
	for _, Host := range ContainerNodes.Items {
		logicalClusterName := Host.Labels["logical-cluster"]
		if ret := isInSlice(logicalClusterName, clusterNames); !ret {
			clusterNames = append(clusterNames, logicalClusterName)
			hostsInCluster, err := GetNodesFromCluster(clientset, logicalClusterName)
			if err != nil {
				return err
			}
			LogicalClusters = append(LogicalClusters, logicalCluster{
				ClusterName: logicalClusterName, Hosts: hostsInCluster})
		}
		continue
	}
	logicalClustersBytes, _ := json.Marshal(LogicalClusters)
	*clustersInfo = string(logicalClustersBytes)
	return nil
}

// 获取单个逻辑集群及其主机列表
func GetCluster(clientset *kubernetes.Clientset, clusterName string, clusterInfo *string) error {
	hostsInCluster, err := GetNodesFromCluster(clientset, clusterName)
	if err != nil {
		return err
	}
	if len(hostsInCluster) == 0 {
		err := errors.New("该逻辑群不存在")
		return err
	}
	singleLogicalClusterBytes, _ := json.Marshal(
		logicalCluster{ClusterName: clusterName, Hosts: hostsInCluster})
	*clusterInfo = string(singleLogicalClusterBytes)
	return nil
}

// 删除某个逻辑集群
func DeleteCluster(clientset *kubernetes.Clientset, clusterName string) error {
	hostsLabelRemoved, err := GetNodesFromCluster(clientset, clusterName)
	if err != nil {
		return err
	}
	if err := RemoveLabelFromNode(clientset, hostsLabelRemoved); err != nil {
		return err
	}
	return nil
}

// 更新逻辑集群名称
func UpdateClusterName(clientset *kubernetes.Clientset, clusterName string, hostsInCluster []string, clusterInfo *string) error {
	if err := AddOrUpdateNodeLabel(clientset, hostsInCluster, clusterName); err != nil {
		return err
	}
	if err := GetCluster(clientset, clusterName, clusterInfo); err != nil {
		return err
	}
	return nil
}

// 逻辑集群的扩缩容
func ScaleCluster(clientset *kubernetes.Clientset, clusterName string, nodesList []string, isScale bool, clusterInfo *string) error {
	if isScale { // 扩容
		if err := AddOrUpdateNodeLabel(clientset, nodesList, clusterName); err != nil {
			return err
		}
	} else { // 缩容
		if err := RemoveLabelFromNode(clientset, nodesList); err != nil {
			return err
		}
	}
	if err := GetCluster(clientset, clusterName, clusterInfo); err != nil {
		return err
	}
	return nil
}

// 节点添加、更新标签，也是添加一个逻辑集群的关键入口
func AddOrUpdateNodeLabel(clientset *kubernetes.Clientset, hostsUpdated []string, labelName string) error {
	patchData := map[string]interface{}{"metadata": map[string]map[string]string{
		"labels": {"logical-cluster": labelName}}}
	patchDataBytes, _ := json.Marshal(patchData)
	for _, targetNode := range hostsUpdated {
		_, err := clientset.CoreV1().Nodes().Patch(targetNode, types.StrategicMergePatchType, patchDataBytes)
		if err != nil {
			return err
		}
	}
	return nil
}

// 移除某个特定节点标签
func RemoveLabelFromNode(clientset *kubernetes.Clientset, targetNodes []string) error {
	patchDataMap := make(map[string]string)
	patchDataMap["op"] = "remove"
	patchDataMap["path"] = "/metadata/labels/logical-cluster"
	var patchData []map[string]string
	patchData = append(patchData, patchDataMap)
	patchDataBytes, _ := json.Marshal(patchData)
	for _, targetNode := range targetNodes {
		if _, err := clientset.CoreV1().Nodes().Patch(targetNode, types.JSONPatchType, patchDataBytes); err != nil {
			return err
		}
	}
	return nil
}

// 查找具有特定标签的所有主机
func GetNodesFromCluster(clientset *kubernetes.Clientset, clusterName string) (hostList, error) {
	var hosts []string
	labelMap := make(map[string]string)
	labelMap["logical-cluster"] = clusterName
	labelSelector := metav1.LabelSelector{MatchLabels: labelMap}
	labelSelectorMap, _ := metav1.LabelSelectorAsMap(&labelSelector)
	listOptions := metav1.ListOptions{LabelSelector: labels.SelectorFromSet(labelSelectorMap).String()}
	ContainerNodes, err := clientset.CoreV1().Nodes().List(listOptions)
	if err != nil {
		log.Println(err.Error())
		return hosts, err
	}

	for _, node := range ContainerNodes.Items {
		hosts = append(hosts, node.Name)
	}

	return hosts, nil
}

// 判断切片里某个值是否存在
func isInSlice(data string, slice []string) bool {
	for _, val := range slice {
		if val == data {
			return true
		}
		continue
	}
	return false
}
