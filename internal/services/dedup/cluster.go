package dedup

import (
	"context"
	"sort"
)

// BKNode represents a node in the Burkhard-Keller metric tree.
type BKNode struct {
	IDs      []uint64
	Hash     uint64
	Children map[int]*BKNode
}

// BKTree is a metric tree for efficient discrete metric space searches (Hamming distance).
type BKTree struct {
	Root *BKNode
}

func NewBKTree() *BKTree {
	return &BKTree{}
}

// Insert adds an item to the BK-Tree. Handles identical hashes by grouping IDs.
func (t *BKTree) Insert(id uint64, hash uint64) {
	if t.Root == nil {
		t.Root = &BKNode{
			IDs:      []uint64{id},
			Hash:     hash,
			Children: make(map[int]*BKNode),
		}
		return
	}

	curr := t.Root
	for {
		dist := HammingDistance(curr.Hash, hash)
		if dist == 0 {
			// Exact match perceptual hash: append to current node
			curr.IDs = append(curr.IDs, id)
			return
		}

		child, exists := curr.Children[dist]
		if !exists {
			curr.Children[dist] = &BKNode{
				IDs:      []uint64{id},
				Hash:     hash,
				Children: make(map[int]*BKNode),
			}
			return
		}
		curr = child
	}
}

// Search finds all items within maxDist using triangle inequality pruning:
// |d(curr, target) - dist| <= maxDist
func (t *BKTree) Search(targetHash uint64, maxDist int, onMatch func(ids []uint64, dist int)) {
	if t.Root == nil {
		return
	}

	var traverse func(node *BKNode)
	traverse = func(node *BKNode) {
		dist := HammingDistance(node.Hash, targetHash)
		if dist <= maxDist {
			onMatch(node.IDs, dist)
		}

		minD := dist - maxDist
		if minD < 0 {
			minD = 0
		}
		maxD := dist + maxDist

		for edgeDist, child := range node.Children {
			if edgeDist >= minD && edgeDist <= maxD {
				traverse(child)
			}
		}
	}

	traverse(t.Root)
}

// ArePhotosVisuallySimilar tests whether two photos are genuinely similar under perceptual metrics.
func ArePhotosVisuallySimilar(a, b *PhotoMeta, maxDistance int) (dist int, similar bool) {
	distP := HammingDistance(a.PHash, b.PHash)
	distD := HammingDistance(a.DHash, b.DHash)

	// 1. Orientation / Aspect Ratio Check
	// If one is portrait (h > w) and one is landscape (w > h), they have inverted orientations.
	// In photo libraries, inverted orientations cannot be similar unless hash distance is virtually zero (<=3).
	if a.Width > 0 && a.Height > 0 && b.Width > 0 && b.Height > 0 {
		isPortraitA := a.Height > a.Width
		isPortraitB := b.Height > b.Width
		if isPortraitA != isPortraitB {
			if distP > 3 || distD > 3 {
				return 999, false
			}
		}
	}

	// 2. Dual-Hash Validation
	minDist := distP
	maxDist := distD
	if distD < distP {
		minDist = distD
		maxDist = distP
	}

	// If the closer metric is already worse than maxDistance, reject
	if minDist > maxDistance {
		return minDist, false
	}

	// If one metric matches, the other metric cannot be completely divergent
	// (Prevents false positives from flat backgrounds)
	allowedMax := maxDistance + 6
	if minDist <= 2 {
		allowedMax = maxDistance + 12
	}
	if maxDist > allowedMax {
		return maxDist, false
	}

	combined := int(0.6*float64(distP) + 0.4*float64(distD) + 0.5)
	if combined > maxDistance {
		// Never mask an out-of-budget pair by clamping its distance: that turns
		// "strict" mode into a much looser match and can delete false positives.
		return combined, false
	}

	return combined, true
}

// ClusterSimilarPhotos groups photos using dual BK-Tree candidate pruning and Complete-Linkage clustering.
// Every member in a cluster MUST be mutually similar to every other member in the same cluster.
func ClusterSimilarPhotos(photos []*PhotoMeta, maxDistance int) [][]*PhotoMeta {
	return ClusterSimilarPhotosContext(context.Background(), photos, maxDistance)
}

