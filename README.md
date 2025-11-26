# Cassandra Multi-Region Kubernetes Deployment

This deployment creates a Cassandra cluster configured for multi-region operation with high availability and monitoring.

## Files Created

1. **namespace-rbac.yaml** - Namespace, ServiceAccount, and RBAC configuration
2. **cassandra-statefulset.yaml** - Main StatefulSet with services for Cassandra
3. **cassandra-config.yaml** - Configuration files for Cassandra cluster settings
4. **storage-classes.yaml** - Storage classes for different regions and performance tiers
5. **security-policies.yaml** - Network policies, secrets, and PodDisruptionBudget
6. **monitoring.yaml** - Prometheus monitoring and metrics export

## Deployment Steps

```bash
# 1. Create namespace and RBAC
kubectl apply -f namespace-rbac.yaml

# 2. Create storage classes
kubectl apply -f storage-classes.yaml

# 3. Create configuration
kubectl apply -f cassandra-config.yaml

# 4. Deploy security policies
kubectl apply -f security-policies.yaml

# 5. Deploy Cassandra cluster
kubectl apply -f cassandra-statefulset.yaml

# 6. Deploy monitoring
kubectl apply -f monitoring.yaml
```

## Multi-Region Configuration

- **Data Centers**: Automatically detected from node topology labels
- **Racks**: Configured per node/zone
- **Seed Nodes**: First 3 pods act as seeds
- **Replication**: Configure keyspace replication factor based on regions
- **Network**: GossipingPropertyFileSnitch for multi-region awareness

## Key Features

- **6 replicas** for high availability across regions
- **100Gi persistent storage** per node with fast SSD
- **Health checks** with readiness and liveness probes
- **Network policies** for security
- **Prometheus monitoring** with JMX metrics
- **PodDisruptionBudget** to maintain availability
- **Automatic failover** and data replication

## Access

```bash
# Connect to Cassandra
kubectl exec -it cassandra-0 -n cassandra -- cqlsh

# Check cluster status
kubectl exec -it cassandra-0 -n cassandra -- nodetool status

# View logs
kubectl logs -f cassandra-0 -n cassandra
```

## Customization

- Modify `replicas` in StatefulSet for cluster size
- Adjust storage size in volumeClaimTemplates
- Update regions in storage classes allowedTopologies
- Configure JVM settings in cassandra-config.yaml