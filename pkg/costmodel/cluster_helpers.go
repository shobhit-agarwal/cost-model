package costmodel

import (
	"strconv"
	"time"

	"github.com/kubecost/cost-model/pkg/cloud"

	"github.com/kubecost/cost-model/pkg/env"
	"github.com/kubecost/cost-model/pkg/log"
	"github.com/kubecost/cost-model/pkg/prom"
)

// mergeTypeMaps takes two maps of (cluster name, node name) -> node type
// and combines them into a single map, preferring the k/v pairs in
// the first map.
func mergeTypeMaps(clusterAndNameToType1, clusterAndNameToType2 map[nodeIdentifierNoProviderID]string) map[nodeIdentifierNoProviderID]string {
	merged := map[nodeIdentifierNoProviderID]string{}
	for k, v := range clusterAndNameToType2 {
		merged[k] = v
	}

	// This ordering ensures the mappings in the first arg are preferred.
	for k, v := range clusterAndNameToType1 {
		merged[k] = v
	}

	return merged
}

func buildCPUCostMap(
	resNodeCPUCost []*prom.QueryResult,
	cp cloud.Provider,
	preemptible map[NodeIdentifier]bool,
) (
	map[NodeIdentifier]float64,
	map[nodeIdentifierNoProviderID]string,
) {

	cpuCostMap := make(map[NodeIdentifier]float64)
	clusterAndNameToType := make(map[nodeIdentifierNoProviderID]string)

	customPricingEnabled := cloud.CustomPricesEnabled(cp)
	customPricingConfig, err := cp.GetConfig()
	if err != nil {
		log.Warningf("ClusterNodes: failed to load custom pricing: %s", err)
	}

	for _, result := range resNodeCPUCost {
		cluster, err := result.GetString(env.GetPromClusterLabel())
		if err != nil {
			cluster = env.GetClusterID()
		}

		name, err := result.GetString("node")
		if err != nil {
			log.Warningf("ClusterNodes: CPU cost data missing node")
			continue
		}

		nodeType, _ := result.GetString("instance_type")
		providerID, _ := result.GetString("provider_id")

		key := NodeIdentifier{
			Cluster:    cluster,
			Name:       name,
			ProviderID: cloud.ParseID(providerID),
		}
		keyNon := nodeIdentifierNoProviderID{
			Cluster: cluster,
			Name:    name,
		}

		var cpuCost float64

		if customPricingEnabled && customPricingConfig != nil {

			var customCPUStr string
			if spot, ok := preemptible[key]; ok && spot {
				customCPUStr = customPricingConfig.SpotCPU
			} else {
				customCPUStr = customPricingConfig.CPU
			}

			customCPUCost, err := strconv.ParseFloat(customCPUStr, 64)
			if err != nil {
				log.Warningf("ClusterNodes: error parsing custom CPU price: %s", customCPUStr)
			}
			cpuCost = customCPUCost

		} else {

			cpuCost = result.Values[0].Value

		}

		clusterAndNameToType[keyNon] = nodeType

		cpuCostMap[key] = cpuCost
	}

	return cpuCostMap, clusterAndNameToType
}