// ClusterSimilarPhotosContext is the cancellable variant used by scans. It
// returns nil when the context is canceled; callers must check ctx.Err.
func ClusterSimilarPhotosContext(ctx context.Context, photos []*PhotoMeta, maxDistance int) [][]*PhotoMeta {
	if len(photos) < 2 {
		return nil
	}

	// Sort photos by quality score descending so high-quality photos act as cluster anchors.
	// Engine currently clusters before EvaluateAndScoreGroup fills Score, so use
	// resolution/size as a stable quality proxy for that initial pass.
	sorted := make([]*PhotoMeta, len(photos))
	copy(sorted, photos)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Score != sorted[j].Score {
			return sorted[i].Score > sorted[j].Score
		}
		pixelsI := int64(sorted[i].Width) * int64(sorted[i].Height)
		pixelsJ := int64(sorted[j].Width) * int64(sorted[j].Height)
		if pixelsI != pixelsJ {
			return pixelsI > pixelsJ
		}
		if sorted[i].Size != sorted[j].Size {
			return sorted[i].Size > sorted[j].Size
		}
		return sorted[i].ID < sorted[j].ID
	})

	var clusters [][]*PhotoMeta
	// Map from photo ID to cluster index
	photoClusterMap := make(map[uint64]int)
	bkTreeP := NewBKTree()
	bkTreeD := NewBKTree()

	for _, p := range sorted {
		if err := ctx.Err(); err != nil {
			return nil
		}
		// Use dual BK-Trees (pHash & dHash) to query candidate cluster IDs
		candidateClusterMap := make(map[int]bool)
		searchThreshold := maxDistance + 2

		bkTreeP.Search(p.PHash, searchThreshold, func(ids []uint64, dist int) {
			for _, id := range ids {
				if cIdx, ok := photoClusterMap[id]; ok {
					candidateClusterMap[cIdx] = true
				}
			}
		})
		bkTreeD.Search(p.DHash, searchThreshold, func(ids []uint64, dist int) {
			for _, id := range ids {
				if cIdx, ok := photoClusterMap[id]; ok {
					candidateClusterMap[cIdx] = true
				}
			}
		})

		bestClusterIdx := -1
		bestAvgDist := 999.0

		// Check candidate clusters pruned by dual BK-Trees
		for cIdx := range candidateClusterMap {
			cluster := clusters[cIdx]
			compatible := true
			var sumDist float64
			for _, member := range cluster {
				d, ok := ArePhotosVisuallySimilar(p, member, maxDistance)
				if !ok {
					compatible = false
					break
				}
				sumDist += float64(d)
			}

			if compatible {
				avgDist := sumDist / float64(len(cluster))
				if avgDist < bestAvgDist {
					bestAvgDist = avgDist
					bestClusterIdx = cIdx
				}
			}
		}

		if bestClusterIdx >= 0 {
			clusters[bestClusterIdx] = append(clusters[bestClusterIdx], p)
			photoClusterMap[p.ID] = bestClusterIdx
		} else {
			newIdx := len(clusters)
			clusters = append(clusters, []*PhotoMeta{p})
			photoClusterMap[p.ID] = newIdx
		}

		bkTreeP.Insert(p.ID, p.PHash)
		bkTreeD.Insert(p.ID, p.DHash)
	}

	// Greedy assignment can split a valid group when a photo is closer to a
	// single-member cluster than to the group it belongs to. Merge any two
	// clusters that are fully pairwise compatible before returning results.
	clusters = mergeCompatibleClusters(ctx, clusters, maxDistance)

	// Only return clusters with at least 2 photos
	var result [][]*PhotoMeta
	for _, cluster := range clusters {
		if len(cluster) >= 2 {
			result = append(result, cluster)
		}
	}

	return result
}

// mergeCompatibleClusters merges fully pairwise-compatible clusters to repair
// greedy splits. Two singleton clusters can never merge here: if their photos
// were mutually similar, the greedy phase would already have joined the later
// photo into the earlier one's still-singleton cluster (clusters never shrink
// during assignment, and the search radius covers every mutually-similar
// pair). Skipping those pairs keeps duplicate-free libraries — where singletons
// dominate — from paying an O(C²) scan that can never merge anything.
func mergeCompatibleClusters(ctx context.Context, clusters [][]*PhotoMeta, maxDistance int) [][]*PhotoMeta {
	multiCount := 0
	for _, c := range clusters {
		if len(c) >= 2 {
			multiCount++
		}
	}
	if multiCount == 0 {
		// Only singleton×singleton pairs would remain, and those can never
		// merge; skip the scan entirely.
		return clusters
	}

	for i := 0; i < len(clusters); i++ {
		// This pass is pure CPU work; check cancellation every outer
		// iteration so a stopped scan exits immediately. Returning nil
		// matches the assignment phase's cancelled-result convention.
		if err := ctx.Err(); err != nil {
			return nil
		}
		for j := i + 1; j < len(clusters); j++ {
			if len(clusters[i]) == 0 || len(clusters[j]) == 0 {
				continue
			}
			if len(clusters[i]) == 1 && len(clusters[j]) == 1 {
				// Proven impossible to merge; see the function comment.
				continue
			}
			// If the anchors are not similar, complete linkage can never
			// merge these clusters.
			if _, ok := ArePhotosVisuallySimilar(clusters[i][0], clusters[j][0], maxDistance); !ok {
				continue
			}
			compatible := true
			for _, a := range clusters[i] {
				for _, b := range clusters[j] {
					if _, ok := ArePhotosVisuallySimilar(a, b, maxDistance); !ok {
						compatible = false
						break
					}
				}
				if !compatible {
					break
				}
			}
			if !compatible {
				continue
			}
			clusters[i] = append(clusters[i], clusters[j]...)
			clusters = append(clusters[:j], clusters[j+1:]...)
			// Restart because the merged cluster may now be compatible with
			// another cluster earlier in the slice.
			i = -1
			break
		}
	}
	return clusters
}

// ClusterExactPhotos groups photos with identical full content hashes.
func ClusterExactPhotos(photos []*PhotoMeta) [][]*PhotoMeta {
	hashMap := make(map[uint64][]*PhotoMeta)
	for _, p := range photos {
		if p.ContentHash != 0 {
			hashMap[p.ContentHash] = append(hashMap[p.ContentHash], p)
		}
	}

	var result [][]*PhotoMeta
	for _, grp := range hashMap {
		if len(grp) >= 2 {
			result = append(result, grp)
		}
	}
	return result
}
