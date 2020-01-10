## 写在前面

本项目是`kubernetes CRD`的一个简单实践项目，基于`kubebuilder`工具，具体实现的功能如下：
根据K8s节点的标签来划分逻辑集群，方便不同维度的调度管理。

## Demo流程

这里本地开发环境安装了GO开发环境和`kubectl`工具，并且做了远端K8s集群环境KUBECONFIG环境变量配置，
这样本地开发环境的`kubectl`就可以访问远端的K8s集群环境了。

以下都是在本地开发环境做的操作：

```bash
# 下载源码到本地
cd $GOPATH/src/
git clone git@github.com:VeinFu/example.git

# 编译
cd example/
make && make install

# 制作镜像和上传镜像
make docker-build docker-push IMG=<image_repo>/logical-cluster-crd-img:v1

# 如果远端K8s集群的机器没法访问镜像仓库，可以先把镜像打成tar包，然后通过scp上传至K8s集群，之后导入
docker save -0 logical_cluster.tar <image_repo>/logical-cluster-crd-img:v1
scp logical-cluster.tar root@<remote_ip>:/root
ssh root@<remote_ip> docker load --input /root/logical_cluster.tar

# 部署controller
make deploy IMG=<image_repo>/logical-cluster-crd-img:v1
```

如此部署算是基本完成，下面是验证环节：
```bash
cd example/
# 创建逻辑集群
kubectl create -f config/samples/scheduler-mgr_v1_logicalcluster.yaml
kubectl get lcs
kubectl get nodes --show-labels

# 删除逻辑集群
kubectl delete -f config/samples/scheduler-mgr_v1_logicalcluster.yaml
kubectl get lcs
kubectl get nodes --show-labels

# 也可以重新编辑config/samples/scheduler-mgr_v1_logicalcluster.yaml文件来实现集群更新，比如逻辑集群名称、对集群进行扩缩容等
kubectl apply -f config/samples/scheduler-mgr_v1_logicalcluster.yaml
kubectl get lcs
kubectl get nodes --show-labels 
```