func buildRAMCostMap(
	resNodeRAMCost []*prom.QueryResult,
	cp cloud.Provider,
	preemptible map[NodeIdentifier]bool,
) (
	map[NodeIdentifier]float64,
	map[nodeIdentifierNoProviderID]string,
) {

	ramCostMap := make(map[NodeIdentifier]float64)
	clusterAndNameToType := make(map[nodeIdentifierNoProviderID]string)

	customPricingEnabled := cloud.CustomPricesEnabled(cp)
	customPricingConfig, err := cp.GetConfig()
	if err != nil {
		log.Warningf("ClusterNodes: failed to load custom pricing: %s", err)
	}

	for _, result := range resNodeRAMCost {
		cluster, err := result.GetString(env.GetPromClusterLabel())
		if err != nil {
			cluster = env.GetClusterID()
		}

		name, err := result.GetString("node")
		if err != nil {
			log.Warningf("ClusterNodes: RAM cost data missing node")
			continue
		}

		nodeType, _ := result.GetString("instance_type")
		providerID, _ := result.GetString("provider_id")

		key := NodeIdentifier{
			Cluster:    cluster,
			Name:       name,
			ProviderID: cloud.ParseID(providerID),
		}
		keyNon := nodeIdentifierNoProviderID{
			Cluster: cluster,
			Name:    name,
		}

		var ramCost float64

		if customPricingEnabled && customPricingConfig != nil {

			var customRAMStr string
			if spot, ok := preemptible[key]; ok && spot {
				customRAMStr = customPricingConfig.SpotRAM
			} else {
				customRAMStr = customPricingConfig.RAM
			}

			customRAMCost, err := strconv.ParseFloat(customRAMStr, 64)
			if err != nil {
				log.Warningf("ClusterNodes: error parsing custom RAM price: %s", customRAMStr)
			}
			ramCost = customRAMCost / 1024 / 1024 / 1024

		} else {

			ramCost = result.Values[0].Value

		}

		clusterAndNameToType[keyNon] = nodeType

		ramCostMap[key] = ramCost
	}

	return ramCostMap, clusterAndNameToType
}

func buildGPUCostMap(
	resNodeGPUCost []*prom.QueryResult,
	gpuCountMap map[NodeIdentifier]float64,
	cp cloud.Provider,
	preemptible map[NodeIdentifier]bool,
) (
	map[NodeIdentifier]float64,
	map[nodeIdentifierNoProviderID]string,
) {

	gpuCostMap := make(map[NodeIdentifier]float64)
	clusterAndNameToType := make(map[nodeIdentifierNoProviderID]string)

	customPricingEnabled := cloud.CustomPricesEnabled(cp)
	customPricingConfig, err := cp.GetConfig()
	if err != nil {
		log.Warningf("ClusterNodes: failed to load custom pricing: %s", err)
	}

	for _, result := range resNodeGPUCost {
		cluster, err := result.GetString(env.GetPromClusterLabel())
		if err != nil {
			cluster = env.GetClusterID()
		}

		name, err := result.GetString("node")
		if err != nil {
			log.Warningf("ClusterNodes: GPU cost data missing node")
			continue
		}

		nodeType, _ := result.GetString("instance_type")
		providerID, _ := result.GetString("provider_id")

		key := NodeIdentifier{
			Cluster:    cluster,
			Name:       name,
			ProviderID: cloud.ParseID(providerID),
		}
		keyNon := nodeIdentifierNoProviderID{
			Cluster: cluster,
			Name:    name,
		}

		var gpuCost float64

		if customPricingEnabled && customPricingConfig != nil {

			var customGPUStr string
			if spot, ok := preemptible[key]; ok && spot {
				customGPUStr = customPricingConfig.SpotGPU
			} else {
				customGPUStr = customPricingConfig.GPU
			}

			customGPUCost, err := strconv.ParseFloat(customGPUStr, 64)
			if err != nil {
				log.Warningf("ClusterNodes: error parsing custom GPU price: %s", customGPUStr)
			}
			gpuCost = customGPUCost

		} else {

			gpuCost = result.Values[0].Value

		}

		clusterAndNameToType[keyNon] = nodeType

		// If gpu count is available use it to multiply gpu cost
		if value, ok := gpuCountMap[key]; ok && value != 0 {
			gpuCostMap[key] = gpuCost * value
		} else {
			gpuCostMap[key] = gpuCost
		}

	}

	return gpuCostMap, clusterAndNameToType
}

func buildGPUCountMap(
	resNodeGPUCount []*prom.QueryResult,
) map[NodeIdentifier]float64 {

	gpuCountMap := make(map[NodeIdentifier]float64)

	for _, result := range resNodeGPUCount {
		cluster, err := result.GetString(env.GetPromClusterLabel())
		if err != nil {
			cluster = env.GetClusterID()
		}

		name, err := result.GetString("node")
		if err != nil {
			log.Warningf("ClusterNodes: GPU count data missing node")
			continue
		}

		gpuCount := result.Values[0].Value
		providerID, _ := result.GetString("provider_id")

		key := NodeIdentifier{
			Cluster:    cluster,
			Name:       name,
			ProviderID: cloud.ParseID(providerID),
		}
		gpuCountMap[key] = gpuCount
	}

	return gpuCountMap
}

func buildCPUCoresMap(
	resNodeCPUCores []*prom.QueryResult,
) map[nodeIdentifierNoProviderID]float64 {

	m := make(map[nodeIdentifierNoProviderID]float64)

	for _, result := range resNodeCPUCores {
		cluster, err := result.GetString(env.GetPromClusterLabel())
		if err != nil {
			cluster = env.GetClusterID()
		}

		name, err := result.GetString("node")
		if err != nil {
			log.Warningf("ClusterNodes: CPU cores data missing node")
			continue
		}

		cpuCores := result.Values[0].Value

		key := nodeIdentifierNoProviderID{
			Cluster: cluster,
			Name:    name,
		}
		m[key] = cpuCores
	}

	return m
}

func buildRAMBytesMap(resNodeRAMBytes []*prom.QueryResult) map[nodeIdentifierNoProviderID]float64 {

	m := make(map[nodeIdentifierNoProviderID]float64)

	for _, result := range resNodeRAMBytes {
		cluster, err := result.GetString(env.GetPromClusterLabel())
		if err != nil {
			cluster = env.GetClusterID()
		}

		name, err := result.GetString("node")
		if err != nil {
			log.Warningf("ClusterNodes: RAM bytes data missing node")
			continue
		}

		ramBytes := result.Values[0].Value

		key := nodeIdentifierNoProviderID{
			Cluster: cluster,
			Name:    name,
		}
		m[key] = ramBytes
	}

	return m
}

// Mapping of cluster/node=cpu for computing resource efficiency
func buildCPUBreakdownMap(resNodeCPUModeTotal []*prom.QueryResult) map[nodeIdentifierNoProviderID]*ClusterCostsBreakdown {

	cpuBreakdownMap := make(map[nodeIdentifierNoProviderID]*ClusterCostsBreakdown)

	// Mapping of cluster/node=cpu for computing resource efficiency
	clusterNodeCPUTotal := map[nodeIdentifierNoProviderID]float64{}
	// Mapping of cluster/node:mode=cpu for computing resource efficiency
	clusterNodeModeCPUTotal := map[nodeIdentifierNoProviderID]map[string]float64{}

	// Build intermediate structures for CPU usage by (cluster, node) and by
	// (cluster, node, mode) for computing resouce efficiency
	for _, result := range resNodeCPUModeTotal {
		cluster, err := result.GetString(env.GetPromClusterLabel())
		if err != nil {
			cluster = env.GetClusterID()
		}

		node, err := result.GetString("kubernetes_node")
		if err != nil {
			log.DedupedWarningf(5, "ClusterNodes: CPU mode data missing node")
			continue
		}

		mode, err := result.GetString("mode")
		if err != nil {
			log.Warningf("ClusterNodes: unable to read CPU mode: %s", err)
			mode = "other"
		}

		key := nodeIdentifierNoProviderID{
			Cluster: cluster,
			Name:    node,
		}

		total := result.Values[0].Value

		// Increment total
		clusterNodeCPUTotal[key] += total

		// Increment mode
		if _, ok := clusterNodeModeCPUTotal[key]; !ok {
			clusterNodeModeCPUTotal[key] = map[string]float64{}
		}
		clusterNodeModeCPUTotal[key][mode] += total
	}

	// Compute resource efficiency from intermediate structures
	for key, total := range clusterNodeCPUTotal {
		if modeTotals, ok := clusterNodeModeCPUTotal[key]; ok {
			for mode, subtotal := range modeTotals {
				// Compute percentage for the current cluster, node, mode
				pct := 0.0
				if total > 0 {
					pct = subtotal / total
				}

				if _, ok := cpuBreakdownMap[key]; !ok {
					cpuBreakdownMap[key] = &ClusterCostsBreakdown{}
				}

				switch mode {
				case "idle":
					cpuBreakdownMap[key].Idle += pct
				case "system":
					cpuBreakdownMap[key].System += pct
				case "user":
					cpuBreakdownMap[key].User += pct
				default:
					cpuBreakdownMap[key].Other += pct
				}
			}
		}
	}

	return cpuBreakdownMap
}

func buildRAMUserPctMap(resNodeRAMUserPct []*prom.QueryResult) map[nodeIdentifierNoProviderID]float64 {

	m := make(map[nodeIdentifierNoProviderID]float64)

	for _, result := range resNodeRAMUserPct {
		cluster, err := result.GetString(env.GetPromClusterLabel())
		if err != nil {
			cluster = env.GetClusterID()
		}

		name, err := result.GetString("instance")
		if err != nil {
			log.Warningf("ClusterNodes: RAM user percent missing node")
			continue
		}

		pct := result.Values[0].Value

		key := nodeIdentifierNoProviderID{
			Cluster: cluster,
			Name:    name,
		}

		m[key] = pct
	}

	return m
}

func buildRAMSystemPctMap(resNodeRAMSystemPct []*prom.QueryResult) map[nodeIdentifierNoProviderID]float64 {

	m := make(map[nodeIdentifierNoProviderID]float64)

	for _, result := range resNodeRAMSystemPct {
		cluster, err := result.GetString(env.GetPromClusterLabel())
		if err != nil {
			cluster = env.GetClusterID()
		}

		name, err := result.GetString("instance")
		if err != nil {
			log.Warningf("ClusterNodes: RAM system percent missing node")
			continue
		}

		pct := result.Values[0].Value

		key := nodeIdentifierNoProviderID{
			Cluster: cluster,
			Name:    name,
		}

		m[key] = pct
	}

	return m
}

type activeData struct {
	start   time.Time
	end     time.Time
	minutes float64
}

func buildActiveDataMap(resActiveMins []*prom.QueryResult, resolution time.Duration) map[NodeIdentifier]activeData {

	m := make(map[NodeIdentifier]activeData)

	for _, result := range resActiveMins {
		cluster, err := result.GetString(env.GetPromClusterLabel())
		if err != nil {
			cluster = env.GetClusterID()
		}

		name, err := result.GetString("node")
		if err != nil {
			log.Warningf("ClusterNodes: active mins missing node")
			continue
		}

		providerID, _ := result.GetString("provider_id")

		key := NodeIdentifier{
			Cluster:    cluster,
			Name:       name,
			ProviderID: cloud.ParseID(providerID),
		}

		if len(result.Values) == 0 {
			continue
		}

		s := time.Unix(int64(result.Values[0].Timestamp), 0)
		e := time.Unix(int64(result.Values[len(result.Values)-1].Timestamp), 0).Add(resolution)
		mins := e.Sub(s).Minutes()

		// TODO niko/assets if mins >= threshold, interpolate for missing data?
		m[key] = activeData{
			start:   s,
			end:     e,
			minutes: mins,
		}
	}

	return m
}

// Determine preemptibility with node labels
// node id -> is preemptible?
func buildPreemptibleMap(
	resIsSpot []*prom.QueryResult,
) map[NodeIdentifier]bool {

	m := make(map[NodeIdentifier]bool)

	for _, result := range resIsSpot {
		nodeName, err := result.GetString("node")
		if err != nil {
			continue
		}

		// GCP preemptible label
		pre := result.Values[0].Value

		cluster, err := result.GetString(env.GetPromClusterLabel())
		if err != nil {
			cluster = env.GetClusterID()
		}

		providerID, _ := result.GetString("provider_id")

		key := NodeIdentifier{
			Cluster:    cluster,
			Name:       nodeName,
			ProviderID: cloud.ParseID(providerID),
		}

		// TODO(michaelmdresser): check this condition at merge time?
		// if node, ok := nodeMap[key]; pre > 0.0 && ok {
		// 	node.Preemptible = true
		// }
		m[key] = pre > 0.0

		// TODO AWS preemptible

		// TODO Azure preemptible
	}

	return m
}

func buildLabelsMap(
	resLabels []*prom.QueryResult,
) map[nodeIdentifierNoProviderID]map[string]string {

	m := make(map[nodeIdentifierNoProviderID]map[string]string)

	// Copy labels into node
	for _, result := range resLabels {
		cluster, err := result.GetString(env.GetPromClusterLabel())
		if err != nil {
			cluster = env.GetClusterID()
		}
		node, err := result.GetString("node")
		if err != nil {
			log.DedupedWarningf(5, "ClusterNodes: label data missing node")
			continue
		}
		key := nodeIdentifierNoProviderID{
			Cluster: cluster,
			Name:    node,
		}

		m[key] = make(map[string]string)

		for name, value := range result.Metric {
			if val, ok := value.(string); ok {
				m[key][name] = val
			}
		}
	}
	return m
}

// checkForKeyAndInitIfMissing inits a key in the provided nodemap if
// it does not exist. Intended to be called ONLY by buildNodeMap
func checkForKeyAndInitIfMissing(
	nodeMap map[NodeIdentifier]*Node,
	key NodeIdentifier,
	clusterAndNameToType map[nodeIdentifierNoProviderID]string,
) {
	if _, ok := nodeMap[key]; !ok {
		// default nodeType in case we don't have the mapping
		var nodeType string
		if t, ok := clusterAndNameToType[nodeIdentifierNoProviderID{
			Cluster: key.Cluster,
			Name:    key.Name,
		}]; ok {
			nodeType = t
		} else {
			log.Warningf("ClusterNodes: Type does not exist for node identifier %s", key)
		}

		nodeMap[key] = &Node{
			Cluster:      key.Cluster,
			Name:         key.Name,
			NodeType:     nodeType,
			ProviderID:   key.ProviderID,
			CPUBreakdown: &ClusterCostsBreakdown{},
			RAMBreakdown: &ClusterCostsBreakdown{},
		}
	}
}

// buildNodeMap creates the main set of node data for ClusterNodes from
// the data maps built from Prometheus queries. Some of the Prometheus
// data has access to the provider_id field and some does not. To get
// around this problem, we use the data that includes provider_id
// to build up the definitive set of nodes and then use the data
// with less-specific identifiers (i.e. without provider_id) to fill
// in the remaining fields.
//
// For example, let's say we have nodes identified like so:
// cluster name/node name/provider_id. For the sake of the example,
// we will also limit data to CPU cost, CPU cores, and preemptibility.
//
// We have CPU cost data that looks like this:
// cluster1/node1/prov_node1_A: $10
// cluster1/node1/prov_node1_B: $8
// cluster1/node2/prov_node2: $15
//
// We have Preemptible data that looks like this:
// cluster1/node1/prov_node1_A: true
// cluster1/node1/prov_node1_B: false
// cluster1/node2/prov_node2_B: false
//
// We have CPU cores data that looks like this:
// cluster1/node1: 4
// cluster1/node2: 6
//
// This function first combines the data that is fully identified,
// creating the following:
// cluster1/node1/prov_node1_A: CPUCost($10), Preemptible(true)
// cluster1/node1/prov_node1_B: CPUCost($8), Preemptible(false)
// cluster1/node2/prov_node2: CPUCost($15), Preemptible(false)
//
// It then uses the less-specific data to extend the specific data,
// making the following:
// cluster1/node1/prov_node1_A: CPUCost($10), Preemptible(true), Cores(4)
// cluster1/node1/prov_node1_B: CPUCost($8), Preemptible(false), Cores(4)
// cluster1/node2/prov_node2: CPUCost($15), Preemptible(false), Cores(6)
//
// In the situation where provider_id doesn't exist for any metrics,
// that is the same as all provider_ids being empty strings. If
// provider_id doesn't exist at all, then we (without having to do
// extra work) easily fall back on identifying nodes only by cluster name
// and node name because the provider_id part of the key will always
// be the empty string.
//
// It is worth nothing that, in this approach, if a node is not present
// in the more specific data but is present in the less-specific data,
// that data is never processed into the final node map. For example,
// let's say the CPU cores map has the following entry:
// cluster1/node8: 6
// But none of the maps with provider_id (CPU cost, RAM cost, etc.)
// have an identifier for cluster1/node8 (regardless of provider_id).
// In this situation, the final node map will not have a cluster1/node8
// entry. This could be fixed by iterating over all of the less specific
// identifiers and, inside that iteration, all of the identifiers in
// the node map, but this would introduce a roughly quadratic time
// complexity.
func buildNodeMap(
	cpuCostMap, ramCostMap, gpuCostMap, gpuCountMap map[NodeIdentifier]float64,
	cpuCoresMap, ramBytesMap, ramUserPctMap,
	ramSystemPctMap map[nodeIdentifierNoProviderID]float64,
	cpuBreakdownMap map[nodeIdentifierNoProviderID]*ClusterCostsBreakdown,
	activeDataMap map[NodeIdentifier]activeData,
	preemptibleMap map[NodeIdentifier]bool,
	labelsMap map[nodeIdentifierNoProviderID]map[string]string,
	clusterAndNameToType map[nodeIdentifierNoProviderID]string,
) map[NodeIdentifier]*Node {

	nodeMap := make(map[NodeIdentifier]*Node)

	// Initialize the map with the most-specific data:

	for id, cost := range cpuCostMap {
		checkForKeyAndInitIfMissing(nodeMap, id, clusterAndNameToType)
		nodeMap[id].CPUCost = cost
	}

	for id, cost := range ramCostMap {
		checkForKeyAndInitIfMissing(nodeMap, id, clusterAndNameToType)
		nodeMap[id].RAMCost = cost
	}

	for id, cost := range gpuCostMap {
		checkForKeyAndInitIfMissing(nodeMap, id, clusterAndNameToType)
		nodeMap[id].GPUCost = cost
	}

	for id, count := range gpuCountMap {
		checkForKeyAndInitIfMissing(nodeMap, id, clusterAndNameToType)
		nodeMap[id].GPUCount = count
	}

	for id, preemptible := range preemptibleMap {
		checkForKeyAndInitIfMissing(nodeMap, id, clusterAndNameToType)
		nodeMap[id].Preemptible = preemptible
	}

	for id, activeData := range activeDataMap {
		checkForKeyAndInitIfMissing(nodeMap, id, clusterAndNameToType)
		nodeMap[id].Start = activeData.start
		nodeMap[id].End = activeData.end
		nodeMap[id].Minutes = activeData.minutes
	}

	// We now merge in data that doesn't have a provider id by looping over
	// all keys already added and inserting data according to their
	// cluster name/node name combos.
	for id, nodePtr := range nodeMap {
		clusterAndNameID := nodeIdentifierNoProviderID{
			Cluster: id.Cluster,
			Name:    id.Name,
		}

		if cores, ok := cpuCoresMap[clusterAndNameID]; ok {
			nodePtr.CPUCores = cores
			if v, ok := partialCPUMap[nodePtr.NodeType]; ok {
				if cores > 0 {
					nodePtr.CPUCores = v
					adjustmentFactor := v / cores
					nodePtr.CPUCost = nodePtr.CPUCost * adjustmentFactor
				}
			}
		}

		if ramBytes, ok := ramBytesMap[clusterAndNameID]; ok {
			nodePtr.RAMBytes = ramBytes
		}

		if ramUserPct, ok := ramUserPctMap[clusterAndNameID]; ok {
			nodePtr.RAMBreakdown.User = ramUserPct
		}

		if ramSystemPct, ok := ramSystemPctMap[clusterAndNameID]; ok {
			nodePtr.RAMBreakdown.System = ramSystemPct
		}

		if cpuBreakdown, ok := cpuBreakdownMap[clusterAndNameID]; ok {
			nodePtr.CPUBreakdown = cpuBreakdown
		}

		if labels, ok := labelsMap[clusterAndNameID]; ok {
			nodePtr.Labels = labels
		}
	}

	return nodeMap
}